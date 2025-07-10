#!/bin/bash
set -euo pipefail

# Script to run comprehensive tests for swarm-operator

TEST_NAMESPACE="${TEST_NAMESPACE:-swarm-operator-test}"
TIMEOUT="${TIMEOUT:-300}"
VERBOSE="${VERBOSE:-false}"

echo "🧪 Running swarm-operator tests..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test result tracking
TESTS_PASSED=0
TESTS_FAILED=0

# Function to run a test
run_test() {
    local test_name=$1
    local test_function=$2
    
    echo -e "\n🔍 Running test: ${test_name}"
    if ${test_function}; then
        echo -e "${GREEN}✅ PASSED${NC}: ${test_name}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}❌ FAILED${NC}: ${test_name}"
        ((TESTS_FAILED++))
    fi
}

# Function to cleanup test resources
cleanup_test_resources() {
    echo "🧹 Cleaning up test resources..."
    kubectl delete namespace "${TEST_NAMESPACE}" --ignore-not-found=true --wait=false
}

# Trap to ensure cleanup on exit
trap cleanup_test_resources EXIT

# Test 1: Verify CRDs are installed
test_crds_installed() {
    echo "Checking CRDs..."
    kubectl get crd swarms.swarm.cloudflow.io >/dev/null 2>&1 && \
    kubectl get crd agents.swarm.cloudflow.io >/dev/null 2>&1 && \
    kubectl get crd tasks.swarm.cloudflow.io >/dev/null 2>&1
}

# Test 2: Verify operator is running
test_operator_running() {
    echo "Checking operator deployment..."
    local ready_replicas=$(kubectl get deployment -n swarm-operator swarm-operator -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    [[ "${ready_replicas}" -gt 0 ]]
}

# Test 3: Create and verify basic swarm
test_create_basic_swarm() {
    echo "Creating test namespace..."
    kubectl create namespace "${TEST_NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -
    
    echo "Creating basic swarm..."
    cat <<EOF | kubectl apply -f -
apiVersion: swarm.cloudflow.io/v1alpha1
kind: Swarm
metadata:
  name: test-basic-swarm
  namespace: ${TEST_NAMESPACE}
spec:
  topology: mesh
  maxAgents: 3
  strategy: balanced
  resources:
    limits:
      cpu: "100m"
      memory: "128Mi"
    requests:
      cpu: "50m"
      memory: "64Mi"
EOF
    
    echo "Waiting for swarm to be ready..."
    local timeout=60
    local start_time=$(date +%s)
    while true; do
        local phase=$(kubectl get swarm test-basic-swarm -n "${TEST_NAMESPACE}" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
        if [[ "${phase}" == "Ready" ]]; then
            return 0
        fi
        
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        if [[ ${elapsed} -gt ${timeout} ]]; then
            echo "Timeout waiting for swarm to be ready"
            kubectl describe swarm test-basic-swarm -n "${TEST_NAMESPACE}"
            return 1
        fi
        
        sleep 2
    done
}

# Test 4: Verify agents are created
test_agents_created() {
    echo "Checking agents..."
    local agent_count=$(kubectl get agents -n "${TEST_NAMESPACE}" --no-headers 2>/dev/null | wc -l)
    echo "Found ${agent_count} agents"
    [[ ${agent_count} -gt 0 ]]
}

# Test 5: Create and execute task
test_create_task() {
    echo "Creating test task..."
    cat <<EOF | kubectl apply -f -
apiVersion: swarm.cloudflow.io/v1alpha1
kind: Task
metadata:
  name: test-task
  namespace: ${TEST_NAMESPACE}
spec:
  swarmRef:
    name: test-basic-swarm
  description: "Test task for E2E validation"
  priority: medium
  strategy: parallel
  maxAgents: 2
EOF
    
    echo "Waiting for task to complete..."
    local timeout=120
    local start_time=$(date +%s)
    while true; do
        local phase=$(kubectl get task test-task -n "${TEST_NAMESPACE}" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
        if [[ "${phase}" == "Completed" || "${phase}" == "Failed" ]]; then
            [[ "${phase}" == "Completed" ]]
            return $?
        fi
        
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        if [[ ${elapsed} -gt ${timeout} ]]; then
            echo "Timeout waiting for task to complete"
            kubectl describe task test-task -n "${TEST_NAMESPACE}"
            return 1
        fi
        
        sleep 2
    done
}

# Test 6: Test hierarchical swarm
test_hierarchical_swarm() {
    echo "Creating hierarchical swarm..."
    cat <<EOF | kubectl apply -f -
apiVersion: swarm.cloudflow.io/v1alpha1
kind: Swarm
metadata:
  name: test-hierarchical-swarm
  namespace: ${TEST_NAMESPACE}
spec:
  topology: hierarchical
  maxAgents: 5
  strategy: specialized
  autoscaling:
    enabled: true
    minAgents: 2
    maxAgents: 10
    targetCPUUtilization: 70
EOF
    
    sleep 10
    local phase=$(kubectl get swarm test-hierarchical-swarm -n "${TEST_NAMESPACE}" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
    [[ "${phase}" == "Ready" ]]
}

# Test 7: Test swarm update
test_swarm_update() {
    echo "Updating swarm configuration..."
    kubectl patch swarm test-basic-swarm -n "${TEST_NAMESPACE}" --type='merge' -p '
    {
      "spec": {
        "maxAgents": 5,
        "strategy": "adaptive"
      }
    }'
    
    sleep 5
    local max_agents=$(kubectl get swarm test-basic-swarm -n "${TEST_NAMESPACE}" -o jsonpath='{.spec.maxAgents}' 2>/dev/null)
    [[ "${max_agents}" == "5" ]]
}

# Test 8: Test resource limits
test_resource_limits() {
    echo "Checking resource limits on agents..."
    local agent_name=$(kubectl get agents -n "${TEST_NAMESPACE}" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [[ -z "${agent_name}" ]]; then
        echo "No agents found"
        return 1
    fi
    
    local cpu_limit=$(kubectl get agent "${agent_name}" -n "${TEST_NAMESPACE}" -o jsonpath='{.spec.resources.limits.cpu}' 2>/dev/null)
    [[ -n "${cpu_limit}" ]]
}

# Test 9: Test monitoring endpoints
test_monitoring_endpoints() {
    echo "Checking metrics endpoint..."
    local operator_pod=$(kubectl get pods -n swarm-operator -l app.kubernetes.io/name=swarm-operator -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [[ -z "${operator_pod}" ]]; then
        echo "Operator pod not found"
        return 1
    fi
    
    # Check if metrics endpoint responds
    kubectl exec -n swarm-operator "${operator_pod}" -- wget -q -O- http://localhost:8080/metrics | grep -q "swarm_operator"
}

# Test 10: Test CLI functionality
test_cli_commands() {
    echo "Testing CLI commands..."
    
    # Test list swarms
    ./bin/swarmctl list swarms -n "${TEST_NAMESPACE}" >/dev/null 2>&1 || return 1
    
    # Test list agents
    ./bin/swarmctl list agents -n "${TEST_NAMESPACE}" >/dev/null 2>&1 || return 1
    
    # Test list tasks
    ./bin/swarmctl list tasks -n "${TEST_NAMESPACE}" >/dev/null 2>&1 || return 1
    
    return 0
}

# Main test execution
echo "🏁 Starting E2E tests for swarm-operator..."
echo "Test namespace: ${TEST_NAMESPACE}"
echo ""

# Create test namespace
kubectl create namespace "${TEST_NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

# Run all tests
run_test "CRDs Installed" test_crds_installed
run_test "Operator Running" test_operator_running
run_test "Create Basic Swarm" test_create_basic_swarm
run_test "Agents Created" test_agents_created
run_test "Create and Execute Task" test_create_task
run_test "Hierarchical Swarm" test_hierarchical_swarm
run_test "Swarm Update" test_swarm_update
run_test "Resource Limits" test_resource_limits
run_test "Monitoring Endpoints" test_monitoring_endpoints
run_test "CLI Commands" test_cli_commands

# Summary
echo -e "\n📊 Test Summary:"
echo -e "  ${GREEN}Passed${NC}: ${TESTS_PASSED}"
echo -e "  ${RED}Failed${NC}: ${TESTS_FAILED}"
echo -e "  Total: $((TESTS_PASSED + TESTS_FAILED))"

if [[ ${TESTS_FAILED} -eq 0 ]]; then
    echo -e "\n${GREEN}✅ All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}❌ Some tests failed!${NC}"
    exit 1
fi