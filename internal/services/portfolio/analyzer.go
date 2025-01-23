package portfolio

import (
    "context"
    "database/sql"
    "fmt"
    "math"
    "time"

    "github.com/Cryptoprojectsfun/quantai-clone/internal/models"
)

type PortfolioAnalyzer struct {
    db *sql.DB
}

type PortfolioMetrics struct {
    TotalValue     float64   `json:"total_value"`
    PnL            float64   `json:"pnl"`
    PnLPercentage  float64   `json:"pnl_percentage"`
    Volatility     float64   `json:"volatility"`
    SharpeRatio    float64   `json:"sharpe_ratio"`
    LastUpdated    time.Time `json:"last_updated"`
}

type PositionMetrics struct {
    Symbol         string    `json:"symbol"`
    Quantity       float64   `json:"quantity"`
    CurrentPrice   float64   `json:"current_price"`
    Value          float64   `json:"value"`
    PnL            float64   `json:"pnl"`
    PnLPercentage  float64   `json:"pnl_percentage"`
}

func NewPortfolioAnalyzer(db *sql.DB) *PortfolioAnalyzer {
    return &PortfolioAnalyzer{db: db}
}

func (a *PortfolioAnalyzer) AnalyzePortfolio(ctx context.Context, portfolioID int64) (*PortfolioMetrics, error) {
    positions, err := a.getPositions(ctx, portfolioID)
    if err != nil {
        return nil, err
    }

    positionMetrics, err := a.analyzePositions(ctx, positions)
    if err != nil {
        return nil, err
    }

    return a.calculatePortfolioMetrics(ctx, positionMetrics)
}

func (a *PortfolioAnalyzer) getPositions(ctx context.Context, portfolioID int64) ([]models.Position, error) {
    query := `
        SELECT id, portfolio_id, symbol, quantity, entry_price
        FROM positions 
        WHERE portfolio_id = $1
    `
    
    rows, err := a.db.QueryContext(ctx, query, portfolioID)
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

func (a *PortfolioAnalyzer) analyzePositions(ctx context.Context, positions []models.Position) ([]PositionMetrics, error) {
    var metrics []PositionMetrics

    for _, pos := range positions {
        currentPrice, err := a.getLatestPrice(ctx, pos.Symbol)
        if err != nil {
            return nil, fmt.Errorf("failed to get price for %s: %v", pos.Symbol, err)
        }

        value := pos.Quantity * currentPrice
        pnl := value - (pos.Quantity * pos.EntryPrice)
        pnlPercentage := (pnl / (pos.Quantity * pos.EntryPrice)) * 100

        metrics = append(metrics, PositionMetrics{
            Symbol:        pos.Symbol,
            Quantity:      pos.Quantity,
            CurrentPrice:  currentPrice,
            Value:        value,
            PnL:          pnl,
            PnLPercentage: pnlPercentage,
        })
    }

    return metrics, nil
}

func (a *PortfolioAnalyzer) calculatePortfolioMetrics(ctx context.Context, positions []PositionMetrics) (*PortfolioMetrics, error) {
    var totalValue, totalPnL float64
    
    for _, pos := range positions {
        totalValue += pos.Value
        totalPnL += pos.PnL
    }

    volatility, err := a.calculateVolatility(ctx, positions)
    if err != nil {
        return nil, err
    }

    // Calculate Sharpe ratio using risk-free rate of 2%
    riskFreeRate := 0.02
    sharpeRatio := (totalPnL/totalValue - riskFreeRate) / volatility

    return &PortfolioMetrics{
        TotalValue:    totalValue,
        PnL:           totalPnL,
        PnLPercentage: (totalPnL / (totalValue - totalPnL)) * 100,
        Volatility:    volatility,
        SharpeRatio:   sharpeRatio,
        LastUpdated:   time.Now(),
    }, nil
}

func (a *PortfolioAnalyzer) calculateVolatility(ctx context.Context, positions []PositionMetrics) (float64, error) {
    // Calculate 30-day rolling volatility
    days := 30
    returns := make([]float64, days)
    
    query := `
        SELECT close 
        FROM market_data 
        WHERE symbol = $1 
        AND timestamp >= NOW() - INTERVAL '30 days'
        ORDER BY timestamp DESC
    `

    var totalVolatility float64
    for _, pos := range positions {
        rows, err := a.db.QueryContext(ctx, query, pos.Symbol)
        if err != nil {
            return 0, err
        }
        defer rows.Close()

        i := 0
        var prevClose float64
        for rows.Next() && i < days {
            var close float64
            if err := rows.Scan(&close); err != nil {
                return 0, err
            }
            
            if i > 0 {
                returns[i-1] = (close - prevClose) / prevClose
            }
            prevClose = close
            i++
        }

        // Calculate standard deviation of returns
        var sum, sumSq float64
        for _, r := range returns[:i-1] {
            sum += r
            sumSq += r * r
        }
        mean := sum / float64(i-1)
        variance := (sumSq / float64(i-1)) - (mean * mean)
        
        // Weight volatility by position value
        weight := pos.Value / totalVolatility
        totalVolatility += math.Sqrt(variance) * weight
    }

    return totalVolatility, nil
}

func (a *PortfolioAnalyzer) getLatestPrice(ctx context.Context, symbol string) (float64, error) {
    query := `
        SELECT close 
        FROM market_data 
        WHERE symbol = $1 
        ORDER BY timestamp DESC 
        LIMIT 1
    `
    
    var price float64
    if err := a.db.QueryRowContext(ctx, query, symbol).Scan(&price); err != nil {
        return 0, err
    }
    
    return price, nil
}
