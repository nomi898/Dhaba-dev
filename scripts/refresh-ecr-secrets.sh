
#!/usr/bin/env bash
set -euo pipefail

AWS_REGION="eu-north-1"
AWS_ACCOUNT_ID="270254976774"
ECR_REGISTRY="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"

echo "Refreshing ECR credentials..."

ECR_PASSWORD="$(aws ecr get-login-password --region "$AWS_REGION")"

for NAMESPACE in staging production; do
  kubectl create secret docker-registry ecr-pull-secret \
    --namespace "$NAMESPACE" \
    --docker-server="$ECR_REGISTRY" \
    --docker-username=AWS \
    --docker-password="$ECR_PASSWORD" \
    --dry-run=client -o yaml |
    kubectl apply -f -

  echo "Updated ECR secret in $NAMESPACE"
done

unset ECR_PASSWORD

echo "ECR credentials refreshed successfully."
