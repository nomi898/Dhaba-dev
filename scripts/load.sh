#!/usr/bin/env bash

set -euo pipefail

URL="${1:-}"
REQUESTS="${2:-100}"
CONCURRENCY="${3:-10}"

if [[ -z "$URL" ]]; then
    echo "Usage: ./scripts/load.sh <URL> [requests] [concurrency]"
    exit 1
fi

if ! [[ "$REQUESTS" =~ ^[0-9]+$ ]] || (( REQUESTS < 1 )); then
    echo "Error: requests must be a positive number."
    exit 1
fi

if ! [[ "$CONCURRENCY" =~ ^[0-9]+$ ]] || (( CONCURRENCY < 1 )); then
    echo "Error: concurrency must be a positive number."
    exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
    echo "Error: curl is required."
    exit 1
fi

RESULTS=$(mktemp)
trap 'rm -f "$RESULTS"' EXIT

echo "================================"
echo "       DHABA LOAD TEST"
echo "================================"
echo "Target:      $URL"
echo "Requests:    $REQUESTS"
echo "Concurrency: $CONCURRENCY"
echo "================================"

START_TIME=$(date +%s)

seq "$REQUESTS" | xargs -P "$CONCURRENCY" -I {} \
    sh -c '
        curl -sS \
            --max-time 15 \
            -o /dev/null \
            -w "%{http_code} %{time_total}\n" \
            "$1" >> "$2" || true
    ' _ "$URL" "$RESULTS"

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

TOTAL=$(wc -l < "$RESULTS" | tr -d ' ')
SUCCESS=$(awk '$1 >= 200 && $1 < 300 { count++ } END { print count+0 }' "$RESULTS")
FAILED=$((TOTAL - SUCCESS))

AVG_TIME=$(awk '
    { total += $2 }
    END {
        if (NR > 0) printf "%.3f", total / NR
        else print "0"
    }
' "$RESULTS")

echo ""
echo "========== RESULTS =========="
echo "Total completed: $TOTAL / $REQUESTS"
echo "Successful:      $SUCCESS"
echo "Failed:          $FAILED"
echo "Average time:    ${AVG_TIME}s"
echo "Duration:        ${DURATION}s"
echo "=============================="

if (( TOTAL != REQUESTS || FAILED > 0 )); then
    echo "LOAD TEST FAILED"
    exit 1
fi

echo "LOAD TEST PASSED"
