#!/bin/bash

# Script to create Kubernetes secrets from environment variables

echo "Creating GitHub and Claude credentials as Kubernetes secrets..."

# Check if GitHub token exists
if [ -z "$GITHUB_TOKEN" ]; then
    echo "Error: GITHUB_TOKEN environment variable not set"
    echo "Please run: export GITHUB_TOKEN=your_github_token"
    exit 1
fi

# Create GitHub secret
kubectl create secret generic github-credentials \
    --from-literal=username=liamhelmer \
    --from-literal=token="$GITHUB_TOKEN" \
    --from-literal=email="${GITHUB_EMAIL:-liamhelmer@users.noreply.github.com}" \
    --namespace=default \
    --dry-run=client -o yaml | kubectl apply -f -

echo "✅ GitHub credentials secret created"

# Create Claude secret (using placeholder for now)
kubectl create secret generic claude-credentials \
    --from-literal=api_key="${CLAUDE_API_KEY:-placeholder}" \
    --from-literal=model="claude-3-opus-20240229" \
    --namespace=default \
    --dry-run=client -o yaml | kubectl apply -f -

echo "✅ Claude credentials secret created"

# Verify secrets
echo ""
echo "Secrets created:"
kubectl get secrets | grep -E "github-credentials|claude-credentials"