#!/bin/bash
set -e

ACCOUNT=270254976774
REGION=eu-north-1
TAG=v1

docker compose build

for SERVICE in web menu-api orders-api kitchen-worker stats-api stats-nginx
do
  LOCAL="dhaba-dev-${SERVICE}:latest"
  REMOTE="${ACCOUNT}.dkr.ecr.${REGION}.amazonaws.com/dhaba-${SERVICE}:${TAG}"

  echo "Pushing ${SERVICE}..."

  docker tag "$LOCAL" "$REMOTE"
  docker push "$REMOTE"
done

echo "All images pushed successfully!"
