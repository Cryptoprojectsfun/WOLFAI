package models

import (
	"context"
	"math"
	"sync"

	"github.com/Cryptoprojectsfun/quantai-clone/internal/models"
)

// EnsembleModel combines predictions from multiple models
type EnsembleModel struct {
	BaseModel
	
	models []Model
	weights map[string]float64 // weights for each model
	
	// Performance tracking
	modelScores map[string]float64
	adaptiveWeights bool
}

// ModelWeight represents weight configuration for ensemble models
type ModelWeight struct {
	ModelID string
	Weight  float64
}

// EnsembleConfig represents ensemble-specific configuration
type EnsembleConfig struct {
	BaseConfig    ModelConfig
	ModelWeights  []ModelWeight
	AdaptiveWeights bool
}

// NewEnsembleModel creates a new ensemble model
func NewEnsembleModel(config EnsembleConfig) *EnsembleModel {
	weights := make(map[string]float64)
	for _, w := range config.ModelWeights {
		weights[w.ModelID] = w.Weight
	}

	return &EnsembleModel{
		BaseModel: BaseModel{
			config: config.BaseConfig,
		},
		weights: weights,
		modelScores: make(map[string]float64),
		adaptiveWeights: config.AdaptiveWeights,
	}
}

// AddModel adds a model to the ensemble
func (m *EnsembleModel) AddModel(model Model, id string, weight float64) {
	m.models = append(m.models, model)
	m.weights[id] = weight
	m.modelScores[id] = 1.0 // Initial score
}

// Predict generates ensemble predictions
func (m *EnsembleModel) Predict(ctx context.Context, input *PredictionInput) (*PredictionOutput, error) {
	var (
		predictions = make([]*PredictionOutput, len(m.models))
		errors     = make([]error, len(m.models))
		wg         sync.WaitGroup
	)

	// Get predictions from all models in parallel
	for i, model := range m.models {
		wg.Add(1)
		go func(idx int, mdl Model) {
			defer wg.Done()
			pred, err := mdl.Predict(ctx, input)
			predictions[idx] = pred
			errors[idx] = err
		}(i, model)
	}

	wg.Wait()

	// Check for errors
	for _, err := range errors {
		if err != nil {
			return nil, err
		}
	}

	// Combine predictions using weighted average
	combinedPred := m.combinePredictions(predictions)

	// Update model weights if adaptive weighting is enabled
	if m.adaptiveWeights {
		m.updateModelWeights(predictions, input)
	}

	return combinedPred, nil
}

// combinePredictions merges predictions from multiple models
func (m *EnsembleModel) combinePredictions(predictions []*PredictionOutput) *PredictionOutput {
	var (
		totalWeight float64
		weightedHigh float64
		weightedLow float64
		weightedConfidence float64
	)

	// Calculate weighted averages
	for i, pred := range predictions {
		modelID := m.getModelID(i)
		weight := m.weights[modelID]

		weightedHigh += pred.PredictedHigh * weight
		weightedLow += pred.PredictedLow * weight
		weightedConfidence += pred.Confidence * weight
		totalWeight += weight
	}

	// Normalize by total weight
	if totalWeight > 0 {
		weightedHigh /= totalWeight
		weightedLow /= totalWeight
		weightedConfidence /= totalWeight
	}

	// Combine support and resistance levels
	supports, resistances := m.combineSupportsAndResistances(predictions)

	// Combine signals with confidence weighting
	signals := m.combineSignals(predictions)

	return &PredictionOutput{
		PredictedHigh:     weightedHigh,
		PredictedLow:      weightedLow,
		Confidence:        weightedConfidence,
		SupportLevels:     supports,
		ResistanceLevels:  resistances,
		Signals:           signals,
	}
}

// updateModelWeights adjusts model weights based on performance
func (m *EnsembleModel) updateModelWeights(predictions []*PredictionOutput, input *PredictionInput) {
	// Calculate recent accuracy for each model
	for i, pred := range predictions {
		modelID := m.getModelID(i)
		
		// Calculate prediction error
		avgPred := (pred.PredictedHigh + pred.PredictedLow) / 2
		actualPrice := input.Historical[len(input.Historical)-1].Close
		error := math.Abs(avgPred - actualPrice) / actualPrice

		// Update model score using exponential moving average
		alpha := 0.1 // Learning rate for weight adaptation
		m.modelScores[modelID] = m.modelScores[modelID]*(1-alpha) + (1-error)*alpha
	}

	// Normalize scores to weights
	var totalScore float64
	for _, score := range m.modelScores {
		totalScore += score
	}

	if totalScore > 0 {
		for modelID, score := range m.modelScores {
			m.weights[modelID] = score / totalScore
		}
	}
}

