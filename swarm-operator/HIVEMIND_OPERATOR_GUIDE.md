# Enhanced Swarm Operator with Hive-Mind Support

## 🧠 Overview

The Enhanced Swarm Operator v3.0.0 brings advanced capabilities from the upstream claude-flow project to Kubernetes, including:

- **Hive-Mind Collective Intelligence** - Distributed decision-making and knowledge sharing
- **Advanced Autoscaling** - Multi-metric, predictive, and topology-aware scaling
- **Neural Network Integration** - WASM SIMD accelerated ML models
- **Distributed Memory** - High-performance caching with Redis/Hazelcast/etcd
- **Sophisticated Monitoring** - Prometheus metrics, Grafana dashboards, and alerts

## 🚀 Quick Start

### Prerequisites

- Kubernetes 1.26+
- Docker for building images
- kubectl configured
- (Optional) Prometheus Operator for monitoring
- (Optional) GPU nodes for neural acceleration

### Installation

```bash
# Deploy the enhanced operator
./deploy/deploy-hivemind-operator.sh

# Deploy a hive-mind enabled cluster
kubectl apply -f examples/hivemind-cluster.yaml

# Watch the swarm come alive
kubectl get swarmclusters,swarmagents -w
```

## 🏗️ Architecture

### Core Components

1. **SwarmCluster** - The top-level resource managing the entire swarm
2. **SwarmAgent** - Individual agents with specialized capabilities
3. **SwarmTask** - Work units distributed across agents
4. **SwarmMemory** - Persistent shared memory entries

### Hive-Mind System

The hive-mind enables collective intelligence through:

- **Consensus Algorithms** - Raft-based decision making
- **Neural Synchronization** - Shared learning across agents
- **Distributed Memory** - Fast access to collective knowledge
- **Performance Optimization** - Continuous adaptation

## 📖 API Reference

### SwarmCluster

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: research-swarm
spec:
  topology: hierarchical        # mesh, hierarchical, ring, star
  queenMode: distributed        # centralized, distributed
  strategy: adaptive            # parallel, sequential, adaptive, consensus
  consensusThreshold: 0.66      # For consensus decisions
  
  hiveMind:
    enabled: true
    databaseSize: 10Gi
    syncInterval: 30s
    backupEnabled: true
    backupInterval: 1h
    
  autoscaling:
    enabled: true
    minAgents: 3
    maxAgents: 20
    targetUtilization: 80
    topologyRatios:             # Maintain agent type ratios
      coordinator: 10           # 10% coordinators
      researcher: 30            # 30% researchers
      coder: 40                 # 40% coders
      analyst: 20               # 20% analysts
    metrics:
    - type: custom
      name: task_queue_depth
      target: "10"
      
  memory:
    type: redis                 # redis, hazelcast, etcd
    size: 2Gi
    replication: 3
    persistence: true
    cachePolicy: LRU
    compression: true
    
  neural:
    enabled: true
    acceleration: wasm-simd     # cpu, gpu, wasm-simd
    trainingEnabled: true
    models:
    - name: pattern-recognition-v1
      type: pattern-recognition
      path: s3://models/pattern-v1
      
  monitoring:
    enabled: true
    metricsPort: 9090
    dashboardEnabled: true
    alertRules:
    - name: high-task-failure
      expression: |
        rate(swarm_task_failures_total[5m]) > 0.1
      duration: 5m
      severity: critical
```

### SwarmAgent

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmAgent
metadata:
  name: research-lead
spec:
  type: researcher
  clusterRef: research-swarm
  cognitivePattern: divergent   # convergent, divergent, lateral, systems, critical
  priority: 100
  maxConcurrentTasks: 5
  
  capabilities:
  - search
  - analyze
  - summarize
  - cite
  
  specialization:
  - "machine learning"
  - "distributed systems"
  
  neuralModels:
  - pattern-recognition-v1
  - optimization-v1
  
  resources:
    cpu: "2000m"
    memory: "4Gi"
    gpu: "1"                    # For neural acceleration
```

### SwarmMemory

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmMemory
metadata:
  name: research-findings
spec:
  clusterRef: research-swarm
  namespace: research
  type: knowledge
  key: "ml-papers-summary"
  value: |
    {
      "papers": [...],
      "insights": [...],
      "recommendations": [...]
    }
  ttl: 86400                    # 24 hours
  priority: 100
  compression: true
  encryption: true
  sharedWith:                   # Specific agents only
  - research-lead
  - analyst-1
```

## 🎯 Features

### 1. Hive-Mind Collective Intelligence

The hive-mind enables swarms to:
- **Learn collectively** - Agents share experiences and learnings
- **Make consensus decisions** - Distributed voting on complex choices
- **Optimize globally** - System-wide performance improvements
- **Recover from failures** - Collective memory survives agent failures

#### Example: Research Swarm with Hive-Mind

```yaml
# Deploy a research swarm that collectively analyzes papers
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: paper-analyzer
spec:
  topology: mesh                # All agents can communicate
  queenMode: distributed        # No single point of failure
  strategy: consensus           # Decisions require agreement
  consensusThreshold: 0.75      # 75% agreement needed
  
  hiveMind:
    enabled: true
    syncInterval: 10s           # Fast synchronization
    
  # Agents will collectively:
  # 1. Discover papers
  # 2. Distribute analysis tasks
  # 3. Share findings
  # 4. Build consensus on conclusions
```

### 2. Advanced Autoscaling

Multi-dimensional scaling based on:
- **CPU/Memory utilization**
- **Task queue depth**
- **Custom metrics** (latency, throughput, error rate)
- **Predictive scaling** using neural models
- **Cost optimization** with spot instances

#### Example: Autoscaling Configuration

```yaml
autoscaling:
  enabled: true
  minAgents: 5
  maxAgents: 100
  metrics:
  - type: cpu
    target: "80"
  - type: custom
    name: pending_tasks
    target: "50"
  - type: custom
    name: avg_task_duration
    target: "30s"
  topologyRatios:
    coordinator: 5      # Always 5% coordinators
    coder: 60          # 60% coders during scale
    tester: 35         # 35% testers
```

### 3. Neural Network Integration

Integrated ML capabilities:
- **Pattern Recognition** - Identify trends and anomalies
- **Optimization** - Improve task distribution
- **Prediction** - Forecast resource needs
- **WASM SIMD Acceleration** - Fast in-browser inference

#### Example: Neural-Enhanced Agent

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmAgent
metadata:
  name: ml-optimizer
spec:
  type: optimizer
  neuralModels:
  - name: task-predictor
    type: prediction
    path: s3://models/task-predictor-v2
  - name: resource-optimizer
    type: optimization
    path: s3://models/resource-opt-v1
  resources:
    gpu: "1"    # GPU acceleration
```

### 4. Distributed Memory System

High-performance memory with:
- **Multiple backends** - Redis, Hazelcast, etcd
- **Compression** - Reduce memory usage
- **Encryption** - Secure sensitive data
- **TTL support** - Automatic expiration
- **Replication** - High availability

#### Example: Shared Knowledge Base

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmMemory
metadata:
  name: code-patterns
spec:
  clusterRef: dev-swarm
  namespace: patterns
  type: knowledge
  key: "design-patterns/microservices"
  value: |
    {
      "pattern": "saga",
      "implementation": "...",
      "bestPractices": ["..."],
      "examples": ["..."]
    }
  ttl: 0                # Permanent
  compression: true
  priority: 100         # Keep in cache
```

### 5. Monitoring and Observability

Comprehensive monitoring with:
- **Prometheus metrics** - All components instrumented
- **Grafana dashboards** - Pre-built visualizations
- **Custom alerts** - Proactive issue detection
- **Distributed tracing** - Request flow tracking

#### Metrics Available

- `swarm_agents_total` - Total agents by type and status
- `swarm_tasks_completed_total` - Completed tasks counter
- `swarm_task_duration_seconds` - Task execution time
- `swarm_memory_hit_rate` - Cache hit rate
- `swarm_hivemind_sync_duration` - Sync time histogram
- `swarm_consensus_decisions_total` - Consensus outcomes
- `swarm_neural_inference_duration` - ML inference time

## 🔧 Advanced Configuration

### GPU Support

Enable GPU acceleration for neural models:

```yaml
# In agent spec
resources:
  gpu: "1"
  
# Node selector for GPU nodes
nodeSelector:
  accelerator: nvidia-tesla-v100
  
# Or use tolerations
tolerations:
- key: nvidia.com/gpu
  operator: Exists
  effect: NoSchedule
```

### Multi-Cloud Support

The enhanced executor image includes:
- kubectl
- terraform
- gcloud (with ALL alpha components)
- aws-cli v2
- azure-cli
- helm 3
- docker

Example cloud-native task:

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: deploy-infrastructure
spec:
  task: |
    # Authenticate with cloud providers
    gcloud auth activate-service-account --key-file=/secrets/gcp/key.json
    aws configure set region us-west-2
    
    # Deploy with Terraform
    cd /workspace/infrastructure
    terraform init
    terraform plan -out=tfplan
    terraform apply tfplan
    
    # Update Kubernetes
    kubectl apply -f k8s-manifests/
    
    # Use gcloud alpha features
    gcloud alpha compute instances create-with-container ...
```

### Persistent Task State

Enable task resumption:

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: long-running-analysis
spec:
  resume: true              # Enable checkpointing
  persistentVolumes:
  - name: workspace
    size: 100Gi
    mountPath: /workspace
  - name: checkpoint
    size: 10Gi
    mountPath: /checkpoint
  task: |
    # Restore from checkpoint if exists
    if [ -f /checkpoint/state.json ]; then
      echo "Resuming from checkpoint..."
      CURRENT_STEP=$(cat /checkpoint/state.json | jq -r .step)
    else
      CURRENT_STEP=1
    fi
    
    # Long running task with checkpoints
    for step in $(seq $CURRENT_STEP 1000); do
      # Do work...
      
      # Save checkpoint
      echo "{\"step\": $step, \"progress\": $progress}" > /checkpoint/state.json
    done
```

## 📊 Example Use Cases

### 1. Distributed Code Analysis

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: code-analyzer
spec:
  topology: hierarchical
  hiveMind:
    enabled: true
  autoscaling:
    enabled: true
    metrics:
    - type: custom
      name: repositories_pending
      target: "10"
---
# Agents will:
# - Coordinator: Manages repository queue
# - Researchers: Find and clone repositories  
# - Analyzers: Run static analysis
# - Reporters: Generate insights
# All share findings through hive-mind
```

### 2. ML Model Training Pipeline

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: ml-trainer
spec:
  topology: star          # Central coordinator
  neural:
    enabled: true
    trainingEnabled: true
    acceleration: gpu
  persistentVolumes:
  - name: datasets
    size: 1Ti
    mountPath: /data
  - name: models
    size: 100Gi
    mountPath: /models
---
# Agents will:
# - Coordinator: Hyperparameter optimization
# - Trainers: Distributed training on GPU
# - Validators: Model evaluation
# - Optimizers: Model compression
```

### 3. Infrastructure Automation

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: infra-automation
spec:
  topology: ring          # Sequential processing
  memory:
    type: etcd           # For configuration
  monitoring:
    alertRules:
    - name: terraform-failures
      expression: rate(swarm_task_failures{task=~".*terraform.*"}[5m]) > 0
      severity: critical
---
# Agents will:
# - Planners: Terraform plan
# - Validators: Policy checks
# - Executors: Apply changes
# - Monitors: Verify deployment
```

## 🔍 Troubleshooting

### Common Issues

1. **Agents not connecting to hive-mind**
   ```bash
   # Check hive-mind pods
   kubectl get pods -l component=hivemind
   
   # Check agent logs
   kubectl logs <agent-pod> | grep -i hivemind
   ```

2. **Autoscaling not working**
   ```bash
   # Check HPA status
   kubectl describe hpa
   
   # Check metrics server
   kubectl top nodes
   kubectl top pods
   ```

3. **Memory backend issues**
   ```bash
   # Check Redis/Hazelcast pods
   kubectl get pods -l component=memory
   
   # Test connectivity
   kubectl exec -it <agent-pod> -- redis-cli -h <redis-service> ping
   ```

### Debug Commands

```bash
# Get all swarm resources
kubectl get swarmclusters,swarmagents,swarmtasks,swarmmemories

# Describe cluster with events
kubectl describe swarmcluster <name>

# Check operator logs
kubectl -n swarm-system logs deployment/swarm-operator -f

# Export Prometheus metrics
kubectl port-forward -n swarm-system svc/swarm-operator 8080:8080
curl http://localhost:8080/metrics

# Access Grafana dashboards
kubectl port-forward -n monitoring svc/grafana 3000:3000
```

## 🚀 Production Deployment

### High Availability

1. **Operator HA**
   ```yaml
   replicas: 3                    # In operator deployment
   ```

2. **Hive-Mind HA**
   ```yaml
   hiveMind:
     replicas: 5                  # Odd number for consensus
   ```

3. **Memory Backend HA**
   ```yaml
   memory:
     replication: 3               # Triple replication
     persistence: true            # Disk persistence
   ```

### Security

1. **Network Policies**
   ```yaml
   # Restrict agent communication
   apiVersion: networking.k8s.io/v1
   kind: NetworkPolicy
   metadata:
     name: swarm-agent-policy
   spec:
     podSelector:
       matchLabels:
         component: agent
     ingress:
     - from:
       - podSelector:
           matchLabels:
             component: agent
       - podSelector:
           matchLabels:
             component: hivemind
   ```

2. **RBAC**
   ```yaml
   # Limit agent permissions
   apiVersion: rbac.authorization.k8s.io/v1
   kind: Role
   metadata:
     name: swarm-agent
   rules:
   - apiGroups: [""]
     resources: ["configmaps", "secrets"]
     verbs: ["get", "list"]
   ```

3. **Pod Security**
   ```yaml
   securityContext:
     runAsNonRoot: true
     runAsUser: 1000
     fsGroup: 1000
     seccompProfile:
       type: RuntimeDefault
   ```

### Resource Planning

| Component | CPU | Memory | Storage | Notes |
|-----------|-----|---------|---------|-------|
| Operator | 500m | 512Mi | - | HA: 3 replicas |
| Hive-Mind | 1 | 2Gi | 10Gi | Per replica |
| Redis | 1 | 2Gi | 20Gi | Or as per data |
| Agent-Coordinator | 500m | 1Gi | - | Light workload |
| Agent-Coder | 2 | 4Gi | 10Gi | Heavier workload |
| Agent-Analyst | 4 | 8Gi | 20Gi | Memory intensive |
| Agent-Optimizer | 2 | 4Gi | - | GPU recommended |

## 🎓 Learning Resources

- [Claude Flow Documentation](https://github.com/ruvnet/claude-flow)
- [Kubernetes Operators](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [Prometheus Monitoring](https://prometheus.io/docs/introduction/overview/)
- [WASM SIMD](https://webassembly.org/docs/simd/)

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.