#!/bin/bash

echo "📝 Creating GitHub App credentials secret..."

# GitHub App details
APP_ID="1566739"
CLIENT_ID="Iv23liHKGCAQhZpOdb7a"
INSTALLATION_ID="75143529"
PRIVATE_KEY_FILE="/Users/liam.helmer/Downloads/swarm-sandbox-bot.2025-07-09.private-key.pem"

# Check if private key file exists
if [ ! -f "$PRIVATE_KEY_FILE" ]; then
    echo "❌ Error: Private key file not found at $PRIVATE_KEY_FILE"
    exit 1
fi

# Create the secret from the private key file and literals
kubectl create secret generic github-app-credentials \
    --from-file=private-key="$PRIVATE_KEY_FILE" \
    --from-literal=app-id="$APP_ID" \
    --from-literal=client-id="$CLIENT_ID" \
    --from-literal=installation-id="$INSTALLATION_ID" \
    --namespace=default \
    --dry-run=client -o yaml | kubectl apply -f -

echo "✅ GitHub App credentials secret created"

# Also keep the personal access token secret for compatibility
echo "📝 Checking existing github-credentials secret..."
kubectl get secret github-credentials -n default >/dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✅ Personal access token secret already exists"
else
    echo "⚠️  No personal access token secret found"
fi

echo ""
echo "Secrets available:"
kubectl get secrets -n default | grep github