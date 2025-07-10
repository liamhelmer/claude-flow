#!/bin/bash
set -euo pipefail

# Interactive demo script for swarm-operator

DEMO_NAMESPACE="${DEMO_NAMESPACE:-swarm-demo}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to print demo header
print_header() {
    echo -e "\n${PURPLE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${PURPLE}═══════════════════════════════════════════════════════════════${NC}\n"
}

# Function to print step
print_step() {
    echo -e "${BLUE}▶ $1${NC}"
}

# Function to wait for user
wait_for_user() {
    echo -e "\n${YELLOW}Press ENTER to continue...${NC}"
    read -r
}

# Function to run command with output
run_command() {
    echo -e "${GREEN}\$ $1${NC}"
    eval "$1"
}

# Cleanup function
cleanup_demo() {
    echo -e "\n${YELLOW}Cleaning up demo resources...${NC}"
    kubectl delete namespace "${DEMO_NAMESPACE}" --ignore-not-found=true --wait=false
}

# Main demo
clear
echo -e "${CYAN}"
cat << "EOF"
   _____                                 ____                       __            
  / ___/      ______ __________ ___     / __ \____  ___  _________ _/ /_____  _____
  \__ \ | /| / / __ `/ ___/ __ `__ \   / / / / __ \/ _ \/ ___/ __ `/ __/ __ \/ ___/
 ___/ / |/ |/ / /_/ / /  / / / / / /  / /_/ / /_/ /  __/ /  / /_/ / /_/ /_/ / /    
/____/|__/|__/\__,_/_/  /_/ /_/ /_/   \____/ .___/\___/_/   \__,_/\__/\____/_/     
                                           /_/                                      
EOF
echo -e "${NC}"
echo -e "${PURPLE}Welcome to the Swarm Operator Demo!${NC}"
echo -e "${YELLOW}This demo will showcase the key features of the swarm-operator.${NC}"

wait_for_user

# Step 1: Check Prerequisites
print_header "Step 1: Checking Prerequisites"
print_step "Verifying Kubernetes cluster connection..."
run_command "kubectl cluster-info"
echo ""
print_step "Checking swarm-operator installation..."
run_command "kubectl get deployment -n swarm-operator swarm-operator"
wait_for_user

# Step 2: Create Demo Namespace
print_header "Step 2: Creating Demo Environment"
print_step "Creating demo namespace..."
run_command "kubectl create namespace ${DEMO_NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -"
wait_for_user

# Step 3: Create Basic Swarm
print_header "Step 3: Creating a Basic Mesh Swarm"
print_step "Deploying a simple mesh topology swarm..."
cat > /tmp/demo-mesh-swarm.yaml << EOF
apiVersion: swarm.cloudflow.io/v1alpha1
kind: Swarm
metadata:
  name: demo-mesh-swarm
  namespace: ${DEMO_NAMESPACE}
spec:
  topology: mesh
  maxAgents: 4
  strategy: balanced
  resources:
    limits:
      cpu: "200m"
      memory: "256Mi"
    requests:
      cpu: "100m"
      memory: "128Mi"
EOF

run_command "cat /tmp/demo-mesh-swarm.yaml"
echo ""
run_command "kubectl apply -f /tmp/demo-mesh-swarm.yaml"
echo ""
print_step "Waiting for swarm to be ready..."
run_command "kubectl wait --for=condition=Ready swarm/demo-mesh-swarm -n ${DEMO_NAMESPACE} --timeout=60s"
echo ""
print_step "Checking swarm status..."
run_command "kubectl get swarm demo-mesh-swarm -n ${DEMO_NAMESPACE}"
wait_for_user

# Step 4: Show Agents
print_header "Step 4: Viewing Swarm Agents"
print_step "Listing created agents..."
run_command "kubectl get agents -n ${DEMO_NAMESPACE}"
echo ""
print_step "Describing an agent..."
FIRST_AGENT=$(kubectl get agents -n ${DEMO_NAMESPACE} -o jsonpath='{.items[0].metadata.name}')
run_command "kubectl describe agent ${FIRST_AGENT} -n ${DEMO_NAMESPACE} | head -20"
wait_for_user

# Step 5: Create Task
print_header "Step 5: Creating and Executing a Task"
print_step "Creating a parallel task..."
cat > /tmp/demo-task.yaml << EOF
apiVersion: swarm.cloudflow.io/v1alpha1
kind: Task
metadata:
  name: demo-analysis-task
  namespace: ${DEMO_NAMESPACE}
spec:
  swarmRef:
    name: demo-mesh-swarm
  description: "Analyze system performance and generate optimization report"
  priority: high
  strategy: parallel
  maxAgents: 3
EOF

run_command "cat /tmp/demo-task.yaml"
echo ""
run_command "kubectl apply -f /tmp/demo-task.yaml"
echo ""
print_step "Monitoring task progress..."
for i in {1..5}; do
    sleep 2
    run_command "kubectl get task demo-analysis-task -n ${DEMO_NAMESPACE}"
done
wait_for_user

# Step 6: Hierarchical Swarm
print_header "Step 6: Creating a Hierarchical Swarm with Auto-scaling"
print_step "Deploying hierarchical topology with auto-scaling..."
cat > /tmp/demo-hierarchical-swarm.yaml << EOF
apiVersion: swarm.cloudflow.io/v1alpha1
kind: Swarm
metadata:
  name: demo-hierarchical-swarm
  namespace: ${DEMO_NAMESPACE}
spec:
  topology: hierarchical
  maxAgents: 8
  strategy: specialized
  autoscaling:
    enabled: true
    minAgents: 3
    maxAgents: 12
    targetCPUUtilization: 70
  coordinatorConfig:
    replicas: 2
    capabilities:
      - task-distribution
      - monitoring
  resources:
    limits:
      cpu: "500m"
      memory: "512Mi"
    requests:
      cpu: "200m"
      memory: "256Mi"
EOF

run_command "cat /tmp/demo-hierarchical-swarm.yaml"
echo ""
run_command "kubectl apply -f /tmp/demo-hierarchical-swarm.yaml"
echo ""
print_step "Waiting for hierarchical swarm..."
run_command "kubectl wait --for=condition=Ready swarm/demo-hierarchical-swarm -n ${DEMO_NAMESPACE} --timeout=60s"
wait_for_user

# Step 7: Complex Task Workflow
print_header "Step 7: Complex Task Workflow"
print_step "Creating a multi-step task workflow..."
cat > /tmp/demo-complex-task.yaml << EOF
apiVersion: swarm.cloudflow.io/v1alpha1
kind: Task
metadata:
  name: demo-complex-workflow
  namespace: ${DEMO_NAMESPACE}
spec:
  swarmRef:
    name: demo-hierarchical-swarm
  description: "Multi-stage data processing pipeline"
  priority: critical
  strategy: adaptive
  maxAgents: 6
  stages:
    - name: data-collection
      description: "Collect data from multiple sources"
      agentTypes: ["researcher"]
    - name: data-analysis
      description: "Analyze collected data"
      agentTypes: ["analyst"]
    - name: optimization
      description: "Generate optimization recommendations"
      agentTypes: ["optimizer"]
EOF

run_command "cat /tmp/demo-complex-task.yaml"
echo ""
run_command "kubectl apply -f /tmp/demo-complex-task.yaml"
wait_for_user

# Step 8: Using CLI
print_header "Step 8: Using the swarmctl CLI"
print_step "Listing all swarms..."
run_command "./bin/swarmctl list swarms -A"
echo ""
print_step "Getting swarm details..."
run_command "./bin/swarmctl get swarm demo-mesh-swarm -n ${DEMO_NAMESPACE}"
echo ""
print_step "Listing tasks..."
run_command "./bin/swarmctl list tasks -n ${DEMO_NAMESPACE}"
wait_for_user

# Step 9: Monitoring
print_header "Step 9: Monitoring and Metrics"
print_step "Checking swarm metrics..."
run_command "kubectl get --raw /apis/metrics.k8s.io/v1beta1/namespaces/${DEMO_NAMESPACE}/pods | jq '.items[0].containers[0].usage' 2>/dev/null || echo 'Metrics server not available'"
echo ""
print_step "Viewing operator metrics endpoint..."
OPERATOR_POD=$(kubectl get pods -n swarm-operator -l app.kubernetes.io/name=swarm-operator -o jsonpath='{.items[0].metadata.name}')
run_command "kubectl exec -n swarm-operator ${OPERATOR_POD} -- wget -q -O- http://localhost:8080/metrics | grep swarm_operator | head -10"
wait_for_user

# Step 10: Cleanup Options
print_header "Step 10: Demo Complete!"
echo -e "${GREEN}✅ Congratulations! You've completed the swarm-operator demo.${NC}"
echo ""
echo -e "${YELLOW}Key Features Demonstrated:${NC}"
echo "  • Multiple swarm topologies (mesh, hierarchical)"
echo "  • Dynamic agent creation and management"
echo "  • Task orchestration with different strategies"
echo "  • Auto-scaling capabilities"
echo "  • CLI tool usage"
echo "  • Monitoring and metrics"
echo ""
echo -e "${CYAN}What's Next?${NC}"
echo "  • Explore more examples in the examples/ directory"
echo "  • Read the documentation in docs/"
echo "  • Try creating your own custom swarms and tasks"
echo "  • Contribute to the project!"
echo ""
echo -e "${YELLOW}Would you like to clean up the demo resources? (y/N)${NC}"
read -r CLEANUP_RESPONSE

if [[ "${CLEANUP_RESPONSE}" =~ ^[Yy]$ ]]; then
    cleanup_demo
    echo -e "${GREEN}✅ Demo resources cleaned up!${NC}"
else
    echo -e "${BLUE}ℹ️  Demo resources left running in namespace: ${DEMO_NAMESPACE}${NC}"
    echo -e "${BLUE}   To clean up later, run: kubectl delete namespace ${DEMO_NAMESPACE}${NC}"
fi

echo -e "\n${PURPLE}Thank you for trying swarm-operator!${NC}"