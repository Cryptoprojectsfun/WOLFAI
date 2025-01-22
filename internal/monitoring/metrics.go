package monitoring

import (
	"encoding/json"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics represents the monitoring system
type Metrics struct {
	// HTTP metrics
	requestDuration *prometheus.HistogramVec
	requestCount    *prometheus.CounterVec
	errorCount      *prometheus.CounterVec

	// Model metrics
	modelPredictionDuration *prometheus.HistogramVec
	modelPredictionCount    *prometheus.CounterVec
	modelConfidence        *prometheus.HistogramVec

	// Portfolio metrics
	portfolioValue         *prometheus.GaugeVec
	portfolioReturnRate    *prometheus.GaugeVec
	portfolioTradeCount    *prometheus.CounterVec

	// System metrics
	memoryUsage    *prometheus.GaugeVec
	goroutineCount prometheus.Gauge
	cpuUsage       prometheus.Gauge

	// Custom metrics
	customMetrics map[string]prometheus.Collector
	mu           sync.RWMutex
}

// NewMetrics creates a new metrics collector
func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		// HTTP metrics
		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "Duration of HTTP requests",
				Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"handler", "method", "status"},
		),

		requestCount: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"handler", "method", "status"},
		),

		errorCount: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "error_count_total",
				Help:      "Total number of errors",
			},
			[]string{"type", "code"},
		),

		// Model metrics
		modelPredictionDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "model_prediction_duration_seconds",
				Help:      "Duration of model predictions",
				Buckets:   []float64{.01, .025, .05, .1, .25, .5, 1, 2.5, 5},
			},
			[]string{"model_id", "type"},
		),

		modelPredictionCount: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "model_predictions_total",
				Help:      "Total number of model predictions",
			},
			[]string{"model_id", "type"},
		),

		modelConfidence: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "model_confidence",
				Help:      "Model prediction confidence",
				Buckets:   []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9},
			},
			[]string{"model_id"},
		),

		// Portfolio metrics
		portfolioValue: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "portfolio_value",
				Help:      "Current portfolio value",
			},
			[]string{"portfolio_id", "currency"},
		),

		portfolioReturnRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "portfolio_return_rate",
				Help:      "Portfolio return rate",
			},
			[]string{"portfolio_id", "timeframe"},
		),

		portfolioTradeCount: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "portfolio_trades_total",
				Help:      "Total number of portfolio trades",
			},
			[]string{"portfolio_id", "type"},
		),

		// System metrics
		memoryUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "memory_usage_bytes",
				Help:      "Current memory usage",
			},
			[]string{"type"},
		),

		goroutineCount: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "goroutine_count",
				Help:      "Number of goroutines",
			},
		),

		cpuUsage: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "cpu_usage_percent",
				Help:      "CPU usage percentage",
			},
		),

		customMetrics: make(map[string]prometheus.Collector),
	}

	return m
}

// ObserveRequest records HTTP request metrics
func (m *Metrics) ObserveRequest(handler, method string, status int, duration time.Duration) {
	m.requestDuration.WithLabelValues(handler, method, string(status)).Observe(duration.Seconds())
	m.requestCount.WithLabelValues(handler, method, string(status)).Inc()
}

// ObserveError records error metrics
func (m *Metrics) ObserveError(errorType, errorCode string) {
	m.errorCount.WithLabelValues(errorType, errorCode).Inc()
}

// ObserveModelPrediction records model prediction metrics
func (m *Metrics) ObserveModelPrediction(modelID, predictionType string, duration time.Duration, confidence float64) {
	m.modelPredictionDuration.WithLabelValues(modelID, predictionType).Observe(duration.Seconds())
	m.modelPredictionCount.WithLabelValues(modelID, predictionType).Inc()
	m.modelConfidence.WithLabelValues(modelID).Observe(confidence)
}

// UpdatePortfolioMetrics updates portfolio-related metrics
func (m *Metrics) UpdatePortfolioMetrics(portfolioID string, value float64, currency string, returns map[string]float64) {
	m.portfolioValue.WithLabelValues(portfolioID, currency).Set(value)
	
	for timeframe, returnRate := range returns {
		m.portfolioReturnRate.WithLabelValues(portfolioID, timeframe).Set(returnRate)
	}
}

// RecordTrade records a trade for a portfolio
func (m *Metrics) RecordTrade(portfolioID, tradeType string) {
	m.portfolioTradeCount.WithLabelValues(portfolioID, tradeType).Inc()
}

// UpdateSystemMetrics updates system-level metrics
func (m *Metrics) UpdateSystemMetrics() {
	// Update memory metrics
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	
	m.memoryUsage.WithLabelValues("heap_alloc").Set(float64(mem.HeapAlloc))
	m.memoryUsage.WithLabelValues("heap_inuse").Set(float64(mem.HeapInuse))
	m.memoryUsage.WithLabelValues("heap_idle").Set(float64(mem.HeapIdle))
	m.memoryUsage.WithLabelValues("heap_released").Set(float64(mem.HeapReleased))

	// Update goroutine count
	m.goroutineCount.Set(float64(runtime.NumGoroutine()))
}

// RegisterCustomMetric registers a custom prometheus metric
func (m *Metrics) RegisterCustomMetric(name string, metric prometheus.Collector) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.customMetrics[name]; exists {
		return fmt.Errorf("metric %s already registered", name)
	}

	if err := prometheus.Register(metric); err != nil {
		return fmt.Errorf("failed to register metric %s: %v", name, err)
	}

	m.customMetrics[name] = metric
	return nil
}

// GetCustomMetric retrieves a custom metric by name
func (m *Metrics) GetCustomMetric(name string) (prometheus.Collector, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	metric, exists := m.customMetrics[name]
	return metric, exists
}

// MetricsHandler returns an HTTP handler for exposing metrics
func (m *Metrics) MetricsHandler() http.Handler {
	return prometheus.Handler()
}

// StartMetricsCollection starts periodic collection of system metrics
func (m *Metrics) StartMetricsCollection(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			m.UpdateSystemMetrics()
		}
	}()
}

// GetMetricsSnapshot returns a snapshot of current metrics
type MetricsSnapshot struct {
	RequestCount int64                  `json:"request_count"`
	ErrorCount   int64                  `json:"error_count"`
	SystemStats  SystemStats            `json:"system_stats"`
	ModelStats   map[string]ModelStats  `json:"model_stats"`
}

type SystemStats struct {
	MemoryUsage    uint64  `json:"memory_usage"`
	GoroutineCount int     `json:"goroutine_count"`
	CPUUsage       float64 `json:"cpu_usage"`
}

type ModelStats struct {
	PredictionCount  int64   `json:"prediction_count"`
	AvgDuration      float64 `json:"avg_duration"`
	AvgConfidence    float64 `json:"avg_confidence"`
}

func (m *Metrics) GetSnapshot() (*MetricsSnapshot, error) {
	snapshot := &MetricsSnapshot{
		ModelStats: make(map[string]ModelStats),
	}

	// Gather system stats
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	snapshot.SystemStats = SystemStats{
		MemoryUsage:    mem.HeapAlloc,
		GoroutineCount: runtime.NumGoroutine(),
		CPUUsage:       getCPUUsage(),
	}

	return snapshot, nil
}

// Helper function to get CPU usage
func getCPUUsage() float64 {
	// Implementation depends on the OS and available metrics
	// This is a placeholder
	return 0.0
}
