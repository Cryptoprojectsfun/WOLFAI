package monitoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// Health check status constants
const (
	StatusUp      = "UP"
	StatusDown    = "DOWN"
	StatusWarning = "WARNING"
)

// HealthChecker manages system health checks
type HealthChecker struct {
	db            *sql.DB
	services      map[string]HealthCheckFunc
	lastResults   map[string]*CheckResult
	checkInterval time.Duration
	mu            sync.RWMutex
}

// HealthCheckFunc defines a health check function
type HealthCheckFunc func(context.Context) *CheckResult

// CheckResult represents the result of a health check
type CheckResult struct {
	Status      string                 `json:"status"`
	Component   string                 `json:"component"`
	Details     map[string]interface{} `json:"details,omitempty"`
	LastChecked time.Time             `json:"last_checked"`
	Error       string                 `json:"error,omitempty"`
}

// SystemHealth represents overall system health
type SystemHealth struct {
	Status      string                 `json:"status"`
	Components  map[string]*CheckResult `json:"components"`
	Timestamp   time.Time              `json:"timestamp"`
}

// ResourceUsage tracks system resource utilization
type ResourceUsage struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage struct {
		Total     uint64  `json:"total"`
		Used      uint64  `json:"used"`
		Free      uint64  `json:"free"`
		UsagePerc float64 `json:"usage_percentage"`
	} `json:"memory_usage"`
	GoroutineCount int    `json:"goroutine_count"`
	ThreadCount    int    `json:"thread_count"`
	DiskUsage     struct {
		Total     uint64  `json:"total"`
		Used      uint64  `json:"used"`
		Free      uint64  `json:"free"`
		UsagePerc float64 `json:"usage_percentage"`
	} `json:"disk_usage"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *sql.DB, interval time.Duration) *HealthChecker {
	hc := &HealthChecker{
		db:            db,
		services:      make(map[string]HealthCheckFunc),
		lastResults:   make(map[string]*CheckResult),
		checkInterval: interval,
	}

	// Register default health checks
	hc.RegisterCheck("database", hc.DatabaseCheck)
	hc.RegisterCheck("memory", hc.MemoryCheck)
	hc.RegisterCheck("goroutines", hc.GoroutineCheck)
	hc.RegisterCheck("disk", hc.DiskCheck)

	return hc
}

// RegisterCheck adds a new health check
func (h *HealthChecker) RegisterCheck(name string, check HealthCheckFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.services[name] = check
}

// StartChecks begins periodic health checking
func (h *HealthChecker) StartChecks(ctx context.Context) {
	ticker := time.NewTicker(h.checkInterval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				h.performChecks(ctx)
			}
		}
	}()
}

// performChecks runs all registered health checks
func (h *HealthChecker) performChecks(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for name, check := range h.services {
		result := check(ctx)
		result.LastChecked = time.Now()
		h.lastResults[name] = result
	}
}

// GetHealth returns current system health status
func (h *HealthChecker) GetHealth() *SystemHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()

	health := &SystemHealth{
		Status:     StatusUp,
		Components: make(map[string]*CheckResult),
		Timestamp:  time.Now(),
	}

	// Copy last results
	for name, result := range h.lastResults {
		health.Components[name] = result
		if result.Status == StatusDown {
			health.Status = StatusDown
		} else if result.Status == StatusWarning && health.Status != StatusDown {
			health.Status = StatusWarning
		}
	}

	return health
}

// DatabaseCheck checks database connectivity and performance
func (h *HealthChecker) DatabaseCheck(ctx context.Context) *CheckResult {
	result := &CheckResult{
		Status:    StatusUp,
		Component: "database",
		Details:   make(map[string]interface{}),
	}

	// Check connection
	err := h.db.PingContext(ctx)
	if err != nil {
		result.Status = StatusDown
		result.Error = fmt.Sprintf("Database connection failed: %v", err)
		return result
	}

	// Get database stats
	stats := h.db.Stats()
	result.Details = map[string]interface{}{
		"open_connections":    stats.OpenConnections,
		"in_use":             stats.InUse,
		"idle":               stats.Idle,
		"wait_count":         stats.WaitCount,
		"wait_duration":      stats.WaitDuration.Milliseconds(),
		"max_idle_closed":    stats.MaxIdleClosed,
		"max_lifetime_closed": stats.MaxLifetimeClosed,
	}

	// Check connection pool health
	if float64(stats.InUse)/float64(stats.OpenConnections) > 0.8 {
		result.Status = StatusWarning
		result.Error = "High connection pool utilization"
	}

	// Check for long-running queries (example query)
	var longQueries int
	err = h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) 
		FROM pg_stat_activity 
		WHERE state = 'active' 
		AND NOW() - query_start > interval '30 seconds'
	`).Scan(&longQueries)

	if err == nil {
		result.Details["long_running_queries"] = longQueries
		if longQueries > 5 {
			result.Status = StatusWarning
			result.Error = "High number of long-running queries"
		}
	}

	return result
}

