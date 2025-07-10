#!/bin/bash

set -e

echo "🚀 Deploying Swarm Operator to Kubernetes..."

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}❌ kubectl not found. Please install kubectl first.${NC}"
    exit 1
fi

# Check if cluster is accessible
if ! kubectl cluster-info &> /dev/null; then
    echo -e "${RED}❌ Cannot connect to Kubernetes cluster. Please check your kubeconfig.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Kubernetes cluster is accessible${NC}"

# Build the operator image
echo -e "${YELLOW}📦 Building operator image...${NC}"
docker build -t swarm-operator:latest .

# For Docker Desktop, the image is already available
# For other clusters, you might need to push to a registry or load into nodes

# Create namespace
echo -e "${YELLOW}📁 Creating namespace...${NC}"
kubectl apply -f deploy/namespace.yaml

# Install CRDs
echo -e "${YELLOW}📋 Installing CRDs...${NC}"
kubectl apply -f deploy/crds/

# Wait for CRDs to be established
echo -e "${YELLOW}⏳ Waiting for CRDs to be ready...${NC}"
kubectl wait --for condition=established --timeout=60s \
    crd/swarmclusters.swarm.claudeflow.io \
    crd/agents.swarm.claudeflow.io \
    crd/swarmtasks.swarm.claudeflow.io

# Install RBAC
echo -e "${YELLOW}🔐 Setting up RBAC...${NC}"
kubectl apply -f deploy/rbac.yaml

# Deploy operator
echo -e "${YELLOW}🎯 Deploying operator...${NC}"
kubectl apply -f deploy/operator.yaml

# Wait for deployment to be ready
echo -e "${YELLOW}⏳ Waiting for operator to be ready...${NC}"
kubectl -n swarm-system wait --for=condition=available --timeout=120s deployment/swarm-operator

# Check deployment status
echo -e "${GREEN}✅ Deployment complete!${NC}"
echo ""
echo "Operator Status:"
kubectl -n swarm-system get deployment swarm-operator
echo ""
echo "Operator Pods:"
kubectl -n swarm-system get pods -l app.kubernetes.io/name=swarm-operator
echo ""
echo "CRDs installed:"
kubectl get crds | grep claudeflow.io
echo ""
echo -e "${GREEN}🎉 Swarm Operator is ready to use!${NC}"
echo ""
echo "To create your first swarm, use:"
echo "  kubectl apply -f examples/basic-swarm.yaml"
echo ""
echo "To check operator logs:"
echo "  kubectl -n swarm-system logs -l app.kubernetes.io/name=swarm-operator -f"