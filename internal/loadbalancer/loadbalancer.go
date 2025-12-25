package loadbalancer

import (
	"math/rand"
	"net/url"
	"sync"
	"sync/atomic"
)

type Target struct {
	URL         *url.URL
	Weight      int
	Healthy     atomic.Bool
	Connections atomic.Int64
}

type LoadBalancer interface {
	Next() *Target
	Targets() []*Target
	MarkHealthy(target *Target, healthy bool)
}

type RoundRobin struct {
	targets []*Target
	current atomic.Uint64
	mu      sync.RWMutex
}

func NewRoundRobin(targets []*Target) *RoundRobin {
	for _, t := range targets {
		t.Healthy.Store(true)
	}

	return &RoundRobin{targets: targets}
}

func (rr *RoundRobin) Next() *Target {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	if len(rr.targets) == 0 {
		return nil
	}

	n := len(rr.targets)
	for i := 0; i < n; i++ {
		idx := rr.current.Add(1) % uint64(n)
		target := rr.targets[idx]
		if target.Healthy.Load() {
			return target
		}
	}

	return rr.targets[0]
}

func (rr *RoundRobin) Targets() []*Target {
	rr.mu.RLock()
	defer rr.mu.RUnlock()
	return rr.targets
}

func (rr *RoundRobin) MarkHealthy(target *Target, healthy bool) {
	target.Healthy.Store(healthy)
}

type LeastConn struct {
	targets []*Target
	mu      sync.RWMutex
}

func NewLeastConn(targets []*Target) *LeastConn {
	for _, t := range targets {
		t.Healthy.Store(true)
	}

	return &LeastConn{targets: targets}
}

func (lc *LeastConn) Next() *Target {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	if len(lc.targets) == 0 {
		return nil
	}

	var best *Target
	var minConn int64 = -1

	for _, t := range lc.targets {
		if !t.Healthy.Load() {
			continue
		}

		conn := t.Connections.Load()
		if minConn < 0 || conn < minConn {
			minConn = conn
			best = t
		}
	}

	if best == nil {
		return lc.targets[0]
	}
	return best
}

func (lc *LeastConn) Targets() []*Target {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	return lc.targets
}

func (lc *LeastConn) MarkHealthy(target *Target, healthy bool) {
	target.Healthy.Store(healthy)
}

type Random struct {
	targets []*Target
	mu      sync.RWMutex
}

func NewRandom(targets []*Target) *Random {
	for _, t := range targets {
		t.Healthy.Store(true)
	}
	return &Random{targets: targets}
}

func (r *Random) Next() *Target {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.targets) == 0 {
		return nil
	}

	healthy := make([]*Target, 0, len(r.targets))
	for _, t := range r.targets {
		if t.Healthy.Load() {
			healthy = append(healthy, t)
		}
	}

	if len(healthy) == 0 {
		return r.targets[rand.Intn(len(r.targets))]
	}

	return healthy[rand.Intn(len(healthy))]
}

func (r *Random) Targets() []*Target {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.targets
}

func (r *Random) MarkHealthy(target *Target, healthy bool) {
	target.Healthy.Store(healthy)
}

type WeightedRoundRobin struct {
	targets       []*Target
	weights       []int
	currentWeight int
	maxWeight     int
	gcd           int
	current       int
	mu            sync.RWMutex
}

func NewWeightedRoundRobin(targets []*Target) *WeightedRoundRobin {
	for _, t := range targets {
		t.Healthy.Store(true)
	}

	weights := make([]int, len(targets))
	maxWeight := 0

	for i, t := range targets {
		w := t.Weight
		if w <= 0 {
			w = 1
		}
		weights[i] = w
		if w > maxWeight {
			maxWeight = w
		}
	}

	return &WeightedRoundRobin{
		targets:   targets,
		weights:   weights,
		maxWeight: maxWeight,
		gcd:       gcdSlice(weights),
		current:   -1,
	}
}

func (wrr *WeightedRoundRobin) Next() *Target {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	if len(wrr.targets) == 0 {
		return nil
	}

	for {
		wrr.current = (wrr.current + 1) % len(wrr.targets)
		if wrr.current == 0 {
			wrr.currentWeight -= wrr.gcd
			if wrr.currentWeight <= 0 {
				wrr.currentWeight = wrr.maxWeight
			}
		}

		if wrr.weights[wrr.current] >= wrr.currentWeight {
			target := wrr.targets[wrr.current]
			if target.Healthy.Load() {
				return target
			}
		}

		if wrr.current == 0 && wrr.currentWeight == wrr.maxWeight {
			return wrr.targets[0]
		}
	}
}

func (wrr *WeightedRoundRobin) Targets() []*Target {
	wrr.mu.RLock()
	defer wrr.mu.RUnlock()
	return wrr.targets
}

func (wrr *WeightedRoundRobin) MarkHealthy(target *Target, healthy bool) {
	target.Healthy.Store(healthy)
}

func gcdSlice(nums []int) int {
	if len(nums) == 0 {
		return 1
	}
	result := nums[0]
	for _, num := range nums[1:] {
		result = gcd(result, num)
	}
	return result
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func New(strategy string, targets []*Target) LoadBalancer {
	switch strategy {
	case "least_conn":
		return NewLeastConn(targets)
	case "random":
		return NewRandom(targets)
	case "weighted_round_robin":
		return NewWeightedRoundRobin(targets)
	default:
		return NewRoundRobin(targets)
	}
}
