# Swarm Operator Testing Guide

## Overview

This guide covers testing the Hive-Mind and Autoscaling features of the Enhanced Swarm Operator v3.0.0. We provide both automated E2E tests and manual testing procedures.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Running Automated Tests](#running-automated-tests)
3. [Manual Testing](#manual-testing)
4. [Test Scenarios](#test-scenarios)
5. [Troubleshooting](#troubleshooting)
6. [Performance Testing](#performance-testing)

## Prerequisites

Before running tests, ensure you have:

- Kubernetes cluster (1.26+) - local or remote
- kubectl configured and connected
- Docker for building images
- Metrics Server installed (for autoscaling tests)
- (Optional) Prometheus Operator for advanced monitoring

### Quick Setup

```bash
# For local testing with kind
kind create cluster --name swarm-test --config kind-config.yaml

# Install metrics server
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# For kind clusters, patch metrics server for insecure TLS
kubectl patch -n kube-system deployment metrics-server --type=json -p '[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'
```

## Running Automated Tests

### Quick Test

Run all E2E tests:
```bash
cd swarm-operator
./test/run-tests.sh
```

### Individual Test Suites

```bash
# Test only hive-mind features
./test/run-tests.sh hivemind

# Test only autoscaling features  
./test/run-tests.sh autoscaling

# Test integrated features
./test/run-tests.sh integration
```

### Keep Test Resources for Debugging

```bash
SKIP_CLEANUP=true ./test/run-tests.sh
```

### Integration Tests

Run Go integration tests:
```bash
cd swarm-operator
go test ./test/integration/... -v
```

## Manual Testing

### 1. Deploy the Operator

```bash
./deploy/deploy-hivemind-operator.sh
```

Verify deployment:
```bash
kubectl -n swarm-system get all
kubectl -n swarm-system logs deployment/swarm-operator -f
```

### 2. Test Hive-Mind Features

#### Basic Hive-Mind Cluster

```yaml
kubectl apply -f - <<EOF
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: manual-hivemind-test
spec:
  topology: mesh
  queenMode: distributed
  strategy: consensus
  consensusThreshold: 0.75
  
  hiveMind:
    enabled: true
    databaseSize: 2Gi
    syncInterval: 20s
    backupEnabled: true
    backupInterval: 10m
    
  memory:
    type: redis
    size: 1Gi
    persistence: true
EOF
```

Verify hive-mind components:
```bash
# Check StatefulSet
kubectl get statefulset -l swarm-cluster=manual-hivemind-test

# Check hive-mind pods
kubectl get pods -l component=hivemind

# Check sync status
kubectl exec -it manual-hivemind-test-hivemind-0 -- curl localhost:8080/status
```

#### Test Collective Memory

```yaml
kubectl apply -f - <<EOF
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmMemory
metadata:
  name: collective-knowledge
spec:
  clusterRef: manual-hivemind-test
  namespace: research
  type: knowledge
  key: "findings/ml-optimization"
  value: |
    {
      "algorithm": "gradient-descent",
      "learning_rate": 0.01,
      "batch_size": 32,
      "improvements": ["momentum", "adaptive-lr"],
      "confidence": 0.92
    }
  ttl: 0  # Permanent
  priority: 100
  sharedWith: []  # All agents
EOF
```

Verify memory storage:
```bash
# Check Redis
kubectl exec -it deployment/manual-hivemind-test-redis -- redis-cli

# In Redis CLI:
KEYS *
GET "memory:research:findings/ml-optimization"
```

#### Test Consensus Decision

Create multiple agents and a consensus task:
```bash
# Apply test agents
kubectl apply -f examples/hivemind-cluster.yaml

# Create consensus task
kubectl apply -f - <<EOF
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: consensus-decision
spec:
  task: |
    echo "Proposing architecture change..."
    # Agents will vote on this proposal
EOF
```

Monitor consensus:
```bash
# Watch agent votes
kubectl logs -l component=agent -f | grep -i consensus

# Check decision outcome
kubectl get swarmtask consensus-decision -o yaml
```

### 3. Test Autoscaling Features

#### Multi-Metric Autoscaling

```yaml
kubectl apply -f - <<EOF
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: manual-autoscale-test
spec:
  topology: hierarchical
  
  autoscaling:
    enabled: true
    minAgents: 2
    maxAgents: 10
    targetUtilization: 60
    
    topologyRatios:
      coordinator: 10
      coder: 70
      tester: 20
      
    metrics:
    - type: cpu
      target: "60"
    - type: memory  
      target: "70"
    - type: custom
      name: queue_depth
      target: "10"
      
  agentTemplate:
    resources:
      cpu: "100m"
      memory: "128Mi"
EOF
```

Generate load to trigger scaling:
```bash
# Create load generator pod
kubectl run load-gen --image=busybox --command -- sh -c "while true; do echo scale > /dev/null; done"

# Watch scaling events
kubectl get hpa -w
kubectl get swarmagents -l swarm-cluster=manual-autoscale-test -w

# Check metrics
kubectl top pods -l component=agent
```

#### Verify Topology Ratios

```bash
# Count agents by type
for type in coordinator coder tester; do
  count=$(kubectl get swarmagents -l swarm-cluster=manual-autoscale-test,agent-type=$type --no-headers | wc -l)
  echo "$type: $count"
done

# Verify ratios are maintained during scaling
```

#### Test Predictive Scaling

Enable predictive scaling with neural models:
```yaml
kubectl apply -f - <<EOF
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: predictive-scaling-test
spec:
  topology: star
  
  autoscaling:
    enabled: true
    minAgents: 3
    maxAgents: 15
    metrics:
    - type: custom
      name: predicted_load_15m
      target: "0.7"
      
  neural:
    enabled: true
    models:
    - name: load-predictor-lstm
      type: prediction
      path: /models/lstm-predictor
EOF
```

### 4. Test Integrated Features

Deploy a cluster using both hive-mind and autoscaling:

```yaml
kubectl apply -f - <<EOF
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: integrated-test
spec:
  topology: hierarchical
  queenMode: distributed
  
  hiveMind:
    enabled: true
    syncInterval: 30s
    
  autoscaling:
    enabled: true
    minAgents: 4
    maxAgents: 12
    topologyRatios:
      coordinator: 15
      researcher: 25
      coder: 40
      analyst: 20
      
  neural:
    enabled: true
    acceleration: wasm-simd
    
  monitoring:
    enabled: true
    dashboardEnabled: true
EOF
```

## Test Scenarios

### Scenario 1: Distributed Learning

Test agents sharing learned patterns:

```bash
# Deploy ML training swarm
kubectl apply -f examples/ml-training-swarm.yaml

# Monitor pattern sharing
kubectl logs -l swarm-cluster=ml-training -f | grep -i "pattern"

# Check shared neural weights
kubectl exec -it deployment/ml-training-redis -- redis-cli KEYS "neural:*"
```

### Scenario 2: Fault Tolerance

Test hive-mind recovery:

```bash
# Delete a hive-mind pod
kubectl delete pod manual-hivemind-test-hivemind-1

# Verify automatic recovery
kubectl get pods -l component=hivemind -w

# Check data persistence
kubectl exec -it manual-hivemind-test-hivemind-0 -- curl localhost:8080/status
```

### Scenario 3: Rapid Scaling

Test autoscaler response time:

```bash
# Create sudden load spike
for i in {1..50}; do
  kubectl apply -f - <<EOF
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: load-task-$i
spec:
  task: "CPU intensive task $i"
EOF
done

# Monitor scale-up speed
time kubectl wait --for=condition=Ready swarmagents -l swarm-cluster=manual-autoscale-test --timeout=300s
```

## Troubleshooting

### Common Issues

#### 1. Hive-Mind Not Syncing

```bash
# Check hive-mind logs
kubectl logs -l component=hivemind -f

# Verify network connectivity
kubectl exec -it <hivemind-pod> -- ping <other-hivemind-pod>

# Check sync endpoint
kubectl exec -it <hivemind-pod> -- curl localhost:8080/health
```

#### 2. Autoscaling Not Working

```bash
# Check metrics server
kubectl top nodes
kubectl top pods

# Verify HPA status
kubectl describe hpa

# Check operator logs for scaling decisions
kubectl -n swarm-system logs deployment/swarm-operator | grep -i scale
```

#### 3. Memory Backend Issues

```bash
# Check Redis/Hazelcast pods
kubectl get pods -l component=memory

# Test connectivity
kubectl exec -it <agent-pod> -- nc -zv <memory-service> 6379

# Verify data persistence
kubectl exec -it <memory-pod> -- redis-cli INFO persistence
```

### Debug Commands

```bash
# Get all swarm resources
kubectl get swarmclusters,swarmagents,swarmtasks,swarmmemories -A

# Describe cluster with events
kubectl describe swarmcluster <name>

# Check operator logs
kubectl -n swarm-system logs deployment/swarm-operator --tail=100 -f

# Export debug bundle
kubectl cluster-info dump --output-directory=/tmp/swarm-debug

# Check resource usage
kubectl top pods -l swarm-cluster=<name> --containers
```

## Performance Testing

### Load Testing Script

```bash
#!/bin/bash
# performance-test.sh

CLUSTER_NAME="perf-test"
TASK_COUNT=100
AGENT_COUNT=20

# Create test cluster
kubectl apply -f - <<EOF
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: $CLUSTER_NAME
spec:
  topology: mesh
  hiveMind:
    enabled: true
  autoscaling:
    enabled: true
    minAgents: 5
    maxAgents: $AGENT_COUNT
EOF

# Wait for cluster
sleep 30

# Generate load
echo "Creating $TASK_COUNT tasks..."
START_TIME=$(date +%s)

for i in $(seq 1 $TASK_COUNT); do
  kubectl apply -f - <<EOF &
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: perf-task-$i
spec:
  task: "Performance test task $i"
EOF
done

wait

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo "Created $TASK_COUNT tasks in $DURATION seconds"
echo "Rate: $(($TASK_COUNT / $DURATION)) tasks/second"

# Monitor completion
kubectl wait --for=condition=Completed swarmtask -l swarm-cluster=$CLUSTER_NAME --timeout=600s

# Collect metrics
kubectl top pods -l swarm-cluster=$CLUSTER_NAME
```

### Metrics to Monitor

1. **Hive-Mind Performance**
   - Sync latency: `hivemind_sync_duration_seconds`
   - Consensus time: `hivemind_consensus_duration_seconds`
   - Memory operations/sec: `hivemind_memory_ops_total`

2. **Autoscaling Metrics**
   - Scale decision time: `autoscaler_decision_duration_seconds`
   - Agent startup time: `agent_ready_duration_seconds`
   - Ratio accuracy: `autoscaler_ratio_deviation`

3. **Overall Performance**
   - Task completion rate: `swarm_tasks_completed_per_second`
   - Agent utilization: `agent_utilization_percentage`
   - Memory hit rate: `memory_cache_hit_rate`

## Best Practices

1. **Test Isolation**: Always use separate namespaces for tests
2. **Resource Limits**: Set appropriate limits to prevent cluster exhaustion
3. **Monitoring**: Enable Prometheus metrics for detailed insights
4. **Gradual Scaling**: Test with small clusters before scaling up
5. **Clean Up**: Always clean up test resources after completion

## Conclusion

The enhanced Swarm Operator provides powerful features for collective intelligence and dynamic scaling. Regular testing ensures these features work correctly in your environment. For production deployments, implement comprehensive monitoring and alerting based on the test scenarios provided.