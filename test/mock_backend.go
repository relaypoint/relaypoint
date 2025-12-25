package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

var requestCount uint64

func main() {
	port := flag.Int("port", 3001, "Port to listen on")
	name := flag.String("name", "backend", "Service name")
	delay := flag.Duration("delay", 0, "Response delay")
	flag.Parse()

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddUint64(&requestCount, 1)

		if *delay > 0 {
			time.Sleep(*delay)
		}

		w.Header().Set("Content-Type", "application/json")

		response := map[string]interface{}{
			"backend":     *name,
			"port":        *port,
			"path":        r.URL.Path,
			"method":      r.Method,
			"request_num": count,
			"timestamp":   time.Now().Format(time.RFC3339),
			"headers":     flattenHeaders(r.Header),
		}

		if r.URL.Query().Get("id") != "" {
			response["id"] = r.URL.Query().Get("id")
		}

		// Simulate different responses based on path
		switch {
		case contains(r.URL.Path, "/users"):
			response["data"] = map[string]interface{}{
				"users": []map[string]interface{}{
					{"id": 1, "name": "Alice"},
					{"id": 2, "name": "Bob"},
				},
			}
		case contains(r.URL.Path, "/orders"):
			response["data"] = map[string]interface{}{
				"orders": []map[string]interface{}{
					{"id": "ord-001", "total": 99.99},
					{"id": "ord-002", "total": 149.50},
				},
			}
		case contains(r.URL.Path, "/products"):
			response["data"] = map[string]interface{}{
				"products": []map[string]interface{}{
					{"id": "prod-1", "name": "Widget", "price": 19.99},
					{"id": "prod-2", "name": "Gadget", "price": 29.99},
				},
			}
		}

		json.NewEncoder(w).Encode(response)
	})

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("[%s] Starting on %s", *name, addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func flattenHeaders(h http.Header) map[string]string {
	result := make(map[string]string)
	for k, v := range h {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}

	return result
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