// combineSupportsAndResistances merges support and resistance levels from all models
func (m *EnsembleModel) combineSupportsAndResistances(predictions []*PredictionOutput) ([]float64, []float64) {
	// Collect all levels
	var allSupports, allResistances []float64
	for _, pred := range predictions {
		allSupports = append(allSupports, pred.SupportLevels...)
		allResistances = append(allResistances, pred.ResistanceLevels...)
	}

	// Cluster nearby levels
	supports := m.clusterLevels(allSupports)
	resistances := m.clusterLevels(allResistances)

	return supports, resistances
}

// clusterLevels groups nearby price levels
func (m *EnsembleModel) clusterLevels(levels []float64) []float64 {
	if len(levels) == 0 {
		return levels
	}

	// Sort levels
	sort.Float64s(levels)

	// Group levels that are within 0.5% of each other
	var clustered []float64
	currentCluster := []float64{levels[0]}

	for i := 1; i < len(levels); i++ {
		diff := math.Abs(levels[i] - levels[i-1]) / levels[i-1]
		if diff <= 0.005 { // 0.5% threshold
			currentCluster = append(currentCluster, levels[i])
		} else {
			// Add average of current cluster
			avg := 0.0
			for _, v := range currentCluster {
				avg += v
			}
			clustered = append(clustered, avg/float64(len(currentCluster)))
			
			// Start new cluster
			currentCluster = []float64{levels[i]}
		}
	}

	// Add final cluster
	if len(currentCluster) > 0 {
		avg := 0.0
		for _, v := range currentCluster {
			avg += v
		}
		clustered = append(clustered, avg/float64(len(currentCluster)))
	}

	return clustered
}

// combineSignals merges trading signals from all models
func (m *EnsembleModel) combineSignals(predictions []*PredictionOutput) []models.Signal {
	// Map to track unique signals and their combined strengths
	signalMap := make(map[string]*models.Signal)

	for i, pred := range predictions {
		modelID := m.getModelID(i)
		weight := m.weights[modelID]

		for _, signal := range pred.Signals {
			key := signal.Type // Use signal type as key
			
			if existing, exists := signalMap[key]; exists {
				// Update existing signal
				existing.Strength = (existing.Strength + signal.Strength*weight) / 2
				// Combine descriptions if different
				if existing.Description != signal.Description {
					existing.Description += "; " + signal.Description
				}
			} else {
				// Create new signal entry
				newSignal := models.Signal{
					Type:        signal.Type,
					Strength:    signal.Strength * weight,
					Description: signal.Description,
					CreatedAt:   signal.CreatedAt,
				}
				signalMap[key] = &newSignal
			}
		}
	}

	// Convert map back to slice
	var combinedSignals []models.Signal
	for _, signal := range signalMap {
		// Only include signals with meaningful strength
		if signal.Strength >= 0.2 { // 20% minimum strength threshold
			combinedSignals = append(combinedSignals, *signal)
		}
	}

	// Sort by strength
	sort.Slice(combinedSignals, func(i, j int) bool {
		return combinedSignals[i].Strength > combinedSignals[j].Strength
	})

	return combinedSignals
}

func (m *EnsembleModel) getModelID(index int) string {
	return fmt.Sprintf("model_%d", index)
}

// Validate implements the Model interface
func (m *EnsembleModel) Validate(ctx context.Context, data *ValidationData) (*ValidationResults, error) {
	var (
		results = make([]*ValidationResults, len(m.models))
		errors  = make([]error, len(m.models))
		wg      sync.WaitGroup
	)

	// Validate all models in parallel
	for i, model := range m.models {
		wg.Add(1)
		go func(idx int, mdl Model) {
			defer wg.Done()
			res, err := mdl.Validate(ctx, data)
			results[idx] = res
			errors[idx] = err
		}(i, model)
	}

	wg.Wait()

	// Check for errors
	for _, err := range errors {
		if err != nil {
			return nil, err
		}
	}

	// Combine validation results
	return m.combineValidationResults(results), nil
}

// combineValidationResults merges validation results from all models
func (m *EnsembleModel) combineValidationResults(results []*ValidationResults) *ValidationResults {
	combined := &ValidationResults{}
	totalWeight := 0.0

	for i, result := range results {
		modelID := m.getModelID(i)
		weight := m.weights[modelID]

		combined.RMSE += result.RMSE * weight
		combined.MAE += result.MAE * weight
		combined.Accuracy += result.Accuracy * weight
		combined.SharpeRatio += result.SharpeRatio * weight
		combined.MaxDrawdown += result.MaxDrawdown * weight
		combined.WinRate += result.WinRate * weight
		combined.ProfitFactor += result.ProfitFactor * weight

		totalWeight += weight
	}

	// Normalize by total weight
	if totalWeight > 0 {
		combined.RMSE /= totalWeight
		combined.MAE /= totalWeight
		combined.Accuracy /= totalWeight
		combined.SharpeRatio /= totalWeight
		combined.MaxDrawdown /= totalWeight
		combined.WinRate /= totalWeight
		combined.ProfitFactor /= totalWeight
	}

	return combined
}
