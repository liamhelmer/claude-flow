#!/bin/bash
set -euo pipefail

# Script to set up local Kubernetes cluster for testing swarm-operator

CLUSTER_NAME="${CLUSTER_NAME:-swarm-operator-test}"
KUBERNETES_VERSION="${KUBERNETES_VERSION:-v1.28.0}"
USE_KIND="${USE_KIND:-true}"

echo "🚀 Setting up local Kubernetes cluster for swarm-operator testing..."

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to install Kind if not present
install_kind() {
    echo "📦 Installing Kind..."
    if [[ "$OSTYPE" == "darwin"* ]]; then
        brew install kind
    else
        curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
        chmod +x ./kind
        sudo mv ./kind /usr/local/bin/kind
    fi
}

# Function to install kubectl if not present
install_kubectl() {
    echo "📦 Installing kubectl..."
    if [[ "$OSTYPE" == "darwin"* ]]; then
        brew install kubectl
    else
        curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
        chmod +x kubectl
        sudo mv kubectl /usr/local/bin/
    fi
}

# Function to install Helm if not present
install_helm() {
    echo "📦 Installing Helm..."
    if [[ "$OSTYPE" == "darwin"* ]]; then
        brew install helm
    else
        curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
    fi
}

# Check and install prerequisites
echo "🔍 Checking prerequisites..."

if ! command_exists kubectl; then
    echo "kubectl not found. Installing..."
    install_kubectl
fi

if ! command_exists helm; then
    echo "Helm not found. Installing..."
    install_helm
fi

if [[ "$USE_KIND" == "true" ]]; then
    if ! command_exists kind; then
        echo "Kind not found. Installing..."
        install_kind
    fi
    
    # Check if cluster already exists
    if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
        echo "⚠️  Cluster ${CLUSTER_NAME} already exists. Deleting..."
        kind delete cluster --name "${CLUSTER_NAME}"
    fi
    
    echo "🏗️  Creating Kind cluster..."
    cat <<EOF | kind create cluster --name "${CLUSTER_NAME}" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
- role: worker
- role: worker
- role: worker
EOF
    
    echo "🔄 Setting kubectl context..."
    kubectl cluster-info --context "kind-${CLUSTER_NAME}"
    
else
    # Use Minikube
    if ! command_exists minikube; then
        echo "❌ Minikube not found. Please install minikube first."
        exit 1
    fi
    
    echo "🏗️  Starting Minikube cluster..."
    minikube start --profile="${CLUSTER_NAME}" \
        --kubernetes-version="${KUBERNETES_VERSION}" \
        --nodes=3 \
        --cpus=2 \
        --memory=4096 \
        --driver=docker
    
    echo "🔄 Setting kubectl context..."
    kubectl config use-context "${CLUSTER_NAME}"
fi

# Install metrics server for resource monitoring
echo "📊 Installing metrics-server..."
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# Patch metrics-server for local cluster
kubectl patch deployment metrics-server -n kube-system --type='json' -p='[
{
    "op": "add",
    "path": "/spec/template/spec/containers/0/args/-",
    "value": "--kubelet-insecure-tls"
}
]'

# Wait for metrics-server to be ready
kubectl wait --for=condition=ready pod -l k8s-app=metrics-server -n kube-system --timeout=120s

# Create namespace for swarm-operator
echo "📁 Creating swarm-operator namespace..."
kubectl create namespace swarm-operator --dry-run=client -o yaml | kubectl apply -f -

# Install NGINX Ingress Controller (optional, for exposing services)
echo "🌐 Installing NGINX Ingress Controller..."
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

# Wait for ingress controller to be ready
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=120s

echo "✅ Local cluster setup complete!"
echo ""
echo "📋 Cluster Information:"
echo "  - Name: ${CLUSTER_NAME}"
echo "  - Type: $(if [[ "$USE_KIND" == "true" ]]; then echo "Kind"; else echo "Minikube"; fi)"
echo "  - Nodes: $(kubectl get nodes --no-headers | wc -l)"
echo "  - Context: $(kubectl config current-context)"
echo ""
echo "🚀 Next steps:"
echo "  1. Build operator image: make docker-build"
echo "  2. Load image to cluster: make docker-load"
echo "  3. Deploy operator: ./scripts/deploy-operator.sh"
echo "  4. Run tests: ./scripts/run-tests.sh"