package risk

import (
    "context"
    "database/sql"
    "testing"
    "time"

    "github.com/DATA-DOG/go-sqlmock"
    "github.com/stretchr/testify/assert"
    "github.com/Cryptoprojectsfun/quantai-clone/internal/models"
)

func TestRiskManager_AnalyzeRisk(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("Failed to create mock DB: %v", err)
    }
    defer db.Close()

    manager := NewRiskManager(db)
    ctx := context.Background()

    t.Run("Calculate risk metrics for portfolio", func(t *testing.T) {
        portfolioID := int64(1)

        // Mock positions query
        positionRows := sqlmock.NewRows([]string{"id", "portfolio_id", "symbol", "quantity", "entry_price"}).
            AddRow(1, portfolioID, "AAPL", 100.0, 150.0).
            AddRow(2, portfolioID, "GOOGL", 50.0, 2800.0)

        mock.ExpectQuery("SELECT (.+) FROM positions WHERE portfolio_id = ?").
            WithArgs(portfolioID).
            WillReturnRows(positionRows)

        // Mock historical returns for VaR calculation
        returnsRows := sqlmock.NewRows([]string{"symbol", "var_return"}).
            AddRow("AAPL", -0.02).
            AddRow("GOOGL", -0.025)

        mock.ExpectQuery("WITH position_returns").
            WithArgs([]int64{1, 2}, 0.05).
            WillReturnRows(returnsRows)

        // Mock drawdown calculation
        drawdownRows := sqlmock.NewRows([]string{"drawdown"}).
            AddRow(0.15).
            AddRow(0.12)

        mock.ExpectQuery("SELECT (.+) FROM market_data WHERE symbol = (.+)").
            WithArgs("AAPL").
            WillReturnRows(drawdownRows)

        mock.ExpectQuery("SELECT (.+) FROM market_data WHERE symbol = (.+)").
            WithArgs("GOOGL").
            WillReturnRows(drawdownRows)

        // Mock volatility calculation
        volRows := sqlmock.NewRows([]string{"symbol", "volatility"}).
            AddRow("AAPL", 0.2).
            AddRow("GOOGL", 0.25)

        mock.ExpectQuery("WITH daily_returns").
            WithArgs([]string{"AAPL", "GOOGL"}).
            WillReturnRows(volRows)

        metrics, err := manager.AnalyzeRisk(ctx, portfolioID)
        assert.NoError(t, err)
        assert.NotNil(t, metrics)

        // Verify risk metrics
        assert.InDelta(t, 0.022, metrics.ValueAtRisk, 0.001)
        assert.InDelta(t, 0.14, metrics.Drawdown, 0.01)
        assert.InDelta(t, 0.22, metrics.Volatility, 0.01)
        assert.Equal(t, "YELLOW", metrics.AlertLevel)
        assert.Len(t, metrics.Alerts, 1)
    })

    t.Run("Handle empty portfolio", func(t *testing.T) {
        portfolioID := int64(2)

        mock.ExpectQuery("SELECT (.+) FROM positions WHERE portfolio_id = ?").
            WithArgs(portfolioID).
            WillReturnRows(sqlmock.NewRows([]string{"id", "portfolio_id", "symbol", "quantity", "entry_price"}))

        metrics, err := manager.AnalyzeRisk(ctx, portfolioID)
        assert.NoError(t, err)
        assert.NotNil(t, metrics)
        assert.Equal(t, 0.0, metrics.ValueAtRisk)
        assert.Equal(t, 0.0, metrics.Drawdown)
        assert.Equal(t, 0.0, metrics.Volatility)
        assert.Equal(t, "GREEN", metrics.AlertLevel)
        assert.Len(t, metrics.Alerts, 0)
    })

    t.Run("Handle database errors", func(t *testing.T) {
        portfolioID := int64(3)

        mock.ExpectQuery("SELECT (.+) FROM positions WHERE portfolio_id = ?").
            WithArgs(portfolioID).
            WillReturnError(sql.ErrConnDone)

        metrics, err := manager.AnalyzeRisk(ctx, portfolioID)
        assert.Error(t, err)
        assert.Nil(t, metrics)
        assert.Equal(t, sql.ErrConnDone, err)
    })
}

func TestRiskManager_GenerateAlerts(t *testing.T) {
    manager := NewRiskManager(nil)

    tests := []struct {
        name          string
        var_         float64
        drawdown     float64
        concentration float64
        volatility   float64
        wantLevel    string
        wantAlerts   int
    }{
        {
            name:          "No alerts",
            var_:         0.05,
            drawdown:     0.10,
            concentration: 0.20,
            volatility:   0.015,
            wantLevel:    "GREEN",
            wantAlerts:   0,
        },
        {
            name:          "High VaR alert",
            var_:         0.20,
            drawdown:     0.10,
            concentration: 0.20,
            volatility:   0.015,
            wantLevel:    "RED",
            wantAlerts:   1,
        },
        {
            name:          "Multiple alerts",
            var_:         0.20,
            drawdown:     0.18,
            concentration: 0.35,
            volatility:   0.025,
            wantLevel:    "RED",
            wantAlerts:   4,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            alerts := manager.generateAlerts(tt.var_, tt.drawdown, tt.concentration, tt.volatility)
            level := manager.determineAlertLevel(alerts)

            assert.Equal(t, tt.wantLevel, level)
            assert.Len(t, alerts, tt.wantAlerts)
        })
    }
}
