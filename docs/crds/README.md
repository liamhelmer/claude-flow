# Claude Flow Kubernetes CRDs

This directory contains the Custom Resource Definitions (CRDs) for the Claude Flow Kubernetes operator, which enables running AI swarms natively in Kubernetes.

## Overview

The Claude Flow operator introduces three main CRDs to manage distributed AI agent systems:

1. **Swarm** - Defines a collection of agents with specific topology and coordination settings
2. **Agent** - Represents an individual AI agent with capabilities and resources
3. **Task** - Describes work to be distributed across the swarm

## CRD Files

### Core CRDs
- [`swarm-crd.yaml`](./swarm-crd.yaml) - Swarm resource definition
- [`agent-crd.yaml`](./agent-crd.yaml) - Agent resource definition  
- [`task-crd.yaml`](./task-crd.yaml) - Task resource definition

### Supporting Resources
- [`webhook-config.yaml`](./webhook-config.yaml) - Validating and mutating webhooks

### Examples
- [`examples/swarm-example.yaml`](./examples/swarm-example.yaml) - Various swarm configurations
- [`examples/agent-example.yaml`](./examples/agent-example.yaml) - Different agent types
- [`examples/task-example.yaml`](./examples/task-example.yaml) - Task orchestration patterns

## Swarm CRD

The Swarm resource manages collections of AI agents with different topologies:

### Topologies
- **mesh** - All agents can communicate with each other
- **hierarchical** - Tree-like structure with coordinators
- **ring** - Agents connected in a circular pattern
- **star** - Central coordinator with leaf agents

### Key Features
- Auto-scaling based on metrics (CPU, memory, task queue)
- Multiple execution strategies (parallel, sequential, adaptive)
- Persistent memory with various backends
- WASM/SIMD acceleration support
- Built-in monitoring and observability

### Example Usage
```yaml
kubectl apply -f examples/swarm-example.yaml
kubectl get swarms
kubectl describe swarm app-dev-swarm
```

## Agent CRD

The Agent resource represents individual AI workers with specific capabilities:

### Agent Types
- **coordinator** - Orchestrates tasks and manages other agents
- **researcher** - Gathers information and analyzes data
- **coder** - Implements solutions and writes code
- **analyst** - Performs deep analysis and optimization
- **architect** - Designs systems and architectures
- **tester** - Validates and tests implementations
- **reviewer** - Reviews and ensures quality
- **optimizer** - Performance and efficiency optimization
- **documenter** - Creates documentation
- **monitor** - Observes system health
- **specialist** - Domain-specific expertise

### Key Features
- Cognitive patterns (convergent, divergent, lateral, systems, critical, adaptive)
- Neural network configuration with multiple model types
- Learning and adaptation capabilities
- GPU acceleration support
- Secure communication with encryption
- State persistence and checkpointing

### Example Usage
```yaml
kubectl apply -f examples/agent-example.yaml
kubectl get agents -l swarm=app-dev-swarm
kubectl logs agent/backend-coder-1
```

## Task CRD

The Task resource defines work to be executed by the swarm:

### Key Features
- Complex multi-stage workflows with dependencies
- Subtask orchestration and progress tracking
- Resource constraints and quotas
- Retry policies with backoff
- Multiple output formats and destinations
- Comprehensive monitoring and alerting

### Task Lifecycle
1. **Pending** - Task created but not assigned
2. **Assigning** - Agents being selected
3. **Running** - Task in progress
4. **Completing** - Finalizing results
5. **Completed** - Successfully finished
6. **Failed** - Task failed
7. **Cancelled** - Task cancelled

### Example Usage
```yaml
kubectl apply -f examples/task-example.yaml
kubectl get tasks
kubectl describe task build-microservices-app
```

## Installation

1. Install the CRDs:
```bash
kubectl apply -f swarm-crd.yaml
kubectl apply -f agent-crd.yaml
kubectl apply -f task-crd.yaml
```

2. Install webhook configuration (requires cert-manager):
```bash
kubectl apply -f webhook-config.yaml
```

3. Verify installation:
```bash
kubectl get crds | grep flow.claude.ai
```

## Advanced Features

### Status Subresources
All CRDs support status subresources for proper controller reconciliation:
```bash
kubectl get swarm app-dev-swarm -o jsonpath='{.status}'
```

### Scale Subresource
The Swarm CRD supports kubectl scale:
```bash
kubectl scale swarm app-dev-swarm --replicas=15
```

### Printer Columns
Custom columns provide quick status overview:
```bash
kubectl get swarms
NAME            TOPOLOGY      MAX AGENTS   ACTIVE   PHASE    HEALTH
app-dev-swarm   hierarchical  12           10       Ready    Healthy
```

### Webhook Validation
Webhooks ensure resource validity:
- Topology validation
- Resource limit checks
- Dependency cycle detection
- Security policy enforcement

## Monitoring

### Prometheus Metrics
The operator exposes metrics at `/metrics`:
- `swarm_agent_count` - Number of agents per swarm
- `swarm_task_completion_rate` - Task success rate
- `agent_cpu_usage` - CPU usage by agent
- `agent_memory_usage` - Memory usage by agent
- `task_duration_seconds` - Task execution time

### OpenTelemetry Tracing
Distributed tracing for task execution:
- Agent communication spans
- Task assignment traces
- Neural inference timing
- Memory operation tracking

## Security

### RBAC
Each agent can have its own ServiceAccount and Role:
```yaml
spec:
  security:
    rbac:
      enabled: true
      serviceAccount: agent-sa
      role: agent-role
```

### Encryption
- TLS for agent communication
- Encrypted memory storage
- Secure webhook endpoints

## Best Practices

1. **Resource Limits**: Always set memory and CPU limits for agents
2. **Topology Selection**: Choose topology based on communication patterns
3. **Scaling**: Use auto-scaling for variable workloads
4. **Monitoring**: Enable Prometheus and OpenTelemetry
5. **Security**: Use mTLS for production deployments
6. **Persistence**: Enable checkpointing for long-running tasks

## Troubleshooting

### Common Issues

1. **Agents not starting**
   - Check resource availability
   - Verify RBAC permissions
   - Review webhook logs

2. **Tasks stuck in Pending**
   - Ensure enough agents with required capabilities
   - Check agent health status
   - Review task dependencies

3. **Poor performance**
   - Enable WASM/SIMD acceleration
   - Increase cache size
   - Optimize topology for workload

### Debug Commands
```bash
# View swarm events
kubectl describe swarm <name>

# Check agent logs
kubectl logs agent/<name> -f

# View task progress
kubectl get task <name> -o yaml

# Check webhook logs
kubectl logs -n claude-flow-system deployment/claude-flow-webhook
```

## Future Enhancements

- Federation across clusters
- Multi-region swarms
- Advanced neural architectures
- Quantum computing integration
- Edge deployment support

## Contributing

See the main Claude Flow repository for contribution guidelines.