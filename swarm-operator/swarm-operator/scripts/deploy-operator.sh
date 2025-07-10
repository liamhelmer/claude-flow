#!/bin/bash
set -euo pipefail

# Script to deploy swarm-operator using Helm

NAMESPACE="${NAMESPACE:-swarm-operator}"
RELEASE_NAME="${RELEASE_NAME:-swarm-operator}"
CHART_PATH="${CHART_PATH:-./deploy/helm/swarm-operator}"
VALUES_FILE="${VALUES_FILE:-}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
CLUSTER_TYPE="${CLUSTER_TYPE:-kind}"
CLUSTER_NAME="${CLUSTER_NAME:-swarm-operator-test}"

echo "🚀 Deploying swarm-operator to Kubernetes cluster..."

# Function to load image to Kind cluster
load_image_to_kind() {
    local image=$1
    echo "📦 Loading image ${image} to Kind cluster..."
    kind load docker-image "${image}" --name "${CLUSTER_NAME}"
}

# Function to load image to Minikube
load_image_to_minikube() {
    local image=$1
    echo "📦 Loading image ${image} to Minikube cluster..."
    minikube image load "${image}" --profile="${CLUSTER_NAME}"
}

# Check if kubectl is configured
if ! kubectl cluster-info &>/dev/null; then
    echo "❌ kubectl is not configured to connect to a cluster"
    echo "Run ./scripts/local-setup.sh first"
    exit 1
fi

# Ensure namespace exists
echo "📁 Ensuring namespace ${NAMESPACE} exists..."
kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

# Build operator image if not exists
IMAGE_NAME="swarm-operator:${IMAGE_TAG}"
if ! docker images | grep -q "swarm-operator.*${IMAGE_TAG}"; then
    echo "🔨 Building operator image..."
    make docker-build IMG="${IMAGE_NAME}"
fi

# Load image to cluster
if [[ "${CLUSTER_TYPE}" == "kind" ]]; then
    load_image_to_kind "${IMAGE_NAME}"
elif [[ "${CLUSTER_TYPE}" == "minikube" ]]; then
    load_image_to_minikube "${IMAGE_NAME}"
else
    echo "⚠️  Unknown cluster type: ${CLUSTER_TYPE}. Assuming image is already available in cluster."
fi

# Apply CRDs
echo "📋 Installing CRDs..."
kubectl apply -f config/crd/bases/

# Wait for CRDs to be established
echo "⏳ Waiting for CRDs to be established..."
kubectl wait --for condition=established --timeout=60s \
    crd/swarms.swarm.cloudflow.io \
    crd/agents.swarm.cloudflow.io \
    crd/tasks.swarm.cloudflow.io

# Prepare Helm values
HELM_VALUES=""
if [[ -n "${VALUES_FILE}" ]]; then
    HELM_VALUES="-f ${VALUES_FILE}"
fi

# Deploy using Helm
echo "🎯 Deploying swarm-operator with Helm..."
helm upgrade --install "${RELEASE_NAME}" "${CHART_PATH}" \
    --namespace "${NAMESPACE}" \
    --set image.repository=swarm-operator \
    --set image.tag="${IMAGE_TAG}" \
    --set image.pullPolicy=IfNotPresent \
    --set replicaCount=1 \
    --set resources.limits.cpu=500m \
    --set resources.limits.memory=512Mi \
    --set resources.requests.cpu=100m \
    --set resources.requests.memory=128Mi \
    --set monitoring.enabled=true \
    --set monitoring.serviceMonitor.enabled=true \
    --set autoscaling.enabled=false \
    ${HELM_VALUES} \
    --wait \
    --timeout 5m

# Wait for operator to be ready
echo "⏳ Waiting for operator to be ready..."
kubectl rollout status deployment/"${RELEASE_NAME}" -n "${NAMESPACE}" --timeout=300s

# Check operator logs
echo "📜 Checking operator logs..."
kubectl logs -n "${NAMESPACE}" -l "app.kubernetes.io/name=swarm-operator" --tail=20

# Verify operator is running
POD_COUNT=$(kubectl get pods -n "${NAMESPACE}" -l "app.kubernetes.io/name=swarm-operator" --no-headers | grep Running | wc -l)
if [[ "${POD_COUNT}" -eq 0 ]]; then
    echo "❌ Operator is not running properly"
    kubectl describe pods -n "${NAMESPACE}" -l "app.kubernetes.io/name=swarm-operator"
    exit 1
fi

echo "✅ Swarm operator deployed successfully!"
echo ""
echo "📋 Deployment Information:"
echo "  - Namespace: ${NAMESPACE}"
echo "  - Release: ${RELEASE_NAME}"
echo "  - Image: ${IMAGE_NAME}"
echo "  - Replicas: $(kubectl get deployment "${RELEASE_NAME}" -n "${NAMESPACE}" -o jsonpath='{.spec.replicas}')"
echo ""
echo "🔍 Useful commands:"
echo "  - View logs: kubectl logs -n ${NAMESPACE} -l app.kubernetes.io/name=swarm-operator -f"
echo "  - Check status: kubectl get all -n ${NAMESPACE}"
echo "  - Create swarm: kubectl apply -f examples/basic-swarm.yaml"
echo "  - List swarms: kubectl get swarms -A"
echo ""
echo "🚀 Next steps:"
echo "  1. Create a swarm: kubectl apply -f examples/basic-swarm.yaml"
echo "  2. Run tests: ./scripts/run-tests.sh"
echo "  3. View demo: ./scripts/demo.sh"