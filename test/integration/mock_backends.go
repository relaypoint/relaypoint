package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

var (
	userRequests    int64
	orderRequests   int64
	productRequests int64
)

func main() {
	// Start 3 mock backend services
	go startUserService(":3001")
	go startUserService(":3002") // Second instance for load balancing
	go startOrderService(":3003")
	go startProductService(":3004")

	log.Println("🚀 Mock backends started:")
	log.Println("   User Service:    http://localhost:3001, :3002")
	log.Println("   Order Service:   http://localhost:3003")
	log.Println("   Product Service: http://localhost:3004")
	log.Println("")
	log.Println("Now start relaypoint:")
	log.Println("   ./relaypoint -config test/integration/test_config.yml")

	// Keep running
	select {}
}

func startUserService(addr string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})

	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&userRequests, 1)
		simulateLatency()

		users := []map[string]interface{}{
			{"id": 1, "name": "Alice", "email": "alice@example.com"},
			{"id": 2, "name": "Bob", "email": "bob@example.com"},
			{"id": 3, "name": "Charlie", "email": "charlie@example.com"},
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Backend-Server", addr)
		w.Header().Set("X-Request-Count", fmt.Sprintf("%d", count))
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"users":   users,
			"server":  addr,
			"request": count,
		})
	})

	mux.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&userRequests, 1)
		simulateLatency()

		// Extract user ID from path
		id := r.URL.Path[len("/users/"):]

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Backend-Server", addr)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      id,
			"name":    "User " + id,
			"email":   fmt.Sprintf("user%s@example.com", id),
			"server":  addr,
			"request": count,
		})
	})

	log.Printf("User service starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("User service %s error: %v", addr, err)
		os.Exit(1)
	}
}

func startOrderService(addr string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})

	mux.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&orderRequests, 1)
		simulateLatency()

		orders := []map[string]interface{}{
			{"id": "ORD-001", "total": 99.99, "status": "shipped"},
			{"id": "ORD-002", "total": 149.50, "status": "pending"},
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Backend-Server", addr)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"orders":  orders,
			"server":  addr,
			"request": count,
		})
	})

	mux.HandleFunc("/orders/", func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&orderRequests, 1)
		simulateLatency()

		id := r.URL.Path[len("/orders/"):]

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Backend-Server", addr)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      id,
			"total":   99.99,
			"status":  "shipped",
			"server":  addr,
			"request": count,
		})
	})

	log.Printf("Order service starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("Order service error: %v", err)
		os.Exit(1)
	}
}

func startProductService(addr string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})

	mux.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&productRequests, 1)
		simulateLatency()

		products := []map[string]interface{}{
			{"id": "PROD-1", "name": "Widget", "price": 29.99},
			{"id": "PROD-2", "name": "Gadget", "price": 49.99},
			{"id": "PROD-3", "name": "Gizmo", "price": 19.99},
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Backend-Server", addr)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"products": products,
			"server":   addr,
			"request":  count,
		})
	})

	log.Printf("Product service starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("Product service error: %v", err)
		os.Exit(1)
	}
}

// simulateLatency adds realistic latency variation
func simulateLatency() {
	delay := time.Duration(10+rand.Intn(50)) * time.Millisecond
	time.Sleep(delay)
}
