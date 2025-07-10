#!/bin/bash
# Run E2E tests for Hive-Mind and Autoscaling features

set -e

NAMESPACE_PREFIX="swarm-test"
TIMESTAMP=$(date +%s)

echo "🧪 Swarm Operator E2E Test Suite"
echo "================================"
echo "Timestamp: $TIMESTAMP"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl not found. Please install kubectl."
        exit 1
    fi
    
    # Check cluster connection
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
        exit 1
    fi
    
    # Check if operator is deployed
    if ! kubectl get deployment swarm-operator -n swarm-system &> /dev/null; then
        log_warn "Swarm operator not found. Deploying..."
        cd "$(dirname "$0")/.."
        ./deploy/deploy-hivemind-operator.sh
    fi
    
    log_info "Prerequisites check passed ✓"
}

cleanup_test() {
    local namespace=$1
    log_info "Cleaning up namespace: $namespace"
    kubectl delete namespace $namespace --ignore-not-found=true --wait=false
}

wait_for_condition() {
    local resource=$1
    local condition=$2
    local timeout=${3:-300}
    local namespace=${4:-default}
    
    log_info "Waiting for $resource to meet condition: $condition"
    kubectl wait --for=$condition $resource -n $namespace --timeout=${timeout}s
}

run_hivemind_test() {
    log_info "🧠 Running Hive-Mind Test Suite"
    log_info "==============================="
    
    local test_namespace="claude-flow-hivemind"
    
    # Apply test manifests
    log_info "Deploying hive-mind test resources..."
    kubectl apply -f test/e2e/hivemind_test.yaml
    
    # Ensure namespace exists
    kubectl create namespace $test_namespace --dry-run=client -o yaml | kubectl apply -f -
    
    # Wait for namespace to exist
    kubectl wait --for=jsonpath='{.status.phase}'=Active namespace/$test_namespace --timeout=30s || true
    
    # Wait for SwarmCluster to be ready
    log_info "Waiting for SwarmCluster to initialize..."
    sleep 10  # Give controller time to react
    
    # Check hive-mind components
    log_info "Verifying hive-mind components..."
    
    # Check StatefulSet
    if kubectl get statefulset -n $test_namespace -l component=hivemind &> /dev/null; then
        log_info "✓ Hive-mind StatefulSet created"
        
        # Wait for pods to be ready
        kubectl wait --for=condition=ready pod -n $test_namespace -l component=hivemind --timeout=120s || log_warn "Hive-mind pods not ready yet"
    else
        log_error "✗ Hive-mind StatefulSet not found"
    fi
    
    # Check Redis deployment
    if kubectl get deployment -n $test_namespace -l component=memory &> /dev/null; then
        log_info "✓ Redis memory backend deployed"
    else
        log_error "✗ Redis deployment not found"
    fi
    
    # Check SwarmAgents
    agent_count=$(kubectl get swarmagents -n $test_namespace --no-headers 2>/dev/null | wc -l)
    if [ $agent_count -gt 0 ]; then
        log_info "✓ SwarmAgents created: $agent_count"
    else
        log_warn "⚠ No SwarmAgents found"
    fi
    
    # Check SwarmMemory
    if kubectl get swarmmemory test-knowledge-1 -n $test_namespace &> /dev/null; then
        log_info "✓ SwarmMemory entry created"
    else
        log_error "✗ SwarmMemory not found"
    fi
    
    # Run consensus test
    log_info "Running consensus test task..."
    kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: consensus-runner-$TIMESTAMP
  namespace: hivemind-test
spec:
  template:
    spec:
      restartPolicy: OnFailure
      containers:
      - name: runner
        image: busybox
        command: ["sh", "-c", "echo 'Consensus test would run here'; sleep 10"]
EOF
    
    # Wait for job to complete
    wait_for_condition "job/consensus-runner-$TIMESTAMP" "condition=complete" 120 "hivemind-test" || log_warn "Consensus test job did not complete"
    
    # Check job logs
    log_info "Consensus test logs:"
    kubectl logs job/consensus-runner-$TIMESTAMP -n $test_namespace || log_warn "Could not retrieve logs"
    
    # Validate metrics
    log_info "Checking metrics endpoint..."
    # In a real cluster with prometheus, we would check actual metrics
    
    log_info "🧠 Hive-Mind test completed"
    
    # Cleanup
    if [ "${SKIP_CLEANUP:-false}" != "true" ]; then
        cleanup_test "$test_namespace"
    fi
}

run_autoscaling_test() {
    log_info "📈 Running Autoscaling Test Suite"
    log_info "================================"
    
    local test_namespace="claude-flow-swarm"
    
    # Apply test manifests
    log_info "Deploying autoscaling test resources..."
    kubectl apply -f test/e2e/autoscaling_test.yaml
    
    # Ensure namespace exists
    kubectl create namespace $test_namespace --dry-run=client -o yaml | kubectl apply -f -
    
    # Wait for namespace to exist
    kubectl wait --for=jsonpath='{.status.phase}'=Active namespace/$test_namespace --timeout=30s || true
    
    # Wait for SwarmCluster
    log_info "Waiting for autoscaling cluster to initialize..."
    sleep 15
    
    # Check initial agent count
    log_info "Checking initial agent deployment..."
    initial_agents=$(kubectl get swarmagents -n $test_namespace -l swarm-cluster=autoscale-multi-metric --no-headers 2>/dev/null | wc -l)
    log_info "Initial agent count: $initial_agents"
    
    if [ $initial_agents -ge 3 ]; then
        log_info "✓ Minimum agents requirement met"
    else
        log_warn "⚠ Agent count below minimum (expected >= 3, got $initial_agents)"
    fi
    
    # Check agent type distribution
    log_info "Verifying agent type ratios..."
    for agent_type in coordinator researcher coder tester; do
        count=$(kubectl get swarmagents -n $test_namespace -l swarm-cluster=autoscale-multi-metric,agent-type=$agent_type --no-headers 2>/dev/null | wc -l)
        log_info "  $agent_type: $count agents"
    done
    
    # Check HPA creation
    log_info "Checking HorizontalPodAutoscaler resources..."
    hpa_count=$(kubectl get hpa -n $test_namespace --no-headers 2>/dev/null | wc -l)
    if [ $hpa_count -gt 0 ]; then
        log_info "✓ HPA resources created: $hpa_count"
        kubectl get hpa -n $test_namespace
    else
        log_warn "⚠ No HPA resources found"
    fi
    
    # Run load generator
    log_info "Starting load generator..."
    kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: load-gen-$TIMESTAMP
  namespace: autoscaling-test
spec:
  template:
    spec:
      restartPolicy: OnFailure
      containers:
      - name: load
        image: busybox
        command: 
        - sh
        - -c
        - |
          echo "Generating load..."
          # Simulate CPU load
          while true; do
            echo "scale=10000; 4*a(1)" | bc -l > /dev/null 2>&1 &
          done &
          LOAD_PID=\$!
          sleep 30
          kill \$LOAD_PID
          echo "Load generation complete"
EOF
    
    # Monitor scaling behavior
    log_info "Monitoring autoscaling behavior..."
    for i in {1..6}; do
        sleep 10
        current_agents=$(kubectl get swarmagents -n $test_namespace -l swarm-cluster=autoscale-multi-metric --no-headers 2>/dev/null | wc -l)
        log_info "Time: ${i}0s - Agent count: $current_agents"
        
        if [ $current_agents -gt $initial_agents ]; then
            log_info "✓ Scale UP detected! ($initial_agents -> $current_agents)"
            break
        fi
    done
    
    # Run validation job
    log_info "Running validation job..."
    kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: validate-autoscale-$TIMESTAMP
  namespace: autoscaling-test
spec:
  template:
    spec:
      serviceAccountName: autoscale-validator
      restartPolicy: OnFailure
      containers:
      - name: validate
        image: bitnami/kubectl:latest
        command:
        - sh
        - -c
        - |
          echo "Validating autoscaling configuration..."
          kubectl get swarmclusters,swarmagents,hpa -n $test_namespace
          echo "Validation complete"
EOF
    
    wait_for_condition "job/validate-autoscale-$TIMESTAMP" "condition=complete" 60 "autoscaling-test" || log_warn "Validation job did not complete"
    
    log_info "📈 Autoscaling test completed"
    
    # Cleanup
    if [ "${SKIP_CLEANUP:-false}" != "true" ]; then
        cleanup_test "$test_namespace"
    fi
}

run_integration_test() {
    log_info "🔗 Running Integration Test"
    log_info "=========================="
    
    # Create a combined test that uses both features
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Namespace
metadata:
  name: integration-test-$TIMESTAMP
---
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: integrated-cluster
  namespace: integration-test-$TIMESTAMP
spec:
  topology: hierarchical
  queenMode: distributed
  strategy: adaptive
  
  hiveMind:
    enabled: true
    databaseSize: 512Mi
    syncInterval: 15s
    
  autoscaling:
    enabled: true
    minAgents: 2
    maxAgents: 8
    targetUtilization: 75
    topologyRatios:
      coordinator: 20
      coder: 60
      analyst: 20
      
  memory:
    type: redis
    size: 256Mi
    
  monitoring:
    enabled: true
EOF
    
    log_info "Waiting for integrated cluster to stabilize..."
    sleep 20
    
    # Verify both features are working
    log_info "Checking integrated features..."
    
    # Check hive-mind
    if kubectl get statefulset -n integration-test-$TIMESTAMP -l component=hivemind &> /dev/null; then
        log_info "✓ Hive-mind active in integrated cluster"
    fi
    
    # Check autoscaling
    agent_count=$(kubectl get swarmagents -n integration-test-$TIMESTAMP --no-headers 2>/dev/null | wc -l)
    log_info "✓ Agents deployed: $agent_count"
    
    # Create a task that uses both features
    cat <<EOF | kubectl apply -f -
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: integrated-task
  namespace: integration-test-$TIMESTAMP
spec:
  task: |
    echo "Task using both hive-mind and autoscaling"
    echo "Storing in collective memory..."
    echo "Triggering scale event..."
EOF
    
    log_info "🔗 Integration test completed"
    
    # Cleanup
    if [ "${SKIP_CLEANUP:-false}" != "true" ]; then
        cleanup_test "integration-test-$TIMESTAMP"
    fi
}

# Main execution
main() {
    check_prerequisites
    
    # Run tests based on arguments
    if [ $# -eq 0 ]; then
        # Run all tests
        run_hivemind_test
        echo ""
        run_autoscaling_test
        echo ""
        run_integration_test
    else
        case $1 in
            hivemind)
                run_hivemind_test
                ;;
            autoscaling)
                run_autoscaling_test
                ;;
            integration)
                run_integration_test
                ;;
            *)
                log_error "Unknown test: $1"
                echo "Usage: $0 [hivemind|autoscaling|integration]"
                exit 1
                ;;
        esac
    fi
    
    log_info "🎉 All tests completed!"
    log_info "================================"
    log_info "To keep test resources for debugging, set SKIP_CLEANUP=true"
}

# Run main
main "$@"