#!/bin/bash
set -euo pipefail

# Validation script for swarm-operator deployment

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Validation results
CHECKS_PASSED=0
CHECKS_FAILED=0
WARNINGS=0

echo "🔍 Swarm Operator Deployment Validation"
echo "======================================"
echo ""

# Function to run validation check
check() {
    local check_name=$1
    local check_command=$2
    local required=${3:-true}
    
    echo -n "Checking ${check_name}... "
    
    if eval "${check_command}" >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
        ((CHECKS_PASSED++))
        return 0
    else
        if [[ "${required}" == "true" ]]; then
            echo -e "${RED}✗${NC}"
            ((CHECKS_FAILED++))
            return 1
        else
            echo -e "${YELLOW}⚠${NC} (optional)"
            ((WARNINGS++))
            return 2
        fi
    fi
}

# Function to check command exists
check_command() {
    local cmd=$1
    local name=$2
    check "${name}" "command -v ${cmd}"
}

# Function to get resource count
get_resource_count() {
    local resource=$1
    local namespace=${2:-""}
    local label=${3:-""}
    
    local cmd="kubectl get ${resource}"
    [[ -n "${namespace}" ]] && cmd="${cmd} -n ${namespace}"
    [[ -n "${label}" ]] && cmd="${cmd} -l ${label}"
    
    ${cmd} --no-headers 2>/dev/null | wc -l
}

echo "1. Prerequisites Check"
echo "----------------------"
check_command "kubectl" "kubectl installed"
check_command "helm" "helm installed" false
check_command "docker" "docker installed"
check "Kubernetes cluster connection" "kubectl cluster-info"
echo ""

echo "2. CRD Installation Check"
echo "------------------------"
check "Swarm CRD" "kubectl get crd swarms.swarm.cloudflow.io"
check "Agent CRD" "kubectl get crd agents.swarm.cloudflow.io"
check "Task CRD" "kubectl get crd tasks.swarm.cloudflow.io"
echo ""

echo "3. Operator Deployment Check"
echo "---------------------------"
check "Operator namespace" "kubectl get namespace swarm-operator"
check "Operator deployment" "kubectl get deployment -n swarm-operator swarm-operator"
check "Operator running" "kubectl get deployment -n swarm-operator swarm-operator -o jsonpath='{.status.readyReplicas}' | grep -v '^0$'"
check "Operator service account" "kubectl get serviceaccount -n swarm-operator swarm-operator"
check "Operator RBAC" "kubectl get clusterrole swarm-operator-role"
echo ""

echo "4. Operator Health Check"
echo "-----------------------"
OPERATOR_POD=$(kubectl get pods -n swarm-operator -l app.kubernetes.io/name=swarm-operator -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [[ -n "${OPERATOR_POD}" ]]; then
    check "Operator pod ready" "kubectl get pod -n swarm-operator ${OPERATOR_POD} -o jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}' | grep True"
    check "Operator logs clean" "! kubectl logs -n swarm-operator ${OPERATOR_POD} --tail=20 | grep -i 'error\\|fatal'"
    check "Metrics endpoint" "kubectl exec -n swarm-operator ${OPERATOR_POD} -- wget -q -O- http://localhost:8080/metrics | grep swarm_operator" false
else
    echo -e "Operator pod ready... ${RED}✗${NC} (no pod found)"
    ((CHECKS_FAILED++))
fi
echo ""

echo "5. Resource Validation"
echo "---------------------"
# Check if we can create resources
check "Can create swarms" "kubectl auth can-i create swarms.swarm.cloudflow.io"
check "Can create agents" "kubectl auth can-i create agents.swarm.cloudflow.io"
check "Can create tasks" "kubectl auth can-i create tasks.swarm.cloudflow.io"
echo ""

echo "6. Test Resource Creation"
echo "------------------------"
# Create test namespace
TEST_NS="swarm-validation-test-$$"
echo "Creating test namespace: ${TEST_NS}"

if kubectl create namespace "${TEST_NS}" >/dev/null 2>&1; then
    # Try to create a test swarm
    cat <<EOF | kubectl apply -f - >/dev/null 2>&1 && \
    check "Create test swarm" "kubectl get swarm -n ${TEST_NS} test-swarm" || \
    check "Create test swarm" "false"
apiVersion: swarm.cloudflow.io/v1alpha1
kind: Swarm
metadata:
  name: test-swarm
  namespace: ${TEST_NS}
spec:
  topology: mesh
  maxAgents: 2
  strategy: balanced
EOF
    
    # Wait a moment for agents to be created
    sleep 5
    
    # Check if agents were created
    AGENT_COUNT=$(get_resource_count "agents" "${TEST_NS}" "swarm=test-swarm")
    if [[ ${AGENT_COUNT} -gt 0 ]]; then
        echo -e "Agents created... ${GREEN}✓${NC} (${AGENT_COUNT} agents)"
        ((CHECKS_PASSED++))
    else
        echo -e "Agents created... ${YELLOW}⚠${NC} (0 agents - may need more time)"
        ((WARNINGS++))
    fi
    
    # Cleanup
    kubectl delete namespace "${TEST_NS}" --wait=false >/dev/null 2>&1
else
    echo -e "Create test namespace... ${RED}✗${NC}"
    ((CHECKS_FAILED++))
fi
echo ""

echo "7. Monitoring & Metrics"
echo "----------------------"
check "Metrics server installed" "kubectl get deployment -n kube-system metrics-server" false
check "ServiceMonitor CRD" "kubectl get crd servicemonitors.monitoring.coreos.com" false
check "Prometheus operator" "kubectl get deployment -n monitoring prometheus-operator" false
echo ""

echo "8. CLI Tool Check"
echo "----------------"
if [[ -f "./bin/swarmctl" ]]; then
    check "swarmctl binary exists" "test -x ./bin/swarmctl"
    check "swarmctl version" "./bin/swarmctl version"
else
    echo -e "swarmctl binary exists... ${YELLOW}⚠${NC} (not built)"
    ((WARNINGS++))
fi
echo ""

echo "9. Documentation Check"
echo "--------------------"
check "Deployment docs" "test -f docs/deployment.md"
check "Quickstart guide" "test -f docs/quickstart.md"
check "Troubleshooting guide" "test -f docs/troubleshooting.md"
check "Examples present" "test -d examples && ls examples/*.yaml >/dev/null 2>&1"
echo ""

echo "10. Script Permissions"
echo "--------------------"
check "local-setup.sh executable" "test -x scripts/local-setup.sh"
check "deploy-operator.sh executable" "test -x scripts/deploy-operator.sh"
check "run-tests.sh executable" "test -x scripts/run-tests.sh"
check "demo.sh executable" "test -x scripts/demo.sh"
echo ""

# Summary
echo "======================================"
echo "Validation Summary"
echo "======================================"
echo -e "Checks Passed: ${GREEN}${CHECKS_PASSED}${NC}"
echo -e "Checks Failed: ${RED}${CHECKS_FAILED}${NC}"
echo -e "Warnings: ${YELLOW}${WARNINGS}${NC}"
echo ""

if [[ ${CHECKS_FAILED} -eq 0 ]]; then
    echo -e "${GREEN}✅ Deployment validation successful!${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Create a swarm: kubectl apply -f examples/basic-swarm.yaml"
    echo "2. Submit a task: kubectl apply -f examples/task-workflow.yaml"
    echo "3. View the demo: ./scripts/demo.sh"
    exit 0
else
    echo -e "${RED}❌ Deployment validation failed!${NC}"
    echo ""
    echo "Please address the failed checks above."
    echo "Run './scripts/deploy-operator.sh' to deploy the operator."
    echo "Check './docs/troubleshooting.md' for common issues."
    exit 1
fi