# Swarm Operator Quick Start Guide

Get up and running with swarm-operator in minutes!

## Prerequisites

- Kubernetes cluster (or Docker Desktop with Kubernetes enabled)
- kubectl installed and configured
- Helm 3.x (optional, but recommended)

## Installation

### Option 1: Quick Install (Local Development)

```bash
# Clone the repository
git clone https://github.com/cloudflow/swarm-operator.git
cd swarm-operator

# Set up local cluster and deploy operator
make quickstart
```

This will:
1. Create a local Kind cluster
2. Build and load the operator image
3. Install CRDs
4. Deploy the operator
5. Run basic validation tests

### Option 2: Manual Installation

```bash
# 1. Install CRDs
kubectl apply -f https://raw.githubusercontent.com/cloudflow/swarm-operator/main/config/crd/bases/

# 2. Deploy operator
kubectl apply -f https://raw.githubusercontent.com/cloudflow/swarm-operator/main/deploy/install.yaml
```

## Creating Your First Swarm

### 1. Basic Mesh Swarm

Create a simple swarm with mesh topology:

```yaml
# my-first-swarm.yaml
apiVersion: swarm.cloudflow.io/v1alpha1
kind: Swarm
metadata:
  name: my-first-swarm
  namespace: default
spec:
  topology: mesh
  maxAgents: 3
  strategy: balanced
```

Apply it:
```bash
kubectl apply -f my-first-swarm.yaml
```

### 2. Check Swarm Status

```bash
# List swarms
kubectl get swarms

# Check detailed status
kubectl describe swarm my-first-swarm

# List created agents
kubectl get agents -l swarm=my-first-swarm
```

## Creating and Running Tasks

### 1. Create a Task

```yaml
# my-task.yaml
apiVersion: swarm.cloudflow.io/v1alpha1
kind: Task
metadata:
  name: analyze-data
  namespace: default
spec:
  swarmRef:
    name: my-first-swarm
  description: "Analyze dataset and generate report"
  priority: high
  strategy: parallel
```

Apply the task:
```bash
kubectl apply -f my-task.yaml
```

### 2. Monitor Task Progress

```bash
# Check task status
kubectl get task analyze-data

# Watch task progress
kubectl get task analyze-data -w

# View task details
kubectl describe task analyze-data
```

## Using the CLI

### Install swarmctl

```bash
# Download latest release
curl -Lo swarmctl https://github.com/cloudflow/swarm-operator/releases/latest/download/swarmctl
chmod +x swarmctl
sudo mv swarmctl /usr/local/bin/

# Or build from source
make build-cli
```

### Basic CLI Commands

```bash
# List all swarms
swarmctl list swarms

# Get swarm details
swarmctl get swarm my-first-swarm

# Create a swarm from CLI
swarmctl create swarm production-swarm \
  --topology hierarchical \
  --max-agents 10 \
  --strategy specialized

# Submit a task
swarmctl create task "Process customer data" \
  --swarm production-swarm \
  --priority critical \
  --strategy adaptive

# Monitor swarm health
swarmctl status swarm production-swarm
```

## Common Patterns

### 1. Hierarchical Swarm for Complex Workflows

```yaml
apiVersion: swarm.cloudflow.io/v1alpha1
kind: Swarm
metadata:
  name: workflow-swarm
spec:
  topology: hierarchical
  maxAgents: 8
  strategy: specialized
  coordinatorConfig:
    replicas: 2
    capabilities:
      - task-distribution
      - monitoring
```

### 2. Auto-scaling Swarm

```yaml
apiVersion: swarm.cloudflow.io/v1alpha1
kind: Swarm
metadata:
  name: autoscale-swarm
spec:
  topology: mesh
  maxAgents: 20
  autoscaling:
    enabled: true
    minAgents: 3
    maxAgents: 20
    targetCPUUtilization: 70
```

### 3. Resource-Constrained Swarm

```yaml
apiVersion: swarm.cloudflow.io/v1alpha1
kind: Swarm
metadata:
  name: lightweight-swarm
spec:
  topology: ring
  maxAgents: 5
  resources:
    limits:
      cpu: "100m"
      memory: "128Mi"
    requests:
      cpu: "50m"
      memory: "64Mi"
```

## Quick Examples

### Data Processing Pipeline

```bash
# Create specialized swarm
cat <<EOF | kubectl apply -f -
apiVersion: swarm.cloudflow.io/v1alpha1
kind: Swarm
metadata:
  name: data-pipeline
spec:
  topology: hierarchical
  maxAgents: 10
  strategy: specialized
EOF

# Submit processing task
cat <<EOF | kubectl apply -f -
apiVersion: swarm.cloudflow.io/v1alpha1
kind: Task
metadata:
  name: process-batch-001
spec:
  swarmRef:
    name: data-pipeline
  description: "Process batch 001 from data lake"
  priority: high
  strategy: parallel
  maxAgents: 5
EOF
```

### Research and Analysis

```bash
# Create research swarm
swarmctl create swarm research-team \
  --topology mesh \
  --max-agents 6 \
  --strategy balanced

# Submit research task
swarmctl create task "Research quantum computing applications" \
  --swarm research-team \
  --priority medium \
  --max-agents 4
```

## Monitoring and Debugging

### View Logs

```bash
# Operator logs
kubectl logs -n swarm-operator -l app.kubernetes.io/name=swarm-operator

# Agent logs
kubectl logs -l swarm=my-first-swarm

# Task execution logs
kubectl logs -l task=analyze-data
```

### Check Metrics

```bash
# Port-forward to metrics endpoint
kubectl port-forward -n swarm-operator svc/swarm-operator-metrics 8080:8080

# View metrics
curl http://localhost:8080/metrics | grep swarm
```

### Common Issues

1. **Swarm Stuck in Pending**
   ```bash
   kubectl describe swarm my-first-swarm
   kubectl get events --field-selector involvedObject.name=my-first-swarm
   ```

2. **Task Not Progressing**
   ```bash
   kubectl describe task analyze-data
   kubectl logs -l task=analyze-data
   ```

3. **Agent Creation Failed**
   ```bash
   kubectl get agents -l swarm=my-first-swarm
   kubectl describe agents -l swarm=my-first-swarm
   ```

## Clean Up

Remove resources when done:

```bash
# Delete specific resources
kubectl delete task analyze-data
kubectl delete swarm my-first-swarm

# Or delete all swarm resources
kubectl delete swarms,agents,tasks --all

# Uninstall operator
kubectl delete -f https://raw.githubusercontent.com/cloudflow/swarm-operator/main/deploy/install.yaml
kubectl delete -f https://raw.githubusercontent.com/cloudflow/swarm-operator/main/config/crd/bases/
```

## Next Steps

- 📖 Read the [full documentation](https://docs.swarm-operator.io)
- 🎯 Explore [advanced examples](../examples/)
- 🚀 Learn about [production deployment](deployment.md)
- 🤝 Join our [community](https://github.com/cloudflow/swarm-operator/discussions)
- 🐛 Report [issues](https://github.com/cloudflow/swarm-operator/issues)

## Getting Help

- **Documentation**: Check our [comprehensive docs](https://docs.swarm-operator.io)
- **Examples**: Browse the [examples directory](../examples/)
- **Community**: Join our [Slack channel](https://cloudflow.slack.com)
- **Issues**: Report bugs on [GitHub](https://github.com/cloudflow/swarm-operator/issues)