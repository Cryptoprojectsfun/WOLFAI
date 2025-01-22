package ai

import (
	"context"
	"time"

	"github.com/Cryptoprojectsfun/quantai-clone/internal/models"
	"gonum.org/v1/gonum/stat"
)

type Service struct {
	repository PredictionRepository
}

type PredictionRepository interface {
	SavePrediction(ctx context.Context, prediction *models.Prediction) error
	GetPredictions(ctx context.Context, symbol string, timeframe string) ([]models.Prediction, error)
	SaveMarketAnalysis(ctx context.Context, analysis *models.MarketAnalysis) error
}

func NewService(repo PredictionRepository) *Service {
	return &Service{
		repository: repo,
	}
}

// GeneratePrediction creates a new price prediction using AI models
func (s *Service) GeneratePrediction(ctx context.Context, symbol string, timeframe string) (*models.Prediction, error) {
	// Fetch historical data
	historicalData, err := s.fetchHistoricalData(ctx, symbol)
	if err != nil {
		return nil, err
	}

	// Calculate technical indicators
	indicators := s.calculateIndicators(historicalData)

	// Run AI model prediction
	prediction := s.runAIModel(historicalData, indicators)

	// Calculate confidence score
	confidence := s.calculateConfidence(prediction, historicalData)

	// Create prediction model
	pred := &models.Prediction{
		AssetSymbol:   symbol,
		Timeframe:     timeframe,
		PredictedHigh: prediction.High,
		PredictedLow:  prediction.Low,
		Confidence:    confidence,
		Indicators:    indicators,
		CreatedAt:     time.Now(),
		ValidUntil:    time.Now().Add(s.getTimeframeValidation(timeframe)),
	}

	// Save prediction
	if err := s.repository.SavePrediction(ctx, pred); err != nil {
		return nil, err
	}

	return pred, nil
}

// AnalyzeMarketSentiment performs sentiment analysis on market data
func (s *Service) AnalyzeMarketSentiment(ctx context.Context, symbol string) (*models.MarketAnalysis, error) {
	// Fetch market data
	marketData, err := s.fetchMarketData(ctx, symbol)
	if err != nil {
		return nil, err
	}

	// Analyze social media sentiment
	socialSentiment := s.analyzeSocialMediaSentiment(symbol)

	// Analyze news sentiment
	newsSentiment := s.analyzeNewsSentiment(symbol)

	// Generate market signals
	signals := s.generateSignals(marketData, socialSentiment, newsSentiment)

	// Calculate trend strength
	trendStrength := s.calculateTrendStrength(marketData)

	// Create analysis model
	analysis := &models.MarketAnalysis{
		AssetSymbol:    symbol,
		Sentiment:      (socialSentiment + newsSentiment) / 2,
		Volume24h:      marketData.Volume,
		PriceChange24h: marketData.PriceChange,
		TrendStrength:  trendStrength,
		UpdatedAt:      time.Now(),
		Signals:        signals,
	}

	// Save analysis
	if err := s.repository.SaveMarketAnalysis(ctx, analysis); err != nil {
		return nil, err
	}

	return analysis, nil
}

// Helper structures
type HistoricalData struct {
	Prices  []float64
	Volumes []float64
	Times   []time.Time
}

type MarketData struct {
	Price       float64
	Volume      float64
	PriceChange float64
	OHLCV       []OHLCV
}

type OHLCV struct {
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
	Time   time.Time
}

type AIModelPrediction struct {
	High   float64
	Low    float64
	Signal string
}

// Technical Analysis Functions
func (s *Service) calculateIndicators(data *HistoricalData) []models.Indicator {
	var indicators []models.Indicator

	// Calculate Moving Averages
	sma20 := s.calculateSMA(data.Prices, 20)
	ema50 := s.calculateEMA(data.Prices, 50)

	// Calculate RSI
	rsi := s.calculateRSI(data.Prices, 14)

	// Calculate MACD
	macd, signal := s.calculateMACD(data.Prices)

	// Calculate Bollinger Bands
	upper, middle, lower := s.calculateBollingerBands(data.Prices, 20, 2)

	indicators = append(indicators, []models.Indicator{
		{Name: "SMA20", Value: sma20, Weight: 0.2},
		{Name: "EMA50", Value: ema50, Weight: 0.3},
		{Name: "RSI", Value: rsi, Weight: 0.25},
		{Name: "MACD", Value: macd, Weight: 0.15},
		{Name: "MACD_Signal", Value: signal, Weight: 0.1},
		{Name: "BB_Upper", Value: upper, Weight: 0.1},
		{Name: "BB_Middle", Value: middle, Weight: 0.1},
		{Name: "BB_Lower", Value: lower, Weight: 0.1},
	}...)

	return indicators
}

func (s *Service) calculateSMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}

	sum := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
	}

	return sum / float64(period)
}

func (s *Service) calculateEMA(prices []float64, period int) float64 {
	multiplier := 2.0 / float64(period+1)
	ema := prices[0]

	for i := 1; i < len(prices); i++ {
		ema = (prices[i]-ema)*multiplier + ema
	}

	return ema
}

func (s *Service) calculateRSI(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 0
	}

	var gains, losses []float64
	for i := 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change >= 0 {
			gains = append(gains, change)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -change)
		}
	}

	avgGain := stat.Mean(gains, nil)
	avgLoss := stat.Mean(losses, nil)

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

func (s *Service) calculateMACD(prices []float64) (float64, float64) {
	ema12 := s.calculateEMA(prices, 12)
	ema26 := s.calculateEMA(prices, 26)
	macd := ema12 - ema26
	signal := s.calculateEMA([]float64{macd}, 9)
	return macd, signal
}

func (s *Service) calculateBollingerBands(prices []float64, period int, stdDev float64) (float64, float64, float64) {
	if len(prices) < period {
		return 0, 0, 0
	}

	// Calculate middle band (SMA)
	middle := s.calculateSMA(prices, period)

	// Calculate standard deviation
	var sum float64
	for i := len(prices) - period; i < len(prices); i++ {
		diff := prices[i] - middle
		sum += diff * diff
	}
	sd := stdDev * stat.StdDev(prices[len(prices)-period:], nil)

	upper := middle + sd
	lower := middle - sd

	return upper, middle, lower
}

// Market Analysis Functions
func (s *Service) calculateTrendStrength(data MarketData) float64 {
	// Calculate ADX (Average Directional Index)
	adx := s.calculateADX(data.OHLCV, 14)
	
	// Calculate price momentum
	momentum := s.calculateMomentum(data.OHLCV, 14)
	
	// Calculate volume trend
	volumeTrend := s.calculateVolumeTrend(data.OHLCV, 14)
	
	// Combine indicators for trend strength
	trendStrength := (adx*0.4 + momentum*0.4 + volumeTrend*0.2)
	
	return trendStrength
}

func (s *Service) generateSignals(data MarketData, socialSentiment, newsSentiment float64) []models.Signal {
	var signals []models.Signal

	// Technical Analysis Signals
	techSignals := s.generateTechnicalSignals(data)
	signals = append(signals, techSignals...)

	// Sentiment Signals
	sentimentSignals := s.generateSentimentSignals(socialSentiment, newsSentiment)
	signals = append(signals, sentimentSignals...)

	// Volume Analysis Signals
	volumeSignals := s.generateVolumeSignals(data)
	signals = append(signals, volumeSignals...)

	return signals
}

func (s *Service) getTimeframeValidation(timeframe string) time.Duration {
	switch timeframe {
	case "1h":
		return time.Hour
	case "4h":
		return 4 * time.Hour
	case "24h":
		return 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}

// AI Model Functions
func (s *Service) runAIModel(data *HistoricalData, indicators []models.Indicator) AIModelPrediction {
	// Initialize prediction
	pred := AIModelPrediction{}

	// Prepare features for the model
	features := s.prepareFeatures(data, indicators)

	// Run prediction model
	pred.High, pred.Low = s.predictPrices(features)

	// Generate trading signal
	pred.Signal = s.generateTradingSignal(pred, data)

	return pred
}

func (s *Service) calculateConfidence(pred AIModelPrediction, data *HistoricalData) float64 {
	// Calculate model accuracy based on historical predictions
	accuracy := s.calculateModelAccuracy(data)

	// Calculate prediction volatility
	volatility := s.calculatePredictionVolatility(pred, data)

	// Calculate market conditions confidence
	marketConfidence := s.calculateMarketConfidence(data)

	// Combine factors for final confidence score
	confidence := (accuracy*0.4 + (1-volatility)*0.3 + marketConfidence*0.3)

	return confidence
}