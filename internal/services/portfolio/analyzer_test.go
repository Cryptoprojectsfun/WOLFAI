package portfolio

import (
    "context"
    "database/sql"
    "testing"
    "time"

    "github.com/DATA-DOG/go-sqlmock"
    "github.com/stretchr/testify/assert"
    "github.com/Cryptoprojectsfun/quantai-clone/internal/models"
)

func TestPortfolioAnalyzer_AnalyzePortfolio(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("Failed to create mock DB: %v", err)
    }
    defer db.Close()

    analyzer := NewPortfolioAnalyzer(db)
    ctx := context.Background()

    t.Run("Calculate metrics for valid portfolio", func(t *testing.T) {
        portfolioID := int64(1)
        
        // Mock positions query
        positionRows := sqlmock.NewRows([]string{"id", "portfolio_id", "symbol", "quantity", "entry_price"}).
            AddRow(1, portfolioID, "AAPL", 10.0, 150.0).
            AddRow(2, portfolioID, "GOOGL", 5.0, 2800.0)

        mock.ExpectQuery("SELECT (.+) FROM positions WHERE portfolio_id = ?").
            WithArgs(portfolioID).
            WillReturnRows(positionRows)

        // Mock market data queries for current prices
        priceRows := sqlmock.NewRows([]string{"close"}).AddRow(160.0)
        mock.ExpectQuery("SELECT close FROM market_data WHERE symbol = (.+) ORDER BY timestamp DESC LIMIT 1").
            WithArgs("AAPL").
            WillReturnRows(priceRows)

        priceRows = sqlmock.NewRows([]string{"close"}).AddRow(2900.0)
        mock.ExpectQuery("SELECT close FROM market_data WHERE symbol = (.+) ORDER BY timestamp DESC LIMIT 1").
            WithArgs("GOOGL").
            WillReturnRows(priceRows)

        // Mock historical data for volatility calculation
        historicalRows := sqlmock.NewRows([]string{"close"}).
            AddRow(155.0).
            AddRow(158.0).
            AddRow(157.0).
            AddRow(160.0)

        mock.ExpectQuery("SELECT close FROM market_data WHERE symbol = (.+) AND timestamp >= (.+)").
            WithArgs("AAPL").
            WillReturnRows(historicalRows)

        historicalRows = sqlmock.NewRows([]string{"close"}).
            AddRow(2850.0).
            AddRow(2880.0).
            AddRow(2870.0).
            AddRow(2900.0)

        mock.ExpectQuery("SELECT close FROM market_data WHERE symbol = (.+) AND timestamp >= (.+)").
            WithArgs("GOOGL").
            WillReturnRows(historicalRows)

        metrics, err := analyzer.AnalyzePortfolio(ctx, portfolioID)
        assert.NoError(t, err)
        assert.NotNil(t, metrics)

        // Verify calculated metrics
        assert.InDelta(t, 16000.0, metrics.TotalValue, 0.01)  // (10 * 160) + (5 * 2900)
        assert.InDelta(t, 600.0, metrics.PnL, 0.01)          // ((160-150)*10 + (2900-2800)*5)
        assert.InDelta(t, 3.89, metrics.PnLPercentage, 0.01) // (600 / 15500) * 100
        assert.InDelta(t, 0.015, metrics.Volatility, 0.01)   // Based on historical prices
        assert.InDelta(t, 0.8, metrics.SharpeRatio, 0.1)     // (3.89 - 2) / (0.015 * sqrt(252))
    })

    t.Run("Handle empty portfolio", func(t *testing.T) {
        portfolioID := int64(2)
        
        mock.ExpectQuery("SELECT (.+) FROM positions WHERE portfolio_id = ?").
            WithArgs(portfolioID).
            WillReturnRows(sqlmock.NewRows([]string{"id", "portfolio_id", "symbol", "quantity", "entry_price"}))

        metrics, err := analyzer.AnalyzePortfolio(ctx, portfolioID)
        assert.NoError(t, err)
        assert.NotNil(t, metrics)
        assert.Equal(t, 0.0, metrics.TotalValue)
        assert.Equal(t, 0.0, metrics.PnL)
        assert.Equal(t, 0.0, metrics.Volatility)
    })

    t.Run("Handle database errors", func(t *testing.T) {
        portfolioID := int64(3)
        
        mock.ExpectQuery("SELECT (.+) FROM positions WHERE portfolio_id = ?").
            WithArgs(portfolioID).
            WillReturnError(sql.ErrConnDone)

        metrics, err := analyzer.AnalyzePortfolio(ctx, portfolioID)
        assert.Error(t, err)
        assert.Nil(t, metrics)
        assert.Equal(t, sql.ErrConnDone, err)
    })
}

func TestPortfolioAnalyzer_CalculateVolatility(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("Failed to create mock DB: %v", err)
    }
    defer db.Close()

    analyzer := NewPortfolioAnalyzer(db)
    ctx := context.Background()

    t.Run("Calculate volatility for single position", func(t *testing.T) {
        positions := []models.Position{
            {
                Symbol:     "AAPL",
                Quantity:   10,
                EntryPrice: 150,
            },
        }

        historicalRows := sqlmock.NewRows([]string{"close"}).
            AddRow(150.0).
            AddRow(152.0).
            AddRow(151.0).
            AddRow(153.0).
            AddRow(152.0)

        mock.ExpectQuery("SELECT close FROM market_data WHERE symbol = (.+) AND timestamp >= (.+)").
            WithArgs("AAPL").
            WillReturnRows(historicalRows)

        volatility, err := analyzer.calculateVolatility(ctx, positions)
        assert.NoError(t, err)
        assert.InDelta(t, 0.01, volatility, 0.005)
    })

    t.Run("Handle insufficient data points", func(t *testing.T) {
        positions := []models.Position{
            {
                Symbol:     "AAPL",
                Quantity:   10,
                EntryPrice: 150,
            },
        }

        historicalRows := sqlmock.NewRows([]string{"close"}).
            AddRow(150.0).
            AddRow(152.0)

        mock.ExpectQuery("SELECT close FROM market_data WHERE symbol = (.+) AND timestamp >= (.+)").
            WithArgs("AAPL").
            WillReturnRows(historicalRows)

        volatility, err := analyzer.calculateVolatility(ctx, positions)
        assert.Error(t, err)
        assert.Equal(t, 0.0, volatility)
    })
}
