package middleware

import (
    "net/http"
    "sync"
    "time"
    "golang.org/x/time/rate"
)

type RateLimiter struct {
    visitors map[string]*visitor
    mu       sync.RWMutex
    limit    rate.Limit
    burst    int
}

type visitor struct {
    limiter  *rate.Limiter
    lastSeen time.Time
}

func NewRateLimiter(rps float64, burst int) *RateLimiter {
    return &RateLimiter{
        visitors: make(map[string]*visitor),
        limit:    rate.Limit(rps),
        burst:    burst,
    }
}

func (rl *RateLimiter) addVisitor(ip string) *rate.Limiter {
    limiter := rate.NewLimiter(rl.limit, rl.burst)
    rl.mu.Lock()
    rl.visitors[ip] = &visitor{limiter: limiter, lastSeen: time.Now()}
    rl.mu.Unlock()
    return limiter
}

func (rl *RateLimiter) getVisitor(ip string) *rate.Limiter {
    rl.mu.Lock()
    v, exists := rl.visitors[ip]
    if !exists {
        rl.mu.Unlock()
        return rl.addVisitor(ip)
    }

    v.lastSeen = time.Now()
    rl.mu.Unlock()
    return v.limiter
}

func (rl *RateLimiter) cleanupVisitors() {
    for {
        time.Sleep(time.Minute)
        rl.mu.Lock()
        for ip, v := range rl.visitors {
            if time.Since(v.lastSeen) > 3*time.Minute {
                delete(rl.visitors, ip)
            }
        }
        rl.mu.Unlock()
    }
}

func (rl *RateLimiter) RateLimit(next http.Handler) http.Handler {
    go rl.cleanupVisitors()
    
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ip := r.RemoteAddr
        limiter := rl.getVisitor(ip)
        if !limiter.Allow() {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}

func APIRateLimit(requestsPerSecond float64, burstSize int) func(http.Handler) http.Handler {
    limiter := NewRateLimiter(requestsPerSecond, burstSize)
    return limiter.RateLimit
}
