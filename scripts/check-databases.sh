#!/usr/bin/env bash
# Checks the PostgreSQL and Redis StatefulSets and runs a live health command
# in each. Needs pods/exec (admin kubeconfig, not dhaba-deployer).
# Usage: ./scripts/check-databases.sh [namespace ...]   (default: staging production)
# Exit code: 0 = healthy, 1 = a check failed.
set -uo pipefail

if (( $# > 0 )); then
  NAMESPACES=("$@")
else
  NAMESPACES=(staging production)
fi

FAILED=0

echo "================================"
echo "      DHABA DATABASE HEALTH"
echo "================================"

for namespace in "${NAMESPACES[@]}"; do
  echo
  echo "Checking namespace: $namespace"

  # PostgreSQL
  if kubectl rollout status statefulset/postgres \
    -n "$namespace" --timeout=30s >/dev/null 2>&1; then
    echo "PASS: statefulset/postgres is rolled out"

    # Uses the container's own POSTGRES_USER/POSTGRES_DB, so the app database is checked.
    # shellcheck disable=SC2016
    if kubectl exec -n "$namespace" postgres-0 -c postgres -- \
      sh -c 'pg_isready -h 127.0.0.1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' >/dev/null 2>&1; then
      echo "PASS: PostgreSQL is accepting connections"
    else
      echo "FAIL: PostgreSQL is not accepting connections"
      FAILED=1
    fi
  else
    echo "FAIL: statefulset/postgres is not fully rolled out"
    FAILED=1
  fi

  # Redis
  if kubectl rollout status statefulset/redis \
    -n "$namespace" --timeout=30s >/dev/null 2>&1; then
    echo "PASS: statefulset/redis is rolled out"

    redis_result=$(kubectl exec -n "$namespace" redis-0 -c redis -- \
      redis-cli ping 2>/dev/null | tr -d '\r' || true)

    if [[ "$redis_result" == "PONG" ]]; then
      echo "PASS: Redis answered PING"
    else
      echo "FAIL: Redis did not answer PING (response: ${redis_result:-none})"
      FAILED=1
    fi
  else
    echo "FAIL: statefulset/redis is not fully rolled out"
    FAILED=1
  fi
done

echo
if (( FAILED == 0 )); then
  echo "DATABASE HEALTH PASSED"
else
  echo "DATABASE HEALTH FAILED"
fi

exit "$FAILED"
