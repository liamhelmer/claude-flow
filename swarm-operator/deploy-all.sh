#!/bin/bash
set -e

echo "🚀 Claude Flow Swarm Operator Complete Deployment"
echo "==============================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to check if a namespace exists
namespace_exists() {
    kubectl get namespace "$1" &> /dev/null
}

# Function to wait for deployment
wait_for_deployment() {
    local namespace=$1
    local deployment=$2
    local timeout=${3:-300}
    
    echo "⏳ Waiting for $deployment in $namespace to be ready..."
    if kubectl wait --for=condition=available --timeout=${timeout}s deployment/$deployment -n $namespace; then
        echo -e "${GREEN}✅ $deployment is ready${NC}"
    else
        echo -e "${RED}❌ Timeout waiting for $deployment${NC}"
        kubectl get pods -n $namespace -l app=$deployment
        return 1
    fi
}

# Function to wait for CRDs
wait_for_crds() {
    echo "⏳ Waiting for CRDs to be established..."
    for crd in swarmclusters.swarm.claudeflow.io swarmagents.swarm.claudeflow.io swarmtasks.swarm.claudeflow.io swarmmemories.swarm.claudeflow.io swarmmemorystores.swarm.claudeflow.io; do
        if kubectl wait --for=condition=Established --timeout=60s crd/$crd; then
            echo -e "${GREEN}✅ CRD $crd is ready${NC}"
        else
            echo -e "${RED}❌ CRD $crd failed to establish${NC}"
            return 1
        fi
    done
}

echo "📋 Pre-deployment checks..."

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}❌ kubectl not found. Please install kubectl first.${NC}"
    exit 1
fi

# Check if connected to a cluster
if ! kubectl cluster-info &> /dev/null; then
    echo -e "${RED}❌ Not connected to a Kubernetes cluster${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Connected to Kubernetes cluster${NC}"
kubectl cluster-info | head -n 1

echo ""
echo "🏗️  Step 1: Creating namespaces..."

# Create namespaces if they don't exist
for ns in swarm-system claude-flow-swarm claude-flow-hivemind; do
    if namespace_exists "$ns"; then
        echo "  ✓ Namespace $ns already exists"
    else
        kubectl create namespace "$ns"
        echo -e "${GREEN}  ✅ Created namespace $ns${NC}"
    fi
done

echo ""
echo "📦 Step 2: Installing CRDs..."

# Apply CRDs
if kubectl apply -f deploy/all-crds.yaml; then
    echo -e "${GREEN}✅ CRDs applied successfully${NC}"
else
    echo -e "${RED}❌ Failed to apply CRDs${NC}"
    exit 1
fi

# Wait for CRDs to be ready
wait_for_crds

echo ""
echo "🔐 Step 3: Setting up RBAC..."

# Apply RBAC resources
echo "Applying RBAC resources..."
# We'll use the enhanced RBAC from the swarm-operator directory
kubectl apply -f swarm-operator/deploy/enhanced-rbac.yaml || true
kubectl apply -f swarm-operator/deploy/rbac.yaml || true

# Apply cross-namespace RBAC from setup script
kubectl apply -f - <<EOF
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: github-secret-reader
  namespace: swarm-system
rules:
- apiGroups: [""]
  resources: ["secrets"]
  resourceNames: ["github-credentials", "github-app-key"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: swarm-agents-github-access
  namespace: swarm-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: github-secret-reader
subjects:
- kind: ServiceAccount
  name: swarm-agent
  namespace: claude-flow-swarm
- kind: ServiceAccount
  name: swarm-agent
  namespace: claude-flow-hivemind
- kind: ServiceAccount
  name: mcp-server
  namespace: claude-flow-swarm
EOF

echo -e "${GREEN}✅ RBAC configured${NC}"

echo ""
echo "🎛️  Step 4: Deploying Swarm Operator..."

# Deploy the operator
kubectl apply -f swarm-operator/deploy/enhanced-operator-deployment.yaml

# Wait for operator to be ready
wait_for_deployment swarm-system swarm-operator

echo ""
echo "🐝 Step 5: Deploying MCP Server..."

# Deploy MCP server
kubectl apply -f deploy/mcp-server-deployment.yaml

# Wait for MCP server to be ready
wait_for_deployment claude-flow-swarm mcp-server

echo ""
echo "📝 Step 6: Creating ConfigMaps..."

# Apply configmaps  
kubectl apply -f swarm-operator/deploy/github-app-script-configmap.yaml || true
kubectl apply -f swarm-operator/deploy/github-script-configmap.yaml || true

echo -e "${GREEN}✅ ConfigMaps created${NC}"

echo ""
echo "🔍 Step 7: Verifying deployment..."

# Check all deployments
echo ""
echo "Deployments:"
kubectl get deployments -A | grep -E "swarm|claude-flow|NAMESPACE" || true

echo ""
echo "Pods:"
kubectl get pods -A | grep -E "swarm|claude-flow|NAMESPACE" || true

echo ""
echo "Services:"
kubectl get services -A | grep -E "swarm|claude-flow|NAMESPACE" || true

echo ""
echo "CRDs:"
kubectl get crds | grep claudeflow || true

echo ""
echo -e "${GREEN}🎉 Deployment complete!${NC}"
echo ""
echo "📋 Next steps:"
echo ""
echo "1. Set up GitHub credentials (if not already done):"
echo "   ./swarm-operator/setup-github-token.sh"
echo ""
echo "2. Create a SwarmCluster:"
echo "   kubectl apply -f examples/sqlite-memory-cluster.yaml"
echo ""
echo "3. Create a SwarmTask:"
echo "   kubectl apply -f examples/github-automation-task.yaml"
echo ""
echo "4. Monitor the swarm:"
echo "   kubectl get swarmclusters -A"
echo "   kubectl get swarmtasks -A"
echo "   kubectl get swarmagents -A"
echo ""
echo "5. View logs:"
echo "   kubectl logs -n swarm-system deployment/swarm-operator -f"
echo "   kubectl logs -n claude-flow-swarm deployment/mcp-server -f"
echo ""
echo "📚 Documentation:"
echo "   - Namespace Guide: docs/NAMESPACE_MIGRATION_GUIDE.md"
echo "   - Secrets Guide: docs/SECRETS_AND_TOKENS_GUIDE.md"
echo "   - GitHub App Guide: docs/GITHUB_APP_GUIDE.md"