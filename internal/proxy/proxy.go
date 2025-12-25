package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/relaypoint/relaypoint/internal/config"
	"github.com/relaypoint/relaypoint/internal/loadbalancer"
	"github.com/relaypoint/relaypoint/internal/metrics"
	"github.com/relaypoint/relaypoint/internal/ratelimit"
	"github.com/relaypoint/relaypoint/internal/router"
)

type Proxy struct {
	router       *router.Router
	upstreams    map[string]loadbalancer.LoadBalancer
	rateLimiter  *ratelimit.RateLimiter
	metrics      *metrics.Metrics
	usageTracker *metrics.UsageTracker
	apiKeys      map[string]*config.APIKey
	config       *config.Config
	httpClient   *http.Client
}

func New(cfg *config.Config) (*Proxy, error) {
	r := router.New(cfg.Routes)

	upstreams := make(map[string]loadbalancer.LoadBalancer)
	for _, u := range cfg.Upstreams {
		targets := make([]*loadbalancer.Target, len(u.Targets))
		for i, t := range u.Targets {
			parsed, err := url.Parse(t.URL)
			if err != nil {
				return nil, fmt.Errorf("invalid upstream URL %s: %w", t.URL, err)
			}
			weight := t.Weight
			if weight <= 0 {
				weight = 1
			}
			targets[i] = &loadbalancer.Target{
				URL:    parsed,
				Weight: weight,
			}
		}
		upstreams[u.Name] = loadbalancer.New(u.LoadBalance, targets)
	}

	rl := ratelimit.NewRateLimiter(ratelimit.Config{
		DefaultRPS:      cfg.RateLimit.DefaultRPS,
		DefaultBurst:    cfg.RateLimit.DefaultBurst,
		CleanupInterval: cfg.RateLimit.CleanupInterval,
	})

	m := metrics.New(metrics.Config{
		LatencyBuckets: cfg.Metrics.LatencyBuckets,
	})

	apiKeys := make(map[string]*config.APIKey)
	for i := range cfg.APIKeys {
		key := &cfg.APIKeys[i]
		if key.Enabled {
			apiKeys[key.Key] = key
			rl.SetLimits("apikey:"+key.Key, key.RequestsPerSecond, key.BurstSize)
		}
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
	}

	return &Proxy{
		router:       r,
		upstreams:    upstreams,
		rateLimiter:  rl,
		metrics:      m,
		usageTracker: metrics.NewUsageTracker(),
		apiKeys:      apiKeys,
		config:       cfg,
		httpClient:   httpClient,
	}, nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	route := p.router.Match(r)
	if route == nil {
		p.metrics.RecordError("unknown", "not_found")
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	routeName := route.Name
	if routeName == "" {
		routeName = route.Pattern
	}

	done := p.metrics.InFlightRequests(routeName)
	defer done()

	clientIP := getClientIP(r)
	apiKey, apiKeyName := p.extractAPIKey(r)

	if p.config.RateLimit.Enabled {
		if !p.checkRateLimits(w, r, route, clientIP, apiKey, routeName) {
			return
		}
	}

	lb, ok := p.upstreams[route.Upstream]
	if !ok {
		p.metrics.RecordError(routeName, "upstream_not_found")
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	target := lb.Next()
	if target == nil {
		p.metrics.RecordError(routeName, "no_healthy_upstream")
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	target.Connections.Add(1)
	defer target.Connections.Add(-1)

	statusCode, err := p.proxyRequest(w, r, route, target)
	duration := time.Since(start)
	isError := statusCode >= 400

	p.metrics.RecordRequest(routeName, r.Method, statusCode, duration)
	p.metrics.RecordUpstreamDuration(route.Upstream, duration)
	p.usageTracker.RecordRequest(routeName, duration, isError)

	if apiKeyName != "" {
		p.metrics.RecordAPIKeyRequest(apiKeyName, statusCode)
		p.usageTracker.RecordRequest("apikey:"+apiKeyName, duration, isError)
	}

	if err != nil {
		p.metrics.RecordError(routeName, "proxy_error")
	}
}

func (p *Proxy) checkRateLimits(w http.ResponseWriter, r *http.Request, route *router.Route, clientIP, apiKey, routeName string) bool {
	if route.RateLimit != nil && route.RateLimit.Enabled {
		key := "route:" + routeName
		if !p.rateLimiter.AllowWithLimits(key, route.RateLimit.RequestsPerSecond, route.RateLimit.BurstSize) {
			p.metrics.RecordRateLimitHit(routeName, "route")
			w.Header().Set("Retry-After", "1")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return false
		}
	}

	if p.config.RateLimit.PerAPIKey && apiKey != "" {
		key := "apikey:" + apiKey
		if !p.rateLimiter.Allow(key) {
			p.metrics.RecordRateLimitHit(routeName, "apikey")
			w.Header().Set("Retry-After", "1")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return false
		}
	}

	if p.config.RateLimit.PerIP && clientIP != "" {
		key := "ip:" + clientIP
		if !p.rateLimiter.Allow(key) {
			p.metrics.RecordRateLimitHit(routeName, "ip")
			w.Header().Set("Retry-After", "1")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return false
		}
	}

	return true
}

func (p *Proxy) proxyRequest(w http.ResponseWriter, r *http.Request, route *router.Route, target *loadbalancer.Target) (int, error) {
	upstreamURL := *target.URL
	path := route.StripPrefix(r.URL.Path)
	upstreamURL.Path = singleJoiningSlash(upstreamURL.Path, path)
	upstreamURL.RawQuery = r.URL.RawQuery

	ctx := r.Context()
	upstreamReq, err := http.NewRequestWithContext(ctx, r.Method, upstreamURL.String(), r.Body)
	if err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return http.StatusBadGateway, err
	}

	copyHeaders(upstreamReq.Header, r.Header)

	for k, v := range route.Headers {
		upstreamReq.Header.Set(k, v)
	}

	clientIP := getClientIP(r)
	if prior := upstreamReq.Header.Get("X-Forwarded-For"); prior != "" {
		upstreamReq.Header.Set("X-Forwarded-For", prior+", "+clientIP)
	} else {
		upstreamReq.Header.Set("X-Forwarded-For", clientIP)
	}

	upstreamReq.Header.Set("X-Forwarded-Host", r.Host)
	upstreamReq.Header.Set("X-Forwarded-Proto", getScheme(r))
	upstreamReq.Header.Set("X-Real-IP", clientIP)

	removeHopHeaders(upstreamReq.Header)

	resp, err := p.httpClient.Do(upstreamReq)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return 499, err // Client Closed Request
		}
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return http.StatusBadGateway, err
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	removeHopHeaders(w.Header())

	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)

	return resp.StatusCode, nil
}

func (p *Proxy) extractAPIKey(r *http.Request) (key string, name string) {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		key = strings.TrimPrefix(auth, "Bearer ")
	} else if strings.HasPrefix(auth, "ApiKey ") {
		key = strings.TrimPrefix(auth, "ApiKey ")
	}

	if key == "" {
		key = r.Header.Get("X-API-Key")
	}

	if key == "" {
		key = r.URL.Query().Get("api_key")
	}

	if key != "" {
		if apiKey, ok := p.apiKeys[key]; ok {
			return key, apiKey.Name
		}
	}

	return key, ""
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func getScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	return "http"
}

func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

func removeHopHeaders(h http.Header) {
	for _, hdr := range hopHeaders {
		h.Del(hdr)
	}
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

func (p *Proxy) Metrics() *metrics.Metrics {
	return p.metrics
}

func (p *Proxy) UsageStats() []metrics.Stats {
	return p.usageTracker.GetStats()
}

func (p *Proxy) Stop() {
	p.rateLimiter.Stop()
}
