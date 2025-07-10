# Claude Flow Swarm Operator

A Kubernetes operator for managing Claude Flow swarms, providing automated deployment, scaling, and lifecycle management of AI agent swarms.

## Overview

The Claude Flow Swarm Operator extends Kubernetes with custom resources to manage:
- SwarmCluster: Defines a cluster of AI agents
- SwarmAgent: Individual agent configurations
- SwarmWorkflow: Task orchestration workflows
- SwarmMemory: Persistent memory storage
- SwarmInsight: Performance metrics and insights

## Development Setup

### Prerequisites

- Docker and Docker Compose
- kubectl configured with access to a Kubernetes cluster
- Go 1.21+ (optional, included in dev container)

### Quick Start

1. Start the development environment:
```bash
make dev-up
make dev-shell
```

2. Inside the development container, initialize the project:
```bash
make init
```

3. Create CRDs and controllers:
```bash
# This will be done after CRD designs are complete
# kubebuilder create api --group swarm --version v1alpha1 --kind SwarmCluster
```

### Project Structure

```
swarm-operator/
├── api/            # API definitions for CRDs
├── controllers/    # Reconciliation logic
├── config/         # Kubernetes manifests
│   ├── crd/       # CustomResourceDefinitions
│   ├── rbac/      # RBAC configurations
│   └── manager/   # Controller deployment
├── hack/          # Scripts and utilities
├── docs/          # Documentation
├── cmd/           # Main applications
└── bin/           # Compiled binaries
```

### Development Workflow

1. **Define APIs**: Create or modify CRDs in `api/`
2. **Generate Code**: Run `make generate` to create DeepCopy methods
3. **Implement Controllers**: Add reconciliation logic in `controllers/`
4. **Test Locally**: Use `make run` to test the operator
5. **Build & Deploy**: Use `make docker-build` and `make deploy`

### Available Commands

```bash
# Development
make dev-up        # Start development environment
make dev-shell     # Enter development container
make init          # Initialize kubebuilder project

# Build
make build         # Build operator binary
make docker-build  # Build Docker image
make manifests     # Generate CRD manifests

# Test
make test          # Run unit tests
make kind-create   # Create local test cluster
make kind-load     # Load image into kind

# Deploy
make install       # Install CRDs
make deploy        # Deploy operator
make uninstall     # Remove CRDs
```

## Architecture

The operator follows the standard Kubernetes controller pattern:

1. **Watch** for changes to custom resources
2. **Reconcile** desired state with actual state
3. **Update** status and create/modify resources

## Contributing

1. Create feature branches from `main`
2. Implement changes with tests
3. Run `make verify` to ensure code quality
4. Submit pull requests with clear descriptions

## License

[License information to be added]