package risk

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/Cryptoprojectsfun/quantai-clone/internal/models"
)

type RiskManager struct {
    db *sql.DB
    // Risk thresholds
    maxDrawdown     float64
    maxConcentration float64
    varConfidence   float64
    varDays         int
}

type RiskMetrics struct {
    ValueAtRisk    float64   `json:"var"`          // Value at Risk
    Drawdown       float64   `json:"drawdown"`     // Current drawdown
    Concentration  float64   `json:"concentration"` // Highest single asset concentration
    Volatility    float64   `json:"volatility"`   // Portfolio volatility
    AlertLevel    string    `json:"alert_level"`  // GREEN, YELLOW, RED
    Alerts        []Alert   `json:"alerts"`       // Active risk alerts
}

type Alert struct {
    Type        string    `json:"type"`
    Message     string    `json:"message"`
    Severity    string    `json:"severity"`
    Timestamp   time.Time `json:"timestamp"`
}

func NewRiskManager(db *sql.DB) *RiskManager {
    return &RiskManager{
        db:              db,
        maxDrawdown:     0.15,  // 15% maximum drawdown
        maxConcentration: 0.30,  // 30% maximum in single asset
        varConfidence:   0.95,  // 95% VaR confidence
        varDays:         10,    // 10-day VaR
    }
}

func (rm *RiskManager) AnalyzeRisk(ctx context.Context, portfolioID int64) (*RiskMetrics, error) {
    // Get portfolio positions
    positions, err := rm.getPositions(ctx, portfolioID)
    if err != nil {
        return nil, err
    }

    // Calculate metrics
    valueAtRisk, err := rm.calculateVaR(ctx, positions)
    if err != nil {
        return nil, err
    }

    drawdown, err := rm.calculateDrawdown(ctx, positions)
    if err != nil {
        return nil, err
    }

    concentration, err := rm.calculateConcentration(ctx, positions)
    if err != nil {
        return nil, err
    }

    volatility, err := rm.calculateVolatility(ctx, positions)
    if err != nil {
        return nil, err
    }

    // Generate alerts
    alerts := rm.generateAlerts(valueAtRisk, drawdown, concentration, volatility)
    alertLevel := rm.determineAlertLevel(alerts)

    return &RiskMetrics{
        ValueAtRisk:   valueAtRisk,
        Drawdown:      drawdown,
        Concentration: concentration,
        Volatility:    volatility,
        AlertLevel:    alertLevel,
        Alerts:        alerts,
    }, nil
}

func (rm *RiskManager) calculateVaR(ctx context.Context, positions []models.Position) (float64, error) {
    // Historical VaR calculation
    query := `
        WITH position_returns AS (
            SELECT 
                p.symbol,
                p.quantity,
                p.entry_price,
                (m.close - LAG(m.close) OVER (PARTITION BY p.symbol ORDER BY m.timestamp)) / LAG(m.close) OVER (PARTITION BY p.symbol ORDER BY m.timestamp) as daily_return
            FROM positions p
            JOIN market_data m ON p.symbol = m.symbol
            WHERE p.id = ANY($1)
            AND m.timestamp >= NOW() - INTERVAL '1 year'
            ORDER BY m.timestamp DESC
        )
        SELECT 
            symbol,
            PERCENTILE_CONT($2) WITHIN GROUP (ORDER BY daily_return) as var_return
        FROM position_returns
        WHERE daily_return IS NOT NULL
        GROUP BY symbol
    `

    positionIDs := make([]int64, len(positions))
    for i, p := range positions {
        positionIDs[i] = p.ID
    }

    rows, err := rm.db.QueryContext(ctx, query, positionIDs, 1-rm.varConfidence)
    if err != nil {
        return 0, err
    }
    defer rows.Close()

    var totalVaR float64
    for rows.Next() {
        var symbol string
        var varReturn float64
        if err := rows.Scan(&symbol, &varReturn); err != nil {
            return 0, err
        }

        // Find position value
        var positionValue float64
        for _, p := range positions {
            if p.Symbol == symbol {
                positionValue = p.Quantity * p.EntryPrice
                break
            }
        }

        // Add to total VaR
        totalVaR += positionValue * varReturn
    }

    // Scale to configured VaR period
    return totalVaR * float64(rm.varDays), nil
}

func (rm *RiskManager) calculateDrawdown(ctx context.Context, positions []models.Position) (float64, error) {
    var totalDrawdown float64

    for _, pos := range positions {
        query := `
            SELECT (MAX(high) - MIN(close)) / MAX(high) as drawdown
            FROM market_data
            WHERE symbol = $1
            AND timestamp >= NOW() - INTERVAL '1 year'
        `

        var drawdown float64
        if err := rm.db.QueryRowContext(ctx, query, pos.Symbol).Scan(&drawdown); err != nil {
            return 0, err
        }

        totalDrawdown += drawdown * (pos.Quantity * pos.EntryPrice)
    }

    return totalDrawdown, nil
}

func (rm *RiskManager) calculateConcentration(ctx context.Context, positions []models.Position) (float64, error) {
    var totalValue, maxPosition float64

    for _, pos := range positions {
        value := pos.Quantity * pos.EntryPrice
        totalValue += value
        if value > maxPosition {
            maxPosition = value
        }
    }

    return maxPosition / totalValue, nil
}

func (rm *RiskManager) calculateVolatility(ctx context.Context, positions []models.Position) (float64, error) {
    query := `
        WITH daily_returns AS (
            SELECT 
                symbol,
                (close - LAG(close) OVER (PARTITION BY symbol ORDER BY timestamp)) / LAG(close) OVER (PARTITION BY symbol ORDER BY timestamp) as return
            FROM market_data
            WHERE symbol = ANY($1)
            AND timestamp >= NOW() - INTERVAL '30 days'
        )
        SELECT 
            symbol,
            STDDEV(return) as volatility
        FROM daily_returns
        WHERE return IS NOT NULL
        GROUP BY symbol
    `

    symbols := make([]string, len(positions))
    for i, p := range positions {
        symbols[i] = p.Symbol
    }

    rows, err := rm.db.QueryContext(ctx, query, symbols)
    if err != nil {
        return 0, err
    }
    defer rows.Close()

    var totalVolatility float64
    totalValue := 0.0

    for rows.Next() {
        var symbol string
        var volatility float64
        if err := rows.Scan(&symbol, &volatility); err != nil {
            return 0, err
        }

        // Find position value
        for _, p := range positions {
            if p.Symbol == symbol {
                value := p.Quantity * p.EntryPrice
                totalVolatility += volatility * value
                totalValue += value
                break
            }
        }
    }

    return totalVolatility / totalValue, nil
}

func (rm *RiskManager) generateAlerts(var_, drawdown, concentration, volatility float64) []Alert {
    var alerts []Alert
    now := time.Now()

    if var_ > rm.maxDrawdown {
        alerts = append(alerts, Alert{
            Type:      "VAR_EXCEEDED",
            Message:   fmt.Sprintf("Value at Risk (%.2f%%) exceeds threshold (%.2f%%)", var_*100, rm.maxDrawdown*100),
            Severity:  "HIGH",
            Timestamp: now,
        })
    }

    if drawdown > rm.maxDrawdown {
        alerts = append(alerts, Alert{
            Type:      "DRAWDOWN_EXCEEDED",
            Message:   fmt.Sprintf("Portfolio drawdown (%.2f%%) exceeds maximum (%.2f%%)", drawdown*100, rm.maxDrawdown*100),
            Severity:  "HIGH",
            Timestamp: now,
        })
    }

    if concentration > rm.maxConcentration {
        alerts = append(alerts, Alert{
            Type:      "CONCENTRATION_EXCEEDED",
            Message:   fmt.Sprintf("Asset concentration (%.2f%%) exceeds maximum (%.2f%%)", concentration*100, rm.maxConcentration*100),
            Severity:  "MEDIUM",
            Timestamp: now,
        })
    }

    if volatility > 0.02 { // 2% daily volatility threshold
        alerts = append(alerts, Alert{
            Type:      "HIGH_VOLATILITY",
            Message:   fmt.Sprintf("Portfolio volatility (%.2f%%) is high", volatility*100),
            Severity:  "MEDIUM",
            Timestamp: now,
        })
    }

    return alerts
}

func (rm *RiskManager) determineAlertLevel(alerts []Alert) string {
    hasHigh := false
    hasMedium := false

    for _, alert := range alerts {
        switch alert.Severity {
        case "HIGH":
            hasHigh = true
        case "MEDIUM":
            hasMedium = true
        }
    }

    if hasHigh {
        return "RED"
    }
    if hasMedium {
        return "YELLOW"
    }
    return "GREEN"
}

func (rm *RiskManager) getPositions(ctx context.Context, portfolioID int64) ([]models.Position, error) {
    query := `
        SELECT id, portfolio_id, symbol, quantity, entry_price
        FROM positions 
        WHERE portfolio_id = $1
    `
    
    rows, err := rm.db.QueryContext(ctx, query, portfolioID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var positions []models.Position
    for rows.Next() {
        var pos models.Position
        if err := rows.Scan(&pos.ID, &pos.PortfolioID, &pos.Symbol, &pos.Quantity, &pos.EntryPrice); err != nil {
            return nil, err
        }
        positions = append(positions, pos)
    }

    return positions, nil
}
