#!/usr/bin/env bash
# Checks application Deployments are fully rolled out and their Services
# have at least one ready endpoint.
# Usage: ./scripts/check-services.sh [namespace ...]   (default: staging production)
# Exit code: 0 = healthy, 1 = a check failed.
set -uo pipefail

if (( $# > 0 )); then
  NAMESPACES=("$@")
else
  NAMESPACES=(staging production)
fi

DEPLOYMENTS=(web menu-api orders-api kitchen-worker stats-api stats-nginx)

# kitchen-worker is a queue consumer with no Service.
SERVICES=(web menu-api orders-api stats-api stats-nginx)

FAILED=0

echo "================================"
echo "      DHABA SERVICE HEALTH"
echo "================================"

for namespace in "${NAMESPACES[@]}"; do
  echo
  echo "Checking namespace: $namespace"

  for deployment in "${DEPLOYMENTS[@]}"; do
    if kubectl rollout status "deployment/$deployment" \
      -n "$namespace" --timeout=30s >/dev/null 2>&1; then
      echo "PASS: deployment/$deployment is rolled out"
    else
      echo "FAIL: deployment/$deployment is not fully rolled out"
      FAILED=1
    fi
  done

  for service in "${SERVICES[@]}"; do
    # EndpointSlices replace the deprecated v1 Endpoints API; count only ready addresses.
    ready=$(kubectl get endpointslices -n "$namespace" \
      -l "kubernetes.io/service-name=$service" \
      -o jsonpath='{range .items[*].endpoints[?(@.conditions.ready==true)]}{.addresses[0]}{"\n"}{end}' \
      2>/dev/null | grep -c . || true)

    if (( ready > 0 )); then
      echo "PASS: service/$service has $ready ready endpoint(s)"
    else
      echo "FAIL: service/$service has no ready endpoints"
      FAILED=1
    fi
  done
done

echo
if (( FAILED == 0 )); then
  echo "SERVICE HEALTH PASSED"
else
  echo "SERVICE HEALTH FAILED"
fi

exit "$FAILED"
