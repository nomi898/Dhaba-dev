#!/bin/bash
set -euo pipefail

REGION="eu-north-1"

REPOSITORIES=(
  "dhaba-web"
  "dhaba-menu-api"
  "dhaba-orders-api"
  "dhaba-kitchen-worker"
  "dhaba-stats-api"
  "postgres"
  "redis"
  "ops-monitor"
  "dhaba-stats-nginx"
)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
POLICY_FILE="$SCRIPT_DIR/ecr-lifecycle-policy.json"

for REPO in "${REPOSITORIES[@]}"; do
    echo "Processing $REPO..."

    if aws ecr describe-repositories \
        --repository-names "$REPO" \
        --region "$REGION" >/dev/null 2>&1; then
        echo "$REPO already exists"
    else
        aws ecr create-repository \
            --repository-name "$REPO" \
            --image-scanning-configuration scanOnPush=true \
            --region "$REGION"
    fi

    aws ecr put-image-scanning-configuration \
        --repository-name "$REPO" \
        --image-scanning-configuration scanOnPush=true \
        --region "$REGION" >/dev/null

    aws ecr put-lifecycle-policy \
        --repository-name "$REPO" \
        --lifecycle-policy-text "file://$POLICY_FILE" \
        --region "$REGION" >/dev/null

    echo "$REPO configured successfully"
done

echo "All ECR repositories configured."
