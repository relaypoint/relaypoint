#!/bin/bash
# start_test_env.sh - Start everything needed for testing
# Usage: ./start_test_env.sh

echo "-----------------------------------------------"
echo "===       RelayPoint Test Environment       ==="
echo "-----------------------------------------------"

cd "$(dirname "$0")/../.."

# Build the gateway
echo ""
echo "Building relaypoint..."
go build -o relaypoint ./cmd/relaypoint/main.go
echo "✓ Relaypoint built"

echo ""
echo "Starting mock backends in background..."
go run test/integration/mock_backends.go &
BACKENDS_PID=$!
sleep 2

echo ""
echo "Starting relaypoint..."
./relaypoint -config test/integration/test_config.yml &
RELAYPOINT_PID=$!
sleep 2

echo ""
echo "-----------------------------------------------"
echo "===          Environment Ready!             ==="
echo "-----------------------------------------------"
echo ""
echo "Relaypoint:  http://localhost:8080"
echo "Metrics:  http://localhost:9090/metrics"
echo "Stats:    http://localhost:8080/stats"
echo ""
echo "Try these commands:"
echo ""
echo "  # Get users (load balanced)"
echo "  curl http://localhost:8080/api/v1/users"
echo ""
echo "  # Get specific user"
echo "  curl http://localhost:8080/api/v1/users/123"
echo ""
echo "  # Get orders"
echo "  curl http://localhost:8080/api/v1/orders"
echo ""
echo "  # Get products"  
echo "  curl http://localhost:8080/api/v1/products"
echo ""
echo "  # Test rate limiting (run multiple times quickly)"
echo "  for i in {1..20}; do curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/api/v1/orders; done"
echo ""
echo "  # Test with API key"
echo "  curl -H 'X-API-Key: test-key-limited' http://localhost:8080/api/v1/users"
echo ""
echo "  # View metrics"
echo "  curl http://localhost:9090/metrics"
echo ""
echo "  # View stats"
echo "  curl http://localhost:8080/stats"
echo ""
echo "Press Ctrl+C to stop all services..."

# Cleanup on exit
cleanup() {
    echo ""
    echo "Stopping services..."
    kill $BACKENDS_PID 2>/dev/null
    kill $GATEWAY_PID 2>/dev/null
    echo "Done."
    exit 0
}

trap cleanup SIGINT SIGTERM

# Wait
wait
