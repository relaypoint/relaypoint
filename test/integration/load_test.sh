#!/bin/bash
# load_test.sh - Simple load testing for the gateway
# Usage: ./load_test.sh [requests] [concurrency]

REQUESTS=${1:-1000}
CONCURRENCY=${2:-10}
URL="http://localhost:8080/api/v1/users"

echo "---------------------------------------"
echo "=== Load Testing RelayPoint Gateway ==="
echo "---------------------------------------"
echo "Target URL:       $URL"
echo "Total Requests:   $REQUESTS"
echo "Concurrency:      $CONCURRENCY"
echo "---------------------------------------"

if ! curl -s -o /dev/null "$URL" 2>/dev/null; then
  echo "ERROR: RelayPoint gateway is not responding at $URL. Please ensure it is running before starting the load test."
  exit 1
fi

if command -v hey &> /dev/null; then
  echo "Starting load test with 'hey'..."
  hey -n $REQUESTS -c $CONCURRENCY "$URL"
elif command -v ab &> /dev/null; then
  echo "Using Apache Bench (ab) for load testing..."
  ab -n $REQUESTS -c $CONCURRENCY "$URL"
else
    echo "Using bash/curl for load testing (install 'hey' or 'ab' for better results)..."
    echo ""

    start_time=$(date +%s.%N)
    success=0
    failed=0
    rate_limited=0

    for batch in $(seq 1 $((REQUESTS / CONCURRENCY))); do
        pids=""
        for i in $(seq 1 $CONCURRENCY); do
            (
                code=$(curl -s -o /dev/null -w "%{http_code}" "$URL" 2>/dev/null)
                echo $code
            ) &
            pids="$pids $!"
        done
        for pid in $pids; do
            code=$(wait $pid)
            case $code in
                200) ((success++)) ;;
                429) ((rate_limited++)) ;;
                *)   ((failed++)) ;;
            esac
        done

        completed=$((batch * CONCURRENCY))
        printf "\rProgress: %d/%d requests" "$completed" "$REQUESTS"
    done

    end_time=$(date +%s.%N)
    duration=$(echo "$end_time - $start_time" | bc)
    rps=$(echo "$REQUESTS / $duration" | bc -l)

    echo ""
    echo ""
    echo "Load Test Complete!"
    echo ""
    echo "---------------------------------------"
    echo "===             Results             ==="
    echo "---------------------------------------"
    printf "Total Time:       %.2f seconds\n" $duration
    printf "Requests/sec:     %.2f\n" $rps
    echo "Successful:       $success"
    echo "Rate Limited:     $rate_limited"
    echo "Failed:           $failed"
    echo "---------------------------------------"
fi

echo ""
echo "Check detailed metrics at http://localhost:9090/metrics"
echo "Check stats at http://localhost:8080/stats"