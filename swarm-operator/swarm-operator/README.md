# Swarm Operator

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/cloudflow/swarm-operator)](https://goreportcard.com/report/github.com/cloudflow/swarm-operator)
[![CI Status](https://github.com/cloudflow/swarm-operator/workflows/CI/badge.svg)](https://github.com/cloudflow/swarm-operator/actions)
[![Release](https://img.shields.io/github/release/cloudflow/swarm-operator.svg)](https://github.com/cloudflow/swarm-operator/releases/latest)

A Kubernetes operator for orchestrating intelligent agent swarms based on the Claude Flow architecture. The Swarm Operator enables you to deploy and manage distributed AI agent systems that can collaborate on complex tasks using various topologies and strategies.

## 🌟 Features

- **Multiple Swarm Topologies**: Support for mesh, hierarchical, star, and ring configurations
- **Dynamic Agent Management**: Automatic agent lifecycle management with health checks
- **Task Orchestration**: Sophisticated task distribution with priority queuing and dependencies
- **Auto-scaling**: Horizontal and vertical scaling based on workload
- **Multi-stage Workflows**: Complex task pipelines with stage dependencies
- **Resource Management**: Fine-grained control over CPU and memory allocation
- **Observability**: Built-in Prometheus metrics and distributed tracing
- **High Availability**: Leader election and fault tolerance
- **CLI Tool**: Comprehensive command-line interface for swarm management

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                        │
│                                                              │
│  ┌─────────────────┐         ┌─────────────────┐           │
│  │  Swarm Operator │◄────────┤  Swarm CRD      │           │
│  │                 │         └─────────────────┘           │
│  │  - Controller   │         ┌─────────────────┐           │
│  │  - Webhook      │◄────────┤  Agent CRD      │           │
│  │  - Metrics      │         └─────────────────┘           │
│  └────────┬────────┘         ┌─────────────────┐           │
│           │         ◄────────┤  Task CRD       │           │
│           │                  └─────────────────┘           │
│           ▼                                                 │
│  ┌─────────────────────────────────────────────┐           │
│  │            Swarm Instance                    │           │
│  │  ┌─────────────┐    ┌─────────────┐        │           │
│  │  │ Coordinator │───►│   Agent 1   │        │           │
│  │  │             │    └─────────────┘        │           │
│  │  │  - Router   │    ┌─────────────┐        │           │
│  │  │  - Queue    │───►│   Agent 2   │        │           │
│  │  │  - Monitor  │    └─────────────┘        │           │
│  │  └─────────────┘    ┌─────────────┐        │           │
│  │                 ───►│   Agent N   │        │           │
│  │                     └─────────────┘        │           │
│  └─────────────────────────────────────────────┘           │
└─────────────────────────────────────────────────────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Kubernetes 1.24+
- kubectl configured
- Helm 3.x (optional)

### Installation

```bash
# Clone the repository
git clone https://github.com/cloudflow/swarm-operator.git
cd swarm-operator

# Quick install with all components
make quickstart

# Or manual installation:
# 1. Install CRDs
kubectl apply -f config/crd/bases/

# 2. Deploy operator
kubectl apply -f deploy/install.yaml
```

### Create Your First Swarm

```yaml
# swarm.yaml
apiVersion: swarm.cloudflow.io/v1alpha1
kind: Swarm
metadata:
  name: my-first-swarm
spec:
  topology: mesh
  maxAgents: 5
  strategy: balanced
```

```bash
kubectl apply -f swarm.yaml
kubectl get swarms
kubectl get agents -l swarm=my-first-swarm
```

### Submit a Task

```yaml
# task.yaml
apiVersion: swarm.cloudflow.io/v1alpha1
kind: Task
metadata:
  name: analyze-data
spec:
  swarmRef:
    name: my-first-swarm
  description: "Analyze dataset and generate insights"
  priority: high
  strategy: parallel
```

```bash
kubectl apply -f task.yaml
kubectl get tasks -w
```

## 📚 Documentation

- [Deployment Guide](docs/deployment.md) - Detailed deployment instructions
- [Quick Start Guide](docs/quickstart.md) - Get up and running quickly
- [Troubleshooting](docs/troubleshooting.md) - Common issues and solutions
- [API Reference](docs/api-reference.md) - Complete API documentation
- [Examples](examples/) - Sample configurations and use cases

## 🛠️ CLI Tool (swarmctl)

The swarmctl CLI provides an intuitive interface for managing swarms:

```bash
# Install swarmctl
curl -Lo swarmctl https://github.com/cloudflow/swarm-operator/releases/latest/download/swarmctl
chmod +x swarmctl
sudo mv swarmctl /usr/local/bin/

# Basic usage
swarmctl list swarms
swarmctl create swarm production --topology hierarchical --max-agents 20
swarmctl submit task "Process customer data" --swarm production --priority high
swarmctl status swarm production
```

## 🎯 Use Cases

### Data Processing Pipeline
```yaml
apiVersion: swarm.cloudflow.io/v1alpha1
kind: Swarm
metadata:
  name: data-pipeline
spec:
  topology: hierarchical
  maxAgents: 10
  strategy: specialized
  coordinatorConfig:
    replicas: 2
    capabilities: ["task-distribution", "monitoring"]
```

### Machine Learning Training
```yaml
apiVersion: swarm.cloudflow.io/v1alpha1
kind: Swarm
metadata:
  name: ml-training
spec:
  topology: mesh
  maxAgents: 20
  autoscaling:
    enabled: true
    minAgents: 5
    targetCPUUtilization: 70
```

### Research and Analysis
```yaml
apiVersion: swarm.cloudflow.io/v1alpha1
kind: Swarm
metadata:
  name: research-team
spec:
  topology: star
  maxAgents: 8
  strategy: adaptive
```

## 🔧 Development

### Building from Source

```bash
# Clone repository
git clone https://github.com/cloudflow/swarm-operator.git
cd swarm-operator

# Build operator
make build

# Build Docker image
make docker-build

# Run tests
make test

# Run E2E tests
make test-e2e
```

### Running Locally

```bash
# Install CRDs
make install

# Run operator locally
make run

# In another terminal, create test resources
kubectl apply -f config/samples/
```

## 📊 Monitoring

The operator exposes Prometheus metrics:

```bash
# Port-forward to metrics endpoint
kubectl port-forward -n swarm-operator svc/swarm-operator-metrics 8080:8080

# View metrics
curl http://localhost:8080/metrics
```

Key metrics:
- `swarm_operator_swarm_count` - Number of swarms
- `swarm_operator_agent_count` - Number of agents
- `swarm_operator_task_queue_length` - Task queue size
- `swarm_operator_task_processing_duration` - Task processing time

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built on the Claude Flow architecture
- Inspired by swarm intelligence research
- Powered by Kubernetes operator-sdk

## 📞 Support

- **Documentation**: [docs.swarm-operator.io](https://docs.swarm-operator.io)
- **Issues**: [GitHub Issues](https://github.com/cloudflow/swarm-operator/issues)
- **Discussions**: [GitHub Discussions](https://github.com/cloudflow/swarm-operator/discussions)
- **Slack**: [#swarm-operator](https://cloudflow.slack.com)

---

Built with ❤️ by the CloudFlow team