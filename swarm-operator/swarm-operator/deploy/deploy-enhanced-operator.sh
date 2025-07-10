#!/bin/bash
set -e

echo "🚀 Deploying Enhanced Swarm Operator v2.0.0"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="swarm-system"
OPERATOR_IMAGE="claudeflow/swarm-operator:2.0.0"
EXECUTOR_IMAGE="claudeflow/swarm-executor:2.0.0"

# Function to check if resource exists
resource_exists() {
    kubectl get $1 $2 -n $3 &> /dev/null
}

# Function to wait for deployment
wait_for_deployment() {
    echo "⏳ Waiting for $1 to be ready..."
    kubectl wait --for=condition=available --timeout=300s deployment/$1 -n $2
}

# Check kubectl connection
echo "🔍 Checking Kubernetes connection..."
if ! kubectl cluster-info &> /dev/null; then
    echo -e "${RED}❌ Cannot connect to Kubernetes cluster${NC}"
    echo "Please ensure kubectl is configured correctly"
    exit 1
fi

CLUSTER_CONTEXT=$(kubectl config current-context)
echo -e "${GREEN}✅ Connected to cluster: ${CLUSTER_CONTEXT}${NC}"

# Confirmation prompt
echo -e "${YELLOW}⚠️  This will deploy the Enhanced Swarm Operator to your cluster${NC}"
read -p "Continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Deployment cancelled"
    exit 0
fi

# Create namespace
echo "📁 Creating namespace..."
kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

# Apply CRDs
echo "📋 Installing Custom Resource Definitions..."
kubectl apply -f crds/enhanced-swarmtask-crd.yaml
kubectl apply -f crds/swarmcluster-crd.yaml
kubectl apply -f crds/agent-crd.yaml

# Wait for CRDs to be established
echo "⏳ Waiting for CRDs to be ready..."
kubectl wait --for=condition=established --timeout=60s crd/swarmtasks.swarm.claudeflow.io
kubectl wait --for=condition=established --timeout=60s crd/swarmclusters.swarm.claudeflow.io
kubectl wait --for=condition=established --timeout=60s crd/agents.swarm.claudeflow.io

# Apply RBAC
echo "🔐 Setting up RBAC..."
kubectl apply -f enhanced-rbac.yaml

# Apply ConfigMaps
echo "📝 Creating ConfigMaps..."
kubectl apply -f enhanced-operator-deployment.yaml

# Check if operator image exists
echo "🐳 Checking operator image..."
if ! docker manifest inspect $OPERATOR_IMAGE &> /dev/null; then
    echo -e "${YELLOW}⚠️  Operator image not found in registry${NC}"
    echo "Building operator image locally..."
    
    if [ -f "../Dockerfile" ]; then
        docker build -t $OPERATOR_IMAGE ..
        echo -e "${GREEN}✅ Operator image built${NC}"
    else
        echo -e "${RED}❌ Dockerfile not found${NC}"
        echo "Please build and push the operator image first:"
        echo "  docker build -t $OPERATOR_IMAGE ."
        echo "  docker push $OPERATOR_IMAGE"
        exit 1
    fi
fi

# Check if executor image exists
echo "🐳 Checking executor image..."
if ! docker manifest inspect $EXECUTOR_IMAGE &> /dev/null; then
    echo -e "${YELLOW}⚠️  Executor image not found in registry${NC}"
    echo "Building executor image locally..."
    
    if [ -f "../build/Dockerfile.swarm-executor" ]; then
        docker build -f ../build/Dockerfile.swarm-executor -t $EXECUTOR_IMAGE ../build
        echo -e "${GREEN}✅ Executor image built${NC}"
    else
        echo -e "${RED}❌ Dockerfile.swarm-executor not found${NC}"
        echo "Please build and push the executor image first:"
        echo "  docker build -f build/Dockerfile.swarm-executor -t $EXECUTOR_IMAGE build/"
        echo "  docker push $EXECUTOR_IMAGE"
        exit 1
    fi
fi

# Deploy operator
echo "🤖 Deploying operator..."
kubectl apply -f enhanced-operator-deployment.yaml

# Wait for operator to be ready
wait_for_deployment "swarm-operator" $NAMESPACE

# Create default storage classes if they don't exist
echo "💾 Checking storage classes..."
if ! kubectl get storageclass fast-ssd &> /dev/null; then
    echo "Creating fast-ssd storage class..."
    cat <<EOF | kubectl apply -f -
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: fast-ssd
provisioner: kubernetes.io/gce-pd
parameters:
  type: pd-ssd
allowVolumeExpansion: true
EOF
fi

# Create example secrets (dummy values)
echo "🔐 Creating example secrets..."
if ! resource_exists secret github-credentials default; then
    kubectl create secret generic github-credentials \
        --from-literal=username=github-user \
        --from-literal=token=ghp_PLACEHOLDER_TOKEN \
        --from-literal=email=user@example.com \
        --namespace=default \
        --dry-run=client -o yaml | kubectl apply -f -
fi

if ! resource_exists secret api-keys default; then
    kubectl create secret generic api-keys \
        --from-literal=openai_key=sk-PLACEHOLDER \
        --from-literal=anthropic_key=sk-ant-PLACEHOLDER \
        --namespace=default \
        --dry-run=client -o yaml | kubectl apply -f -
fi

# Deploy example swarm cluster
echo "🐝 Creating example SwarmCluster..."
cat <<EOF | kubectl apply -f -
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: default-swarm
  namespace: default
spec:
  topology: mesh
  size: 5
  agents:
  - type: coordinator
    replicas: 1
  - type: researcher
    replicas: 2
  - type: coder
    replicas: 2
EOF

# Show status
echo ""
echo "==================================="
echo -e "${GREEN}✅ Enhanced Swarm Operator Deployed!${NC}"
echo "==================================="
echo ""
echo "📊 Deployment Status:"
kubectl get all -n $NAMESPACE

echo ""
echo "🎯 Next Steps:"
echo "1. Build and push the executor image if not already done:"
echo "   docker build -f build/Dockerfile.swarm-executor -t $EXECUTOR_IMAGE build/"
echo "   docker push $EXECUTOR_IMAGE"
echo ""
echo "2. Configure cloud credentials:"
echo "   kubectl create secret generic gcp-credentials --from-file=key.json"
echo "   kubectl create secret generic aws-credentials --from-file=credentials --from-file=config"
echo ""
echo "3. Deploy your first enhanced task:"
echo "   kubectl apply -f examples/enhanced-task-examples.yaml"
echo ""
echo "4. Monitor operator logs:"
echo "   kubectl logs -n $NAMESPACE deployment/swarm-operator -f"
echo ""
echo "5. Check task status:"
echo "   kubectl get swarmtasks"
echo ""
echo "📚 Full documentation: ENHANCED_OPERATOR_GUIDE.md"