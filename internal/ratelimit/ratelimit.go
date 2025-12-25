package ratelimit

import (
	"sync"
	"time"
)

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(rps int, burst int) *TokenBucket {
	return &TokenBucket{
		tokens:     float64(burst),
		maxTokens:  float64(burst),
		refillRate: float64(rps),
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed and consumes a token if so
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// refill adds tokens based on elapsed time
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	tb.lastRefill = now
}

// RateLimiter manages rate limiting for multiple keys
type RateLimiter struct {
	buckets       map[string]*TokenBucket
	defaultRPS    int
	defaultBurst  int
	mu            sync.RWMutex
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// Config for creating a new RateLimiter
type Config struct {
	DefaultRPS      int
	DefaultBurst    int
	CleanupInterval time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(cfg Config) *RateLimiter {
	rl := &RateLimiter{
		buckets:      make(map[string]*TokenBucket),
		defaultRPS:   cfg.DefaultRPS,
		defaultBurst: cfg.DefaultBurst,
		stopCleanup:  make(chan struct{}),
	}

	if cfg.CleanupInterval > 0 {
		rl.cleanupTicker = time.NewTicker(cfg.CleanupInterval)
		go rl.cleanup()
	}

	return rl
}

// Allow checks if a request with the given key is allowed
func (rl *RateLimiter) Allow(key string) bool {
	return rl.AllowWithLimits(key, rl.defaultRPS, rl.defaultBurst)
}

// AllowWithLimits checks if a request is allowed with custom limits
func (rl *RateLimiter) AllowWithLimits(key string, rps, burst int) bool {
	rl.mu.RLock()
	bucket, exists := rl.buckets[key]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		// Double-check after acquiring write lock
		if bucket, exists = rl.buckets[key]; !exists {
			bucket = NewTokenBucket(rps, burst)
			rl.buckets[key] = bucket
		}
		rl.mu.Unlock()
	}

	return bucket.Allow()
}

// SetLimits updates or creates a bucket with specific limits
func (rl *RateLimiter) SetLimits(key string, rps, burst int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.buckets[key] = NewTokenBucket(rps, burst)
}

// cleanup removes stale buckets periodically
func (rl *RateLimiter) cleanup() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.mu.Lock()
			now := time.Now()
			for key, bucket := range rl.buckets {
				bucket.mu.Lock()
				// Remove buckets that haven't been used in 10 minutes
				if now.Sub(bucket.lastRefill) > 10*time.Minute {
					delete(rl.buckets, key)
				}
				bucket.mu.Unlock()
			}
			rl.mu.Unlock()
		case <-rl.stopCleanup:
			return
		}
	}
}

// Stop stops the rate limiter cleanup goroutine
func (rl *RateLimiter) Stop() {
	if rl.cleanupTicker != nil {
		rl.cleanupTicker.Stop()
		close(rl.stopCleanup)
	}
}

// Stats returns current statistics
func (rl *RateLimiter) Stats() map[string]float64 {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	stats := make(map[string]float64)
	for key, bucket := range rl.buckets {
		bucket.mu.Lock()
		stats[key] = bucket.tokens
		bucket.mu.Unlock()
	}
	return stats
}
