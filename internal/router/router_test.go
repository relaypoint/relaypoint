package router

import (
	"net/http/httptest"
	"testing"

	"github.com/relaypoint/relaypoint/internal/config"
)

func TestRouter_PathMatching(t *testing.T) {
	routes := []config.Route{
		{Path: "/api/v1/users", Upstream: "users"},
		{Path: "/api/v1/users/*", Upstream: "users"},
		{Path: "/api/v1/orders/:id", Upstream: "orders"},
		{Path: "/api/**", Upstream: "catchall"},
	}

	r := New(routes)

	tests := []struct {
		path     string
		expected string
	}{
		{"/api/v1/users", "users"},
		{"/api/v1/users/123", "users"},
		{"/api/v1/orders/456", "orders"},
		{"/api/v2/anything/here", "catchall"},
		{"/api/v1/unknown", "catchall"},
	}

	for _, tc := range tests {
		req := httptest.NewRequest("GET", tc.path, nil)
		route := r.Match(req)
		if route == nil {
			t.Errorf("Path %s should match", tc.path)
			continue
		}
		if route.Upstream != tc.expected {
			t.Errorf("Path %s: expected upstream %s, got %s", tc.path, tc.expected, route.Upstream)
		}
	}
}

func TestRouter_HostMatching(t *testing.T) {
	routes := []config.Route{
		{Host: "api.example.com", Path: "/*", Upstream: "api"},
		{Host: "*.example.com", Path: "/*", Upstream: "wildcard"},
		{Path: "/*", Upstream: "default"},
	}

	r := New(routes)

	tests := []struct {
		host     string
		expected string
	}{
		{"api.example.com", "api"},
		{"test.example.com", "wildcard"},
		{"other.com", "default"},
	}

	for _, tc := range tests {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Host = tc.host
		route := r.Match(req)
		if route == nil {
			t.Errorf("Host %s should match", tc.host)
			continue
		}
		if route.Upstream != tc.expected {
			t.Errorf("Host %s: expected upstream %s, got %s", tc.host, tc.expected, route.Upstream)
		}
	}
}

func TestRouter_MethodMatching(t *testing.T) {
	routes := []config.Route{
		{Path: "/api/read", Methods: []string{"GET"}, Upstream: "read"},
		{Path: "/api/write", Methods: []string{"POST", "PUT"}, Upstream: "write"},
		{Path: "/api/any", Upstream: "any"}, // All methods
	}

	r := New(routes)

	tests := []struct {
		method   string
		path     string
		expected string
		match    bool
	}{
		{"GET", "/api/read", "read", true},
		{"POST", "/api/read", "", false},
		{"POST", "/api/write", "write", true},
		{"PUT", "/api/write", "write", true},
		{"DELETE", "/api/write", "", false},
		{"DELETE", "/api/any", "any", true},
	}

	for _, tc := range tests {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		route := r.Match(req)
		if tc.match {
			if route == nil {
				t.Errorf("%s %s should match", tc.method, tc.path)
				continue
			}
			if route.Upstream != tc.expected {
				t.Errorf("%s %s: expected %s, got %s", tc.method, tc.path, tc.expected, route.Upstream)
			}
		} else {
			if route != nil {
				t.Errorf("%s %s should not match", tc.method, tc.path)
			}
		}
	}
}

func TestRouter_PathParams(t *testing.T) {
	routes := []config.Route{
		{Path: "/users/:id/orders/:orderId", Upstream: "orders"},
	}

	r := New(routes)
	req := httptest.NewRequest("GET", "/users/123/orders/456", nil)
	route := r.Match(req)

	if route == nil {
		t.Fatal("Should match")
	}

	if route.PathParams["id"] != "123" {
		t.Errorf("Expected id=123, got %s", route.PathParams["id"])
	}
	if route.PathParams["orderId"] != "456" {
		t.Errorf("Expected orderId=456, got %s", route.PathParams["orderId"])
	}
}

func TestRouter_StripPath(t *testing.T) {
	route := &Route{
		Pattern:   "/api/v1/*",
		StripPath: true,
	}

	result := route.StripPrefix("/api/v1/users/123")
	if result != "/users/123" {
		t.Errorf("Expected /users/123, got %s", result)
	}
}

func TestRouter_NoMatch(t *testing.T) {
	routes := []config.Route{
		{Host: "specific.com", Path: "/specific", Upstream: "specific"},
	}

	r := New(routes)
	req := httptest.NewRequest("GET", "/other", nil)
	req.Host = "other.com"

	route := r.Match(req)
	if route != nil {
		t.Error("Should not match")
	}
}

func TestRouter_Priority(t *testing.T) {
	// More specific routes should match before less specific
	routes := []config.Route{
		{Path: "/**", Upstream: "catchall"},
		{Path: "/api/**", Upstream: "api"},
		{Path: "/api/v1/users", Upstream: "users-exact"},
		{Path: "/api/v1/*", Upstream: "v1"},
	}

	r := New(routes)

	tests := []struct {
		path     string
		expected string
	}{
		{"/api/v1/users", "users-exact"},
		{"/api/v1/orders", "v1"},
		{"/api/v2/test", "api"},
		{"/other/path", "catchall"},
	}

	for _, tc := range tests {
		req := httptest.NewRequest("GET", tc.path, nil)
		route := r.Match(req)
		if route == nil {
			t.Errorf("Path %s should match", tc.path)
			continue
		}
		if route.Upstream != tc.expected {
			t.Errorf("Path %s: expected %s, got %s", tc.path, tc.expected, route.Upstream)
		}
	}
}

func BenchmarkRouter_Match(b *testing.B) {
	routes := []config.Route{
		{Host: "api.example.com", Path: "/v1/users/*", Upstream: "users"},
		{Host: "api.example.com", Path: "/v1/orders/*", Upstream: "orders"},
		{Host: "api.example.com", Path: "/v1/products/*", Upstream: "products"},
		{Path: "/health", Upstream: "health"},
		{Path: "/**", Upstream: "default"},
	}

	r := New(routes)
	req := httptest.NewRequest("GET", "/v1/users/123", nil)
	req.Host = "api.example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(req)
	}
}
