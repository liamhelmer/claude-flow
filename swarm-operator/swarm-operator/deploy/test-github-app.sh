#!/bin/bash

echo "🔐 Testing GitHub App authentication..."

# GitHub App details
APP_ID="1566739"
INSTALLATION_ID="75143529"
PRIVATE_KEY_FILE="/Users/liam.helmer/Downloads/swarm-sandbox-bot.2025-07-09.private-key.pem"

# Generate JWT
JWT=$(python3 - <<EOF
import jwt
import time
import os

# Read the private key
with open('$PRIVATE_KEY_FILE', 'r') as f:
    private_key = f.read()

# Generate JWT
now = int(time.time())
payload = {
    'iat': now - 60,
    'exp': now + 600,
    'iss': '$APP_ID'
}

token = jwt.encode(payload, private_key, algorithm='RS256')
print(token)
EOF
)

echo "🔑 Generated JWT token"

# Get installation access token
TOKEN_RESPONSE=$(curl -s -X POST \
  -H "Authorization: Bearer $JWT" \
  -H "Accept: application/vnd.github.v3+json" \
  "https://api.github.com/app/installations/$INSTALLATION_ID/access_tokens")

ACCESS_TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.token')

if [ -z "$ACCESS_TOKEN" ] || [ "$ACCESS_TOKEN" = "null" ]; then
    echo "❌ Failed to get access token"
    echo "Response: $TOKEN_RESPONSE"
    exit 1
fi

echo "✅ Got access token"

# Check permissions
echo ""
echo "📋 Checking installation permissions..."
curl -s -H "Authorization: token $ACCESS_TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  "https://api.github.com/installation/repositories" | jq -r '.repositories[] | "\(.full_name) - \(.permissions)"'

echo ""
echo "🔧 Creating repository badal-io/rcm-test1..."

# Create repository
CREATE_RESPONSE=$(curl -s -X POST \
  -H "Authorization: token $ACCESS_TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  "https://api.github.com/orgs/badal-io/repos" \
  -d '{
    "name": "rcm-test1",
    "description": "🐝 Test repository for Claude Flow Swarm Operator using GitHub App authentication",
    "homepage": "https://github.com/claude-flow/swarm-operator",
    "private": false,
    "has_issues": true,
    "has_projects": false,
    "has_wiki": false,
    "auto_init": false
  }')

if echo "$CREATE_RESPONSE" | jq -e '.id' > /dev/null 2>&1; then
    echo "✅ Repository created successfully!"
    echo "URL: $(echo "$CREATE_RESPONSE" | jq -r '.html_url')"
elif echo "$CREATE_RESPONSE" | jq -e '.errors[0].code == "already_exists"' > /dev/null 2>&1; then
    echo "ℹ️  Repository already exists"
else
    echo "❌ Failed to create repository"
    echo "Response: $CREATE_RESPONSE"
fi