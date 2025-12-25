package ratelimit

import (
	"sync"
	"testing"
	"time"
)

func TestTokenBucket_Allow(t *testing.T) {
	tb := NewTokenBucket(10, 10) // 10 rps, burst of 10

	// Should allow first 10 requests
	for i := 0; i < 10; i++ {
		if !tb.Allow() {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 11th should be denied
	if tb.Allow() {
		t.Error("Request 11 should be denied (no tokens)")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	tb := NewTokenBucket(100, 10) // 100 rps, burst of 10

	// Consume all tokens
	for i := 0; i < 10; i++ {
		tb.Allow()
	}

	// Wait for refill (100ms = ~10 tokens at 100rps)
	time.Sleep(100 * time.Millisecond)

	// Should now have tokens again
	if !tb.Allow() {
		t.Error("Should have refilled tokens")
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(Config{
		DefaultRPS:   10,
		DefaultBurst: 10,
	})
	defer rl.Stop()

	// First 10 requests should pass
	for i := 0; i < 10; i++ {
		if !rl.Allow("test-key") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 11th should fail
	if rl.Allow("test-key") {
		t.Error("Request 11 should be denied")
	}
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	rl := NewRateLimiter(Config{
		DefaultRPS:   5,
		DefaultBurst: 5,
	})
	defer rl.Stop()

	// Exhaust key1
	for i := 0; i < 5; i++ {
		rl.Allow("key1")
	}

	// key2 should still work
	if !rl.Allow("key2") {
		t.Error("key2 should have its own bucket")
	}
}

func TestRateLimiter_CustomLimits(t *testing.T) {
	rl := NewRateLimiter(Config{
		DefaultRPS:   10,
		DefaultBurst: 10,
	})
	defer rl.Stop()

	// Set custom limits for premium key
	rl.SetLimits("premium", 100, 100)

	// Premium key should have 100 tokens
	for i := 0; i < 50; i++ {
		if !rl.Allow("premium") {
			t.Errorf("Premium request %d should be allowed", i+1)
		}
	}
}

func TestRateLimiter_Concurrent(t *testing.T) {
	rl := NewRateLimiter(Config{
		DefaultRPS:   1000,
		DefaultBurst: 1000,
	})
	defer rl.Stop()

	var wg sync.WaitGroup
	allowed := make(chan bool, 10000)

	// Concurrent requests
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				allowed <- rl.Allow("concurrent-key")
			}
		}()
	}

	wg.Wait()
	close(allowed)

	// Count allowed requests
	count := 0
	for a := range allowed {
		if a {
			count++
		}
	}

	// Should have allowed ~1000 (burst size)
	if count < 900 || count > 1100 {
		t.Errorf("Expected ~1000 allowed requests, got %d", count)
	}
}