// MemoryCheck checks system memory usage
func (h *HealthChecker) MemoryCheck(ctx context.Context) *CheckResult {
	result := &CheckResult{
		Status:    StatusUp,
		Component: "memory",
		Details:   make(map[string]interface{}),
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	result.Details = map[string]interface{}{
		"heap_alloc":     memStats.HeapAlloc,
		"heap_sys":       memStats.HeapSys,
		"heap_idle":      memStats.HeapIdle,
		"heap_inuse":     memStats.HeapInuse,
		"heap_released":  memStats.HeapReleased,
		"heap_objects":   memStats.HeapObjects,
		"gc_pause_total": memStats.PauseTotalNs,
		"gc_num":         memStats.NumGC,
	}

	// Check memory pressure
	memoryUsage := float64(memStats.HeapInuse) / float64(memStats.HeapSys)
	if memoryUsage > 0.9 {
		result.Status = StatusWarning
		result.Error = "High memory usage"
	}

	return result
}

// GoroutineCheck monitors goroutine count
func (h *HealthChecker) GoroutineCheck(ctx context.Context) *CheckResult {
	result := &CheckResult{
		Status:    StatusUp,
		Component: "goroutines",
		Details:   make(map[string]interface{}),
	}

	goroutineCount := runtime.NumGoroutine()
	result.Details["count"] = goroutineCount

	// Alert on high goroutine count
	if goroutineCount > 10000 {
		result.Status = StatusWarning
		result.Error = "High number of goroutines"
	}

	return result
}

// DiskCheck monitors disk usage
func (h *HealthChecker) DiskCheck(ctx context.Context) *CheckResult {
	result := &CheckResult{
		Status:    StatusUp,
		Component: "disk",
		Details:   make(map[string]interface{}),
	}

	// Get disk usage stats (example implementation)
	diskInfo, err := getDiskUsage("/")
	if err != nil {
		result.Status = StatusWarning
		result.Error = fmt.Sprintf("Failed to get disk usage: %v", err)
		return result
	}

	result.Details = map[string]interface{}{
		"total":             diskInfo.Total,
		"used":              diskInfo.Used,
		"free":              diskInfo.Free,
		"usage_percentage":  diskInfo.UsagePerc,
	}

	// Alert on high disk usage
	if diskInfo.UsagePerc > 85 {
		result.Status = StatusWarning
		result.Error = "High disk usage"
	}

	return result
}

// HTTPHandler returns a health check HTTP handler
func (h *HealthChecker) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := h.GetHealth()
		
		w.Header().Set("Content-Type", "application/json")
		if health.Status != StatusUp {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		json.NewEncoder(w).Encode(health)
	}
}

// Helper functions

func getDiskUsage(path string) (*DiskUsage, error) {
	// Implementation would depend on OS
	// This is a placeholder
	return &DiskUsage{
		Total:     100 * 1024 * 1024 * 1024, // 100GB
		Used:      60 * 1024 * 1024 * 1024,  // 60GB
		Free:      40 * 1024 * 1024 * 1024,  // 40GB
		UsagePerc: 60.0,
	}, nil
}

type DiskUsage struct {
	Total     uint64
	Used      uint64
	Free      uint64
	UsagePerc float64
}

// Custom health checks

// ModelHealthCheck checks AI model health
type ModelHealthCheck struct {
	modelID         string
	predictionCount int64
	lastPrediction  time.Time
	avgLatency      time.Duration
	accuracy        float64
}

func NewModelHealthCheck(modelID string) HealthCheckFunc {
	check := &ModelHealthCheck{
		modelID: modelID,
	}
	
	return check.Check
}

func (m *ModelHealthCheck) Check(ctx context.Context) *CheckResult {
	result := &CheckResult{
		Status:    StatusUp,
		Component: fmt.Sprintf("model-%s", m.modelID),
		Details:   make(map[string]interface{}),
	}

	result.Details = map[string]interface{}{
		"prediction_count": m.predictionCount,
		"last_prediction": m.lastPrediction,
		"avg_latency_ms": m.avgLatency.Milliseconds(),
		"accuracy": m.accuracy,
	}

	// Check model health
	if time.Since(m.lastPrediction) > 5*time.Minute {
		result.Status = StatusWarning
		result.Error = "No recent predictions"
	}

	if m.avgLatency > 500*time.Millisecond {
		result.Status = StatusWarning
		result.Error = "High prediction latency"
	}

	if m.accuracy < 0.7 {
		result.Status = StatusWarning
		result.Error = "Low model accuracy"
	}

	return result
}

// UpdateModelMetrics updates model health metrics
func (m *ModelHealthCheck) UpdateModelMetrics(latency time.Duration, accuracy float64) {
	m.predictionCount++
	m.lastPrediction = time.Now()
	m.avgLatency = (m.avgLatency + latency) / 2
	m.accuracy = (m.accuracy + accuracy) / 2
}
