#!/bin/bash

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}"
echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║                                                               ║"
echo "║           Claude Flow Swarm Operator Demo                     ║"
echo "║                                                               ║"
echo "║          Managing AI Agent Swarms in Kubernetes               ║"
echo "║                                                               ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

echo -e "${GREEN}✅ Swarm Operator is deployed and running!${NC}"
echo ""

# Show operator status
echo -e "${YELLOW}📊 Operator Status:${NC}"
kubectl -n swarm-system get deployment swarm-operator
echo ""

# Show CRDs
echo -e "${YELLOW}📋 Custom Resource Definitions:${NC}"
kubectl get crds | grep claudeflow.io
echo ""

# Show existing swarms
echo -e "${YELLOW}🐝 Current Swarms:${NC}"
kubectl get swarmclusters -A
echo ""

# Create a demo swarm
echo -e "${YELLOW}🚀 Creating a hierarchical swarm with 5 agents...${NC}"
cat <<EOF | kubectl apply -f -
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: demo-hierarchical-swarm
  namespace: default
spec:
  topology: hierarchical
  agentCount: 5
  agentTemplate:
    type: coordinator
    capabilities:
    - "orchestration"
    - "task-distribution"
    - "monitoring"
    resources:
      requests:
        cpu: "200m"
        memory: "256Mi"
  taskDistribution:
    strategy: capability-based
    maxTasksPerAgent: 3
  autoScaling:
    enabled: true
    minAgents: 3
    maxAgents: 10
    metrics:
    - type: taskQueue
      targetValue: "5"
EOF

sleep 2

# Show the new swarm
echo ""
echo -e "${GREEN}✅ Hierarchical swarm created!${NC}"
echo ""
kubectl get swarmclusters demo-hierarchical-swarm -o wide
echo ""

# Create a complex task
echo -e "${YELLOW}📝 Submitting a complex multi-stage task...${NC}"
cat <<EOF | kubectl apply -f -
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: demo-complex-task
  namespace: default
spec:
  swarmRef: demo-hierarchical-swarm
  task: "Analyze and optimize the Kubernetes cluster for AI workloads"
  priority: critical
  strategy: parallel
  timeout: "30m"
  subtasks:
  - name: "cluster-analysis"
    description: "Analyze current cluster resource usage"
  - name: "workload-profiling"
    description: "Profile AI workload requirements"
  - name: "optimization-plan"
    description: "Create optimization recommendations"
    dependencies:
    - "cluster-analysis"
    - "workload-profiling"
  - name: "implementation"
    description: "Implement optimization changes"
    dependencies:
    - "optimization-plan"
EOF

sleep 2

echo ""
echo -e "${GREEN}✅ Complex task submitted!${NC}"
echo ""

# Show all resources
echo -e "${YELLOW}📊 All Swarm Resources:${NC}"
kubectl get swarmclusters,swarmtasks -A
echo ""

# Show metrics
echo -e "${YELLOW}📈 Operator Metrics:${NC}"
kubectl -n swarm-system exec deployment/swarm-operator -- wget -qO- http://localhost:8080/metrics | head -10
echo ""

# Create example agent (simulated)
echo -e "${YELLOW}🤖 Simulating agent creation...${NC}"
kubectl -n swarm-system apply -f examples/demo-swarm-config.yaml
echo -e "${GREEN}✅ Demo agent configuration created${NC}"
echo ""

# Show operator logs
echo -e "${YELLOW}📜 Recent Operator Logs:${NC}"
kubectl -n swarm-system logs deployment/swarm-operator --tail=5
echo ""

echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}🎉 Demo Complete!${NC}"
echo ""
echo "Next steps:"
echo "1. Check swarm status: kubectl describe swarmcluster demo-hierarchical-swarm"
echo "2. Watch task progress: kubectl describe swarmtask demo-complex-task"
echo "3. View operator logs: kubectl -n swarm-system logs -f deployment/swarm-operator"
echo "4. Create your own swarm: kubectl apply -f examples/basic-swarm.yaml"
echo ""
echo -e "${YELLOW}To clean up demo resources:${NC}"
echo "kubectl delete swarmcluster demo-hierarchical-swarm"
echo "kubectl delete swarmtask demo-complex-task"
echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"