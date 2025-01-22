package analytics

import (
	"context"
	"database/sql"
	"time"
	"math"

	"github.com/Cryptoprojectsfun/quantai-clone/internal/models"
)

type Service struct {
	db        *sql.DB
	aiService AIService
}

type AIService interface {
	GeneratePrediction(ctx context.Context, symbol string, timeframe string) (*models.Prediction, error)
	AnalyzeMarketSentiment(ctx context.Context, symbol string) (*models.MarketAnalysis, error)
}

type RiskMetrics struct {
	Volatility   float64 `json:"volatility"`
	SharpeRatio  float64 `json:"sharpe_ratio"`
	SortinoRatio float64 `json:"sortino_ratio"`
	MaxDrawdown  float64 `json:"max_drawdown"`
	VaR          float64 `json:"var"` // Value at Risk
}

type AdvancedAnalytics struct {
	CorrelationMatrix map[string]map[string]float64 `json:"correlation_matrix"`
	RiskMetrics       map[string]RiskMetrics        `json:"risk_metrics"`
	PortfolioMetrics  PortfolioMetrics             `json:"portfolio_metrics"`
}

type PortfolioMetrics struct {
	TotalValue     float64   `json:"total_value"`
	DailyReturn    float64   `json:"daily_return"`
	WeeklyReturn   float64   `json:"weekly_return"`
	MonthlyReturn  float64   `json:"monthly_return"`
	YearlyReturn   float64   `json:"yearly_return"`
	RiskAdjusted   float64   `json:"risk_adjusted"`
	Diversification float64   `json:"diversification"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func NewService(db *sql.DB, aiService AIService) *Service {
	return &Service{
		db:        db,
		aiService: aiService,
	}
}

func (s *Service) GetMarketAnalysis(ctx context.Context, symbol string) (*models.MarketAnalysis, error) {
	// First, try to get recent analysis from cache/db
	analysis, err := s.getStoredAnalysis(ctx, symbol)
	if err == nil && analysis.UpdatedAt.Add(15*time.Minute).After(time.Now()) {
		return analysis, nil
	}

	// Generate new analysis using AI service
	analysis, err = s.aiService.AnalyzeMarketSentiment(ctx, symbol)
	if err != nil {
		return nil, err
	}

	// Store the new analysis
	if err := s.storeAnalysis(ctx, analysis); err != nil {
		return nil, err
	}

	return analysis, nil
}

func (s *Service) GetAdvancedAnalytics(ctx context.Context, portfolioID string) (*AdvancedAnalytics, error) {
	metrics := &AdvancedAnalytics{
		CorrelationMatrix: make(map[string]map[string]float64),
		RiskMetrics:       make(map[string]RiskMetrics),
	}

	// Get portfolio assets
	assets, err := s.getPortfolioAssets(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	// Calculate correlation matrix
	for _, asset1 := range assets {
		metrics.CorrelationMatrix[asset1.Symbol] = make(map[string]float64)
		for _, asset2 := range assets {
			correlation, err := s.calculateCorrelation(ctx, asset1.Symbol, asset2.Symbol)
			if err != nil {
				return nil, err
			}
			metrics.CorrelationMatrix[asset1.Symbol][asset2.Symbol] = correlation
		}

		// Calculate risk metrics for each asset
		riskMetrics, err := s.calculateRiskMetrics(ctx, asset1.Symbol)
		if err != nil {
			return nil, err
		}
		metrics.RiskMetrics[asset1.Symbol] = riskMetrics
	}

	// Calculate portfolio-level metrics
	portfolioMetrics, err := s.calculatePortfolioMetrics(ctx, portfolioID, assets)
	if err != nil {
		return nil, err
	}
	metrics.PortfolioMetrics = portfolioMetrics

	return metrics, nil
}

func (s *Service) calculateCorrelation(ctx context.Context, symbol1, symbol2 string) (float64, error) {
	query := `
		WITH daily_returns AS (
			SELECT 
				date,
				symbol,
				(price - LAG(price) OVER (PARTITION BY symbol ORDER BY date)) / LAG(price) OVER (PARTITION BY symbol ORDER BY date) as return
			FROM asset_prices
			WHERE symbol IN ($1, $2)
			AND date >= $3
		)
		SELECT CORR(r1.return, r2.return) as correlation
		FROM daily_returns r1
		JOIN daily_returns r2 ON r1.date = r2.date AND r1.symbol < r2.symbol
		WHERE r1.symbol = $1 AND r2.symbol = $2
	`

	var correlation float64
	err := s.db.QueryRowContext(
		ctx,
		query,
		symbol1,
		symbol2,
		time.Now().AddDate(0, -6, 0),
	).Scan(&correlation)

	if err != nil {
		return 0, err
	}

	return correlation, nil
}

func (s *Service) calculateRiskMetrics(ctx context.Context, symbol string) (RiskMetrics, error) {
	query := `
		WITH daily_returns AS (
			SELECT 
				(price - LAG(price) OVER (ORDER BY date)) / LAG(price) OVER (ORDER BY date) as return,
				date
			FROM asset_prices
			WHERE symbol = $1
			AND date >= $2
			ORDER BY date
		)
		SELECT 
			STDDEV(return) * SQRT(252) as volatility,
			AVG(return) / STDDEV(return) * SQRT(252) as sharpe_ratio,
			MIN(return) as max_drawdown
		FROM daily_returns
	`

	var metrics RiskMetrics
	err := s.db.QueryRowContext(
		ctx,
		query,
		symbol,
		time.Now().AddDate(-1, 0, 0),
	).Scan(
		&metrics.Volatility,
		&metrics.SharpeRatio,
		&metrics.MaxDrawdown,
	)

	if err != nil {
		return RiskMetrics{}, err
	}

	// Calculate Value at Risk (VaR) using historical simulation
	metrics.VaR = s.calculateVaR(ctx, symbol)
	
	// Calculate Sortino Ratio (similar to Sharpe but only considering negative returns)
	metrics.SortinoRatio = s.calculateSortinoRatio(ctx, symbol)

	return metrics, nil
}

func (s *Service) calculatePortfolioMetrics(ctx context.Context, portfolioID string, assets []models.Asset) (PortfolioMetrics, error) {
	var metrics PortfolioMetrics

	// Calculate total portfolio value
	for _, asset := range assets {
		metrics.TotalValue += asset.Value
	}

	// Calculate returns for different timeframes
	metrics.DailyReturn = s.calculateReturn(ctx, portfolioID, time.Hour*24)
	metrics.WeeklyReturn = s.calculateReturn(ctx, portfolioID, time.Hour*24*7)
	metrics.MonthlyReturn = s.calculateReturn(ctx, portfolioID, time.Hour*24*30)
	metrics.YearlyReturn = s.calculateReturn(ctx, portfolioID, time.Hour*24*365)

	// Calculate risk-adjusted return (Sharpe Ratio)
	riskFreeRate := 0.02 // 2% annual risk-free rate
	portfolioReturn := metrics.YearlyReturn
	portfolioVolatility := s.calculatePortfolioVolatility(ctx, portfolioID)
	metrics.RiskAdjusted = (portfolioReturn - riskFreeRate) / portfolioVolatility

	// Calculate portfolio diversification score
	metrics.Diversification = s.calculateDiversificationScore(assets)

	metrics.UpdatedAt = time.Now()

	return metrics, nil
}

func (s *Service) calculateDiversificationScore(assets []models.Asset) float64 {
	if len(assets) == 0 {
		return 0
	}

	// Calculate Herfindahl-Hirschman Index (HHI)
	var sumSquares float64
	totalValue := 0.0

	for _, asset := range assets {
		totalValue += asset.Value
	}

	for _, asset := range assets {
		weight := asset.Value / totalValue
		sumSquares += weight * weight
	}

	// Convert HHI to diversification score (1 - HHI)
	// This gives a score between 0 (completely concentrated) and 1 (perfectly diversified)
	return 1 - sumSquares
}

func (s *Service) calculateVaR(ctx context.Context, symbol string) float64 {
	// Fetch historical returns
	query := `
		SELECT 
			(price - LAG(price) OVER (ORDER BY date)) / LAG(price) OVER (ORDER BY date) as return
		FROM asset_prices
		WHERE symbol = $1
		AND date >= $2
		ORDER BY return
	`

	rows, err := s.db.QueryContext(
		ctx,
		query,
		symbol,
		time.Now().AddDate(-1, 0, 0),
	)
	if err != nil {
		return 0
	}
	defer rows.Close()

	var returns []float64
	for rows.Next() {
		var ret float64
		if err := rows.Scan(&ret); err != nil {
			return 0
		}
		returns = append(returns, ret)
	}

	// Calculate 95% VaR
	if len(returns) == 0 {
		return 0
	}

	// Sort returns (should already be sorted from query)
	percentileIndex := int(float64(len(returns)) * 0.05)
	return -returns[percentileIndex] // Convert to positive number for reporting
}

func (s *Service) calculateSortinoRatio(ctx context.Context, symbol string) float64 {
	query := `
		WITH daily_returns AS (
			SELECT 
				(price - LAG(price) OVER (ORDER BY date)) / LAG(price) OVER (ORDER BY date) as return
			FROM asset_prices
			WHERE symbol = $1
			AND date >= $2
		)
		SELECT 
			AVG(return) as avg_return,
			STDDEV(CASE WHEN return < 0 THEN return ELSE 0 END) as downside_deviation
		FROM daily_returns
	`

	var avgReturn, downsideDeviation float64
	err := s.db.QueryRowContext(
		ctx,
		query,
		symbol,
		time.Now().AddDate(-1, 0, 0),
	).Scan(&avgReturn, &downsideDeviation)

	if err != nil || downsideDeviation == 0 {
		return 0
	}

	riskFreeRate := 0.02 / 252 // Daily risk-free rate (2% annual)
	return (avgReturn - riskFreeRate) / downsideDeviation * math.Sqrt(252)
}