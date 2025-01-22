package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps the underlying zap logger with additional functionality
type Logger struct {
	*zap.SugaredLogger
	metrics       *Metrics
	contextFields map[string]interface{}
	mu           sync.RWMutex
}

// Metrics tracks various logging metrics
type Metrics struct {
	ErrorCount   int64
	WarningCount int64
	InfoCount    int64
	DebugCount   int64
	
	// Performance metrics
	ResponseTimes  []time.Duration
	ErrorRates     map[string]int64
	SlowQueries    []SlowQuery
	MemoryUsage    []MemorySnapshot
	mu            sync.RWMutex
}

// SlowQuery represents a slow database query
type SlowQuery struct {
	Query      string
	Duration   time.Duration
	Timestamp  time.Time
}

// MemorySnapshot represents memory usage at a point in time
type MemorySnapshot struct {
	Timestamp     time.Time
	HeapAlloc     uint64
	HeapInUse     uint64
	HeapIdle      uint64
	HeapReleased  uint64
}

// Config represents logger configuration
type Config struct {
	Level      string
	Format     string
	Output     string
	MaxSize    int
	MaxAge     int
	MaxBackups int
	Compress   bool
}

func (l *Logger) LogMemoryStats() {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	snapshot := MemorySnapshot{
		Timestamp:     time.Now(),
		HeapAlloc:     stats.HeapAlloc,
		HeapInUse:     stats.HeapInuse,
		HeapIdle:      stats.HeapIdle,
		HeapReleased:  stats.HeapReleased,
	}

	l.WithFields(map[string]interface{}{
		"heap_alloc":     formatBytes(stats.HeapAlloc),
		"heap_in_use":    formatBytes(stats.HeapInuse),
		"heap_idle":      formatBytes(stats.HeapIdle),
		"heap_released":  formatBytes(stats.HeapReleased),
	}).Info("Memory stats")

	l.metrics.mu.Lock()
	l.metrics.MemoryUsage = append(l.metrics.MemoryUsage, snapshot)
	l.metrics.mu.Unlock()

	// Alert on high memory usage
	if float64(stats.HeapInuse)/float64(stats.HeapIdle) > 0.9 {
		l.Warn("High memory usage detected")
	}
}

// LogModel logs model-related events
func (l *Logger) LogModel(modelID string, operation string, duration time.Duration, err error) {
	fields := map[string]interface{}{
		"model_id":  modelID,
		"operation": operation,
		"duration":  duration.Milliseconds(),
	}

	if err != nil {
		fields["error"] = err.Error()
		l.WithFields(fields).Error("Model operation failed")
	} else {
		l.WithFields(fields).Info("Model operation completed")
	}
}

// LogPrediction logs model predictions
func (l *Logger) LogPrediction(modelID string, confidence float64, duration time.Duration) {
	l.WithFields(map[string]interface{}{
		"model_id":   modelID,
		"confidence": confidence,
		"duration":   duration.Milliseconds(),
	}).Info("Prediction generated")
}

// LogTraining logs model training events
func (l *Logger) LogTraining(modelID string, epoch int, loss float64, metrics map[string]float64) {
	fields := map[string]interface{}{
		"model_id": modelID,
		"epoch":    epoch,
		"loss":     loss,
	}
	
	for k, v := range metrics {
		fields[k] = v
	}

	l.WithFields(fields).Info("Training epoch completed")
}

// LogValidation logs model validation results
func (l *Logger) LogValidation(modelID string, metrics map[string]float64) {
	fields := map[string]interface{}{
		"model_id": modelID,
	}
	
	for k, v := range metrics {
		fields[k] = v
	}

	l.WithFields(fields).Info("Model validation completed")
}

// LogPortfolioOperation logs portfolio-related operations
func (l *Logger) LogPortfolioOperation(portfolioID string, operation string, amount float64, err error) {
	fields := map[string]interface{}{
		"portfolio_id": portfolioID,
		"operation":    operation,
		"amount":       amount,
	}

	if err != nil {
		fields["error"] = err.Error()
		l.WithFields(fields).Error("Portfolio operation failed")
	} else {
		l.WithFields(fields).Info("Portfolio operation completed")
	}
}

// LogAPIRequest logs external API requests
func (l *Logger) LogAPIRequest(service string, endpoint string, duration time.Duration, err error) {
	fields := map[string]interface{}{
		"service":   service,
		"endpoint":  endpoint,
		"duration":  duration.Milliseconds(),
	}

	if err != nil {
		fields["error"] = err.Error()
		l.WithFields(fields).Error("API request failed")
		l.trackErrorRate("api_error")
	} else {
		l.WithFields(fields).Info("API request completed")
	}
}

// Metric tracking methods
func (l *Logger) trackResponseTime(duration time.Duration) {
	l.metrics.mu.Lock()
	defer l.metrics.mu.Unlock()
	l.metrics.ResponseTimes = append(l.metrics.ResponseTimes, duration)
}

func (l *Logger) trackErrorRate(errorType string) {
	l.metrics.mu.Lock()
	defer l.metrics.mu.Unlock()
	l.metrics.ErrorRates[errorType]++
}

// GetMetrics returns current metrics
func (l *Logger) GetMetrics() *MetricsSnapshot {
	l.metrics.mu.RLock()
	defer l.metrics.mu.RUnlock()

	return &MetricsSnapshot{
		ErrorCount:     l.metrics.ErrorCount,
		WarningCount:   l.metrics.WarningCount,
		InfoCount:      l.metrics.InfoCount,
		DebugCount:     l.metrics.DebugCount,
		AvgResponseTime: calculateAverageResponseTime(l.metrics.ResponseTimes),
		ErrorRates:     l.metrics.ErrorRates,
		SlowQueryCount: len(l.metrics.SlowQueries),
		MemoryStats:    getLatestMemoryStats(l.metrics.MemoryUsage),
	}
}

// MetricsSnapshot represents a point-in-time snapshot of metrics
type MetricsSnapshot struct {
	ErrorCount      int64
	WarningCount    int64
	InfoCount       int64
	DebugCount      int64
	AvgResponseTime float64
	ErrorRates      map[string]int64
	SlowQueryCount  int
	MemoryStats     *MemorySnapshot
}

// Helper functions
func extractContextFields(ctx context.Context) map[string]interface{} {
	fields := make(map[string]interface{})
	
	// Extract request ID if present
	if requestID, ok := ctx.Value("request_id").(string); ok {
		fields["request_id"] = requestID
	}
	
	// Extract user ID if present
	if userID, ok := ctx.Value("user_id").(string); ok {
		fields["user_id"] = userID
	}

	return fields
}

func fieldsToArgs(fields map[string]interface{}) []interface{} {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return args
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func calculateAverageResponseTime(times []time.Duration) float64 {
	if len(times) == 0 {
		return 0
	}
	var total time.Duration
	for _, t := range times {
		total += t
	}
	return float64(total.Milliseconds()) / float64(len(times))
}

func getLatestMemoryStats(snapshots []MemorySnapshot) *MemorySnapshot {
	if len(snapshots) == 0 {
		return nil
	}
	return &snapshots[len(snapshots)-1]
}
