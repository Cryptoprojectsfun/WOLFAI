package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/Cryptoprojectsfun/quantai-clone/internal/models"
)

type AnalyticsHandler struct {
	analyticsService AnalyticsService
	aiService        AIService
}

type AIService interface {
	GeneratePrediction(ctx context.Context, symbol string, timeframe string) (*models.Prediction, error)
	AnalyzeMarketSentiment(ctx context.Context, symbol string) (*models.MarketAnalysis, error)
}

type MarketAnalysisResponse struct {
	Analysis    *models.MarketAnalysis `json:"analysis"`
	Predictions *models.Prediction     `json:"predictions"`
}

type PredictionResponse struct {
	Prediction  *models.Prediction    `json:"prediction"`
	MarketData  *models.MarketData   `json:"market_data"`
	Indicators  []models.Indicator   `json:"indicators"`
	Confidence  float64              `json:"confidence"`
	ValidUntil  time.Time            `json:"valid_until"`
}

func NewAnalyticsHandler(analyticsService AnalyticsService, aiService AIService) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
		aiService:        aiService,
	}
}

func (h *AnalyticsHandler) GetMarketAnalysis(w http.ResponseWriter, r *http.Request) {
	// Get symbol from URL
	vars := mux.Vars(r)
	symbol := vars["symbol"]

	// Get analysis
	analysis, err := h.analyticsService.GetMarketAnalysis(r.Context(), symbol)
	if err != nil {
		http.Error(w, "Error fetching market analysis", http.StatusInternalServerError)
		return
	}

	// Get predictions (default to 24h timeframe)
	predictions, err := h.aiService.GeneratePrediction(r.Context(), symbol, "24h")
	if err != nil {
		// Log error but continue
		// logger.Error("Failed to generate predictions", "error", err)
	}

	response := MarketAnalysisResponse{
		Analysis:    analysis,
		Predictions: predictions,
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AnalyticsHandler) GetPredictions(w http.ResponseWriter, r *http.Request) {
	// Get symbol from URL
	vars := mux.Vars(r)
	symbol := vars["symbol"]

	// Get timeframe from query params (default to 24h)
	timeframe := r.URL.Query().Get("timeframe")
	if timeframe == "" {
		timeframe = "24h"
	}

	// Validate timeframe
	if !isValidTimeframe(timeframe) {
		http.Error(w, "Invalid timeframe. Valid values: 1h, 4h, 24h, 7d", http.StatusBadRequest)
		return
	}

	// Generate predictions
	prediction, err := h.aiService.GeneratePrediction(r.Context(), symbol, timeframe)
	if err != nil {
		http.Error(w, "Error generating predictions", http.StatusInternalServerError)
		return
	}

	// Get current market data for context
	marketData, err := h.analyticsService.GetMarketData(r.Context(), symbol)
	if err != nil {
		// Log error but continue
		// logger.Error("Failed to fetch market data", "error", err)
	}

	response := PredictionResponse{
		Prediction:  prediction,
		MarketData:  marketData,
		Indicators:  prediction.Indicators,
		Confidence:  prediction.Confidence,
		ValidUntil:  prediction.ValidUntil,
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AnalyticsHandler) GetPortfolioAnalytics(w http.ResponseWriter, r *http.Request) {
	// Get portfolio ID from URL
	vars := mux.Vars(r)
	portfolioID := vars["id"]

	// Get timeframe from query params (default to all)
	timeframe := r.URL.Query().Get("timeframe")

	// Get analytics
	analytics, err := h.analyticsService.GetAdvancedAnalytics(r.Context(), portfolioID)
	if err != nil {
		http.Error(w, "Error fetching portfolio analytics", http.StatusInternalServerError)
		return
	}

	// If timeframe specified, filter metrics
	if timeframe != "" {
		analytics = filterAnalyticsByTimeframe(analytics, timeframe)
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

func (h *AnalyticsHandler) GetHistoricalPerformance(w http.ResponseWriter, r *http.Request) {
	// Get portfolio ID from URL
	vars := mux.Vars(r)
	portfolioID := vars["id"]

	// Get date range from query params
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	// Parse dates
	start, err := time.Parse(time.RFC3339, startDate)
	if err != nil {
		http.Error(w, "Invalid start date format", http.StatusBadRequest)
		return
	}

	end, err := time.Parse(time.RFC3339, endDate)
	if err != nil {
		http.Error(w, "Invalid end date format", http.StatusBadRequest)
		return
	}

	// Get historical performance
	performance, err := h.analyticsService.GetHistoricalPerformance(r.Context(), portfolioID, start, end)
	if err != nil {
		http.Error(w, "Error fetching historical performance", http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(performance)
}

func isValidTimeframe(timeframe string) bool {
	validTimeframes := map[string]bool{
		"1h":  true,
		"4h":  true,
		"24h": true,
		"7d":  true,
	}
	return validTimeframes[timeframe]
}

func filterAnalyticsByTimeframe(analytics *models.AdvancedAnalytics, timeframe string) *models.AdvancedAnalytics {
	// Create a copy of analytics
	filtered := *analytics

	// Filter metrics based on timeframe
	switch timeframe {
	case "24h":
		filtered.PortfolioMetrics.WeeklyReturn = 0
		filtered.PortfolioMetrics.MonthlyReturn = 0
		filtered.PortfolioMetrics.YearlyReturn = 0
	case "7d":
		filtered.PortfolioMetrics.MonthlyReturn = 0
		filtered.PortfolioMetrics.YearlyReturn = 0
	case "30d":
		filtered.PortfolioMetrics.YearlyReturn = 0
	}

	return &filtered
}

// Error types for analytics operations
var (
	ErrInvalidSymbol       = NewValidationError("invalid symbol")
	ErrInvalidTimeframe    = NewValidationError("invalid timeframe")
	ErrInvalidDateRange    = NewValidationError("invalid date range")
	ErrTooManyPredictions  = NewValidationError("too many predictions requested")
)

// Helper structures for analytics
type MarketData struct {
	Symbol        string    `json:"symbol"`
	CurrentPrice  float64   `json:"current_price"`
	Volume24h     float64   `json:"volume_24h"`
	PriceChange24h float64  `json:"price_change_24h"`
	LastUpdate    time.Time `json:"last_update"`
}

type HistoricalPerformance struct {
	PortfolioID  string               `json:"portfolio_id"`
	StartDate    time.Time            `json:"start_date"`
	EndDate      time.Time            `json:"end_date"`
	DailyReturns []DailyReturn        `json:"daily_returns"`
	Summary      PerformanceSummary   `json:"summary"`
}

type DailyReturn struct {
	Date   time.Time `json:"date"`
	Return float64   `json:"return"`
	Value  float64   `json:"value"`
}

type PerformanceSummary struct {
	TotalReturn     float64 `json:"total_return"`
	AnnualizedReturn float64 `json:"annualized_return"`
	Volatility      float64 `json:"volatility"`
	SharpeRatio     float64 `json:"sharpe_ratio"`
	MaxDrawdown     float64 `json:"max_drawdown"`
	WinningDays     int     `json:"winning_days"`
	LosingDays      int     `json:"losing_days"`
}