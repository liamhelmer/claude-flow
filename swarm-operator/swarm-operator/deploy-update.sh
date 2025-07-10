#!/bin/bash

set -e

echo "🔄 Updating Swarm Operator..."

# Rebuild the operator image
echo "📦 Building new operator image..."
docker build -t swarm-operator:v0.2.0 .

# Update RBAC permissions
echo "🔐 Updating RBAC permissions..."
kubectl apply -f deploy/rbac-update.yaml

# Delete old operator pod to force recreation with new image
echo "♻️  Restarting operator..."
kubectl -n swarm-system delete pod -l app.kubernetes.io/name=swarm-operator

# Wait for new pod to be ready
echo "⏳ Waiting for operator to restart..."
kubectl -n swarm-system wait --for=condition=ready pod -l app.kubernetes.io/name=swarm-operator --timeout=60s

echo "✅ Operator updated successfully!"