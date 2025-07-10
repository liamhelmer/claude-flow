#!/bin/bash

# Enhanced Swarm Operator Deployment Script
# This script deploys the enhanced swarm operator with all features

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE=${NAMESPACE:-"swarm-system"}
EXECUTOR_IMAGE=${EXECUTOR_IMAGE:-"claude-flow/swarm-executor:latest"}
OPERATOR_IMAGE=${OPERATOR_IMAGE:-"claude-flow/swarm-operator:enhanced-v0.5.0"}
STORAGE_CLASS=${STORAGE_CLASS:-"standard"}

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed. Please install kubectl first."
        exit 1
    fi
    
    # Check cluster connection
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

create_namespace() {
    log_info "Creating namespace ${NAMESPACE}..."
    kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -
    log_success "Namespace ${NAMESPACE} ready"
}

deploy_crds() {
    log_info "Deploying enhanced CRDs..."
    
    # Apply enhanced SwarmTask CRD
    kubectl apply -f enhanced-swarmtask-crd.yaml
    
    # Apply other CRDs if they exist
    if [ -f "../deploy/crds/swarmcluster-crd.yaml" ]; then
        kubectl apply -f ../deploy/crds/swarmcluster-crd.yaml
    fi
    
    if [ -f "../deploy/crds/agent-crd.yaml" ]; then
        kubectl apply -f ../deploy/crds/agent-crd.yaml
    fi
    
    log_success "CRDs deployed successfully"
}

create_sample_secrets() {
    log_info "Creating sample secrets (replace with your actual credentials)..."
    
    # GitHub credentials (example)
    kubectl create secret generic github-credentials \
        --from-literal=username=your-github-username \
        --from-literal=token=your-github-token \
        --from-literal=email=your-email@example.com \
        --namespace=default \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # Cloud provider examples (commented out - uncomment and modify as needed)
    
    # GCP Service Account
    # kubectl create secret generic gcp-credentials \
    #     --from-file=key.json=/path/to/service-account-key.json \
    #     --namespace=default
    
    # AWS Credentials
    # kubectl create secret generic aws-credentials \
    #     --from-literal=access-key-id=YOUR_ACCESS_KEY \
    #     --from-literal=secret-access-key=YOUR_SECRET_KEY \
    #     --from-literal=region=us-west-2 \
    #     --namespace=default
    
    # Azure Credentials
    # kubectl create secret generic azure-credentials \
    #     --from-literal=client-id=YOUR_CLIENT_ID \
    #     --from-literal=client-secret=YOUR_CLIENT_SECRET \
    #     --from-literal=tenant-id=YOUR_TENANT_ID \
    #     --namespace=default
    
    log_warning "Sample secrets created. Please update with actual credentials."
}

deploy_operator() {
    log_info "Deploying enhanced swarm operator..."
    
    # Apply the deployment manifest with environment variable substitution
    cat enhanced-operator-deployment.yaml | \
        sed "s|claude-flow/swarm-executor:latest|${EXECUTOR_IMAGE}|g" | \
        sed "s|claude-flow/swarm-operator:enhanced-v0.5.0|${OPERATOR_IMAGE}|g" | \
        sed "s|standard|${STORAGE_CLASS}|g" | \
        kubectl apply -f -
    
    log_success "Enhanced operator deployed"
}

wait_for_operator() {
    log_info "Waiting for operator to be ready..."
    
    kubectl rollout status deployment/swarm-operator -n ${NAMESPACE} --timeout=300s
    
    # Check if operator pod is running
    OPERATOR_POD=$(kubectl get pods -n ${NAMESPACE} -l app=swarm-operator -o jsonpath='{.items[0].metadata.name}')
    if [ -z "$OPERATOR_POD" ]; then
        log_error "Operator pod not found"
        exit 1
    fi
    
    log_success "Operator is ready: ${OPERATOR_POD}"
}

create_sample_swarm() {
    log_info "Creating sample SwarmCluster..."
    
    kubectl apply -f - <<EOF
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: sample-swarm
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
    replicas: 1
  - type: analyst
    replicas: 1
EOF
    
    log_success "Sample SwarmCluster created"
}

create_sample_task() {
    log_info "Creating sample SwarmTask..."
    
    kubectl apply -f - <<EOF
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: sample-task
  namespace: default
spec:
  task: "Verify enhanced swarm operator installation"
  swarmRef: sample-swarm
  priority: medium
  strategy: adaptive
  timeout: 5m
  config:
    executorImage: ${EXECUTOR_IMAGE}
    resources:
      requests:
        cpu: "100m"
        memory: "256Mi"
      limits:
        cpu: "500m"
        memory: "512Mi"
EOF
    
    log_success "Sample SwarmTask created"
}

print_status() {
    echo ""
    echo "====================================="
    echo "Enhanced Swarm Operator Deployment"
    echo "====================================="
    echo ""
    
    log_info "Checking deployment status..."
    
    echo ""
    echo "Operator Status:"
    kubectl get deployment -n ${NAMESPACE} swarm-operator
    
    echo ""
    echo "Operator Pods:"
    kubectl get pods -n ${NAMESPACE} -l app=swarm-operator
    
    echo ""
    echo "SwarmClusters:"
    kubectl get swarmclusters -A
    
    echo ""
    echo "SwarmTasks:"
    kubectl get swarmtasks -A
    
    echo ""
    echo "====================================="
    log_success "Deployment complete!"
    echo ""
    echo "Next steps:"
    echo "1. Update secrets with actual credentials:"
    echo "   kubectl edit secret github-credentials -n default"
    echo ""
    echo "2. Build and push the enhanced executor image:"
    echo "   docker build -f Dockerfile.swarm-executor -t ${EXECUTOR_IMAGE} ."
    echo "   docker push ${EXECUTOR_IMAGE}"
    echo ""
    echo "3. Create your own SwarmTasks using the examples in:"
    echo "   examples/enhanced-task-examples.yaml"
    echo ""
    echo "4. Monitor task execution:"
    echo "   kubectl logs -f job/swarm-job-<task-name>"
    echo ""
    echo "5. View operator logs:"
    echo "   kubectl logs -f -n ${NAMESPACE} -l app=swarm-operator"
    echo ""
}

# Main execution
main() {
    echo "====================================="
    echo "Enhanced Swarm Operator Deployment"
    echo "====================================="
    echo ""
    echo "Configuration:"
    echo "  Namespace: ${NAMESPACE}"
    echo "  Executor Image: ${EXECUTOR_IMAGE}"
    echo "  Operator Image: ${OPERATOR_IMAGE}"
    echo "  Storage Class: ${STORAGE_CLASS}"
    echo ""
    
    check_prerequisites
    create_namespace
    deploy_crds
    create_sample_secrets
    deploy_operator
    wait_for_operator
    create_sample_swarm
    create_sample_task
    print_status
}

# Run main function
main "$@"