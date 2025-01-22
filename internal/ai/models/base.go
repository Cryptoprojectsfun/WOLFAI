package models

import (
	"context"
	"time"

	"github.com/Cryptoprojectsfun/quantai-clone/internal/models"
)

// Model represents a machine learning model for market prediction
type Model interface {
	// Train trains the model with historical data
	Train(ctx context.Context, data *TrainingData) error

	// Predict generates predictions for a given asset
	Predict(ctx context.Context, input *PredictionInput) (*PredictionOutput, error)

	// Validate checks model performance against validation data
	Validate(ctx context.Context, data *ValidationData) (*ValidationResults, error)

	// GetConfidence returns the model's confidence in its predictions
	GetConfidence() float64
}

// TrainingData represents the input data for model training
type TrainingData struct {
	AssetSymbol string
	Prices      []float64
	Volumes     []float64
	Times       []time.Time
	Indicators  []models.Indicator
	Labels      []float64 // Actual future prices for supervised learning
}

// PredictionInput represents the input for making predictions
type PredictionInput struct {
	AssetSymbol string
	Historical  []OHLCV
	Indicators  []models.Indicator
	Timeframe   string
}

// PredictionOutput represents the model's predictions
type PredictionOutput struct {
	PredictedHigh  float64
	PredictedLow   float64
	Confidence     float64
	SupportLevels  []float64
	ResistanceLevels []float64
	Signals        []models.Signal
}

// ValidationData represents data for model validation
type ValidationData struct {
	Predictions []*PredictionOutput
	Actuals     []float64
	Times       []time.Time
}

// ValidationResults represents model validation metrics
type ValidationResults struct {
	RMSE          float64  // Root Mean Square Error
	MAE           float64  // Mean Absolute Error
	Accuracy      float64  // Directional Accuracy
	SharpeRatio   float64
	MaxDrawdown   float64
	WinRate       float64
	ProfitFactor  float64
}

// OHLCV represents Open, High, Low, Close, Volume data
type OHLCV struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

// BaseModel provides common functionality for all models
type BaseModel struct {
	// Common fields
	confidence float64
	lastTrain  time.Time
	metrics    ValidationResults

	// Configuration
	config ModelConfig
}

// ModelConfig represents common model configuration
type ModelConfig struct {
	LearningRate     float64
	Epochs           int
	BatchSize        int
	ValidationSplit  float64
	FeatureColumns   []string
	SequenceLength   int
	PredictionSteps  int
	EarlyStopPatience int
}

// Common utility functions for all models

// CalculateRMSE calculates Root Mean Square Error
func CalculateRMSE(predictions, actuals []float64) float64 {
	if len(predictions) != len(actuals) {
		return 0
	}

	var sumSquares float64
	for i := range predictions {
		diff := predictions[i] - actuals[i]
		sumSquares += diff * diff
	}

	return math.Sqrt(sumSquares / float64(len(predictions)))
}

// CalculateMAE calculates Mean Absolute Error
func CalculateMAE(predictions, actuals []float64) float64 {
	if len(predictions) != len(actuals) {
		return 0
	}

	var sumErrors float64
	for i := range predictions {
		sumErrors += math.Abs(predictions[i] - actuals[i])
	}

	return sumErrors / float64(len(predictions))
}

// CalculateDirectionalAccuracy calculates directional prediction accuracy
func CalculateDirectionalAccuracy(predictions, actuals []float64) float64 {
	if len(predictions) <= 1 || len(actuals) <= 1 {
		return 0
	}

	correct := 0
	total := len(predictions) - 1

	for i := 1; i < len(predictions); i++ {
		predDirection := predictions[i] > predictions[i-1]
		actualDirection := actuals[i] > actuals[i-1]
		if predDirection == actualDirection {
			correct++
		}
	}

	return float64(correct) / float64(total)
}

// Normalize normalizes data to [0,1] range
func Normalize(data []float64) []float64 {
	if len(data) == 0 {
		return data
	}

	min := data[0]
	max := data[0]

	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	range_ := max - min
	if range_ == 0 {
		return make([]float64, len(data))
	}

	normalized := make([]float64, len(data))
	for i, v := range data {
		normalized[i] = (v - min) / range_
	}

	return normalized
}

// Standardize performs z-score standardization
func Standardize(data []float64) []float64 {
	if len(data) == 0 {
		return data
	}

	mean := 0.0
	for _, v := range data {
		mean += v
	}
	mean /= float64(len(data))

	variance := 0.0
	for _, v := range data {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(data))
	stdDev := math.Sqrt(variance)

	if stdDev == 0 {
		return make([]float64, len(data))
	}

	standardized := make([]float64, len(data))
	for i, v := range data {
		standardized[i] = (v - mean) / stdDev
	}

	return standardized
}

// DetectOutliers detects outliers using IQR method
func DetectOutliers(data []float64) []bool {
	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)

	q1 := sorted[len(sorted)/4]
	q3 := sorted[len(sorted)*3/4]
	iqr := q3 - q1
	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr

	outliers := make([]bool, len(data))
	for i, v := range data {
		outliers[i] = v < lowerBound || v > upperBound
	}

	return outliers
}

// CalculateSupportResistance calculates support and resistance levels
func CalculateSupportResistance(prices []float64, window int) ([]float64, []float64) {
	if len(prices) < window {
		return nil, nil
	}

	var supports, resistances []float64

	for i := window; i < len(prices)-window; i++ {
		isSupport := true
		isResistance := true

		for j := i - window; j <= i+window; j++ {
			if j == i {
				continue
			}

			if prices[j] < prices[i] {
				isSupport = false
			}
			if prices[j] > prices[i] {
				isResistance = false
			}
		}

		if isSupport {
			supports = append(supports, prices[i])
		}
		if isResistance {
			resistances = append(resistances, prices[i])
		}
	}

	return supports, resistances
}