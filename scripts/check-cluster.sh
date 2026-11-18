#!/usr/bin/env bash
# Checks that the Kubernetes API is reachable and every node is Ready.
# Needs cluster-scoped read access to nodes (admin kubeconfig, not dhaba-deployer).
# Exit code: 0 = healthy, 1 = a check failed.
set -uo pipefail

FAILED=0

echo "================================"
echo "     DHABA CLUSTER HEALTH"
echo "================================"

if ! kubectl cluster-info >/dev/null 2>&1; then
  echo "FAIL: Kubernetes API is unreachable"
  echo
  echo "CLUSTER HEALTH FAILED"
  exit 1
fi
echo "PASS: Kubernetes API is reachable"

if kubectl wait --for=condition=Ready nodes --all --timeout=30s >/dev/null; then
  echo "PASS: All Kubernetes nodes are Ready"
else
  echo "FAIL: One or more nodes are not Ready"
  FAILED=1
fi

echo
kubectl get nodes -o wide

echo
if (( FAILED == 0 )); then
  echo "CLUSTER HEALTH PASSED"
else
  echo "CLUSTER HEALTH FAILED"
fi

exit "$FAILED"
