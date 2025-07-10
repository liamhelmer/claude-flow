#!/bin/bash

set -e

echo "🚀 Deploying GitHub Automation Swarm..."

# Check if GITHUB_TOKEN is set
if [ -z "$GITHUB_TOKEN" ]; then
    echo "❌ Error: GITHUB_TOKEN environment variable not set"
    echo "Please run: export GITHUB_TOKEN=your_github_personal_access_token"
    echo ""
    echo "To create a token:"
    echo "1. Go to https://github.com/settings/tokens"
    echo "2. Generate new token (classic)"
    echo "3. Select scopes: repo (all), delete_repo"
    echo "4. Copy the token and run: export GITHUB_TOKEN=<your_token>"
    exit 1
fi

# Create secrets
echo "🔐 Creating GitHub credentials secret..."
./deploy/secrets/create-secrets.sh

# Update operator
echo "🔄 Updating operator with Job creation capability..."
./deploy-update.sh

# Clean up any existing tasks/jobs
echo "🧹 Cleaning up existing resources..."
kubectl delete swarmtask --all 2>/dev/null || true
kubectl delete job -l swarm.claudeflow.io/type=github-automation 2>/dev/null || true

# Deploy the GitHub swarm and task
echo "🐝 Creating GitHub automation swarm..."
kubectl apply -f examples/github-hello-world-task.yaml

# Wait a moment for the operator to process
sleep 5

# Show status
echo ""
echo "📊 Swarm Status:"
kubectl get swarmclusters github-automation-swarm
echo ""
echo "📋 Task Status:"
kubectl get swarmtasks create-hello-world-repo
echo ""
echo "💼 Jobs:"
kubectl get jobs -l swarm.claudeflow.io/type=github-automation
echo ""

# Watch the job logs
echo "📜 Watching job logs (press Ctrl+C to stop)..."
echo ""

# Wait for job to be created
for i in {1..30}; do
    JOB_NAME=$(kubectl get jobs -l swarm.claudeflow.io/task=create-hello-world-repo -o name 2>/dev/null | head -n1)
    if [ ! -z "$JOB_NAME" ]; then
        break
    fi
    echo "Waiting for job to be created..."
    sleep 2
done

if [ ! -z "$JOB_NAME" ]; then
    kubectl logs -f $JOB_NAME
else
    echo "Job not created yet. Check operator logs:"
    echo "kubectl -n swarm-system logs deployment/swarm-operator"
fi