#!/bin/bash

echo "🔑 GitHub Token Setup for Swarm Operator"
echo "========================================"
echo ""
echo "To create a GitHub repository from Kubernetes, you need a Personal Access Token."
echo ""
echo "Steps to create a token:"
echo "1. Open: https://github.com/settings/tokens"
echo "2. Click 'Generate new token (classic)'"
echo "3. Give it a name like 'swarm-operator'"
echo "4. Select these scopes:"
echo "   ✓ repo (Full control of private repositories)"
echo "   ✓ delete_repo (optional, for cleanup)"
echo "5. Click 'Generate token'"
echo "6. Copy the token immediately (you won't see it again!)"
echo ""
read -p "Enter your GitHub Personal Access Token: " -s GITHUB_TOKEN
echo ""

if [ -z "$GITHUB_TOKEN" ]; then
    echo "❌ No token provided"
    exit 1
fi

# Test the token
echo "🔍 Testing token..."
RESPONSE=$(curl -s -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user)
USERNAME=$(echo $RESPONSE | grep -o '"login":"[^"]*' | cut -d'"' -f4)

if [ -z "$USERNAME" ]; then
    echo "❌ Invalid token or API error"
    echo "Response: $RESPONSE"
    exit 1
fi

echo "✅ Token valid for user: $USERNAME"

# Create the secret
echo "🔐 Creating Kubernetes secret..."
kubectl create secret generic github-credentials \
    --from-literal=username="$USERNAME" \
    --from-literal=token="$GITHUB_TOKEN" \
    --from-literal=email="$USERNAME@users.noreply.github.com" \
    --namespace=default \
    --dry-run=client -o yaml | kubectl apply -f -

echo "✅ Secret created successfully!"

# Also export for current session
export GITHUB_TOKEN="$GITHUB_TOKEN"
echo ""
echo "Token has been stored as a Kubernetes secret and exported for this session."
echo "You can now run: ./deploy-github-swarm.sh"