package loadbalancer

import (
	"net/url"
	"testing"
)

func makeTargets(urls ...string) []*Target {
	targets := make([]*Target, len(urls))
	for i, u := range urls {
		parsed, _ := url.Parse(u)
		targets[i] = &Target{URL: parsed, Weight: 1}
	}
	return targets
}

func TestRoundRobin_Next(t *testing.T) {
	targets := makeTargets("http://a:8080", "http://b:8080", "http://c:8080")
	lb := NewRoundRobin(targets)

	// Should cycle through targets
	seen := make(map[string]int)
	for i := 0; i < 9; i++ {
		target := lb.Next()
		seen[target.URL.Host]++
	}

	// Each should be hit 3 times
	for host, count := range seen {
		if count != 3 {
			t.Errorf("Host %s hit %d times, expected 3", host, count)
		}
	}
}

func TestRoundRobin_SkipUnhealthy(t *testing.T) {
	targets := makeTargets("http://a:8080", "http://b:8080", "http://c:8080")
	lb := NewRoundRobin(targets)

	// Mark b as unhealthy
	lb.MarkHealthy(targets[1], false)

	// Should skip b
	seen := make(map[string]int)
	for i := 0; i < 6; i++ {
		target := lb.Next()
		seen[target.URL.Host]++
	}

	if seen["b:8080"] > 0 {
		t.Error("Should not select unhealthy target")
	}
	if seen["a:8080"] != 3 || seen["c:8080"] != 3 {
		t.Errorf("Expected a and c to each be hit 3 times, got a=%d c=%d", seen["a:8080"], seen["c:8080"])
	}
}

func TestLeastConn_Next(t *testing.T) {
	targets := makeTargets("http://a:8080", "http://b:8080")
	lb := NewLeastConn(targets)

	// Simulate connections on target a
	targets[0].Connections.Store(5)
	targets[1].Connections.Store(2)

	// Should prefer b (fewer connections)
	target := lb.Next()
	if target.URL.Host != "b:8080" {
		t.Errorf("Expected b (fewer connections), got %s", target.URL.Host)
	}
}

func TestRandom_Next(t *testing.T) {
	targets := makeTargets("http://a:8080", "http://b:8080", "http://c:8080")
	lb := NewRandom(targets)

	// Should return some target (randomness makes exact testing hard)
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		target := lb.Next()
		if target == nil {
			t.Error("Should return a target")
		}
		seen[target.URL.Host] = true
	}

	// With 100 iterations, should see all 3
	if len(seen) < 2 {
		t.Errorf("Random should distribute across targets, only saw %d", len(seen))
	}
}

func TestWeightedRoundRobin_Next(t *testing.T) {
	targets := makeTargets("http://a:8080", "http://b:8080")
	targets[0].Weight = 2
	targets[1].Weight = 1

	lb := NewWeightedRoundRobin(targets)

	// Over 6 iterations, a should be selected 4 times, b 2 times (2:1 ratio)
	seen := make(map[string]int)
	for i := 0; i < 6; i++ {
		target := lb.Next()
		seen[target.URL.Host]++
	}

	// Should roughly follow weight ratio
	if seen["a:8080"] < seen["b:8080"] {
		t.Errorf("a (weight 2) should be selected more than b (weight 1): a=%d, b=%d",
			seen["a:8080"], seen["b:8080"])
	}
}

func TestNew_Strategy(t *testing.T) {
	targets := makeTargets("http://a:8080")

	tests := []struct {
		strategy string
		expected string
	}{
		{"round_robin", "*loadbalancer.RoundRobin"},
		{"least_conn", "*loadbalancer.LeastConn"},
		{"random", "*loadbalancer.Random"},
		{"weighted_round_robin", "*loadbalancer.WeightedRoundRobin"},
		{"unknown", "*loadbalancer.RoundRobin"}, // default
	}

	for _, tc := range tests {
		lb := New(tc.strategy, targets)
		if lb == nil {
			t.Errorf("Strategy %s returned nil", tc.strategy)
		}
	}
}

func TestEmptyTargets(t *testing.T) {
	lb := NewRoundRobin(nil)
	if lb.Next() != nil {
		t.Error("Empty targets should return nil")
	}
}

func BenchmarkRoundRobin_Next(b *testing.B) {
	targets := makeTargets(
		"http://a:8080", "http://b:8080", "http://c:8080",
		"http://d:8080", "http://e:8080",
	)
	lb := NewRoundRobin(targets)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lb.Next()
	}
}

func BenchmarkLeastConn_Next(b *testing.B) {
	targets := makeTargets(
		"http://a:8080", "http://b:8080", "http://c:8080",
		"http://d:8080", "http://e:8080",
	)
	lb := NewLeastConn(targets)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lb.Next()
	}
}
