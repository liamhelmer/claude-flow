#!/bin/bash
# Enhanced Swarm Executor Entrypoint

echo "🐝 Swarm Executor v2.0.0 starting..."

# Setup cloud credentials if available
if [ -f "/credentials/gcp/key.json" ]; then
    echo "🔐 Setting up Google Cloud credentials..."
    export GOOGLE_APPLICATION_CREDENTIALS="/credentials/gcp/key.json"
    gcloud auth activate-service-account --key-file="$GOOGLE_APPLICATION_CREDENTIALS"
fi

if [ -f "/credentials/aws/credentials" ]; then
    echo "🔐 Setting up AWS credentials..."
    mkdir -p ~/.aws
    cp /credentials/aws/* ~/.aws/
fi

if [ -f "/credentials/azure/config" ]; then
    echo "🔐 Setting up Azure credentials..."
    mkdir -p ~/.azure
    cp /credentials/azure/* ~/.azure/
fi

# Setup kubectl if kubeconfig is mounted
if [ -f "/credentials/kubeconfig" ]; then
    echo "🔐 Setting up kubectl..."
    export KUBECONFIG="/credentials/kubeconfig"
elif [ -f "/var/run/secrets/kubernetes.io/serviceaccount/token" ]; then
    echo "🔐 Using in-cluster Kubernetes configuration..."
fi

# Check for resume mode
if [ "$RESUME_TASK" = "true" ] && [ -f "/swarm-state/checkpoint.json" ]; then
    echo "📂 Resuming from checkpoint..."
    /scripts/resume.sh
fi

# Execute the command
exec "$@"