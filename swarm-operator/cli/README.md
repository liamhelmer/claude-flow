# Claude Flow Kubernetes CLI

Enhanced Claude Flow CLI with Kubernetes integration for managing AI swarms in cloud-native environments.

## Features

- 🐝 Deploy and manage AI swarms in Kubernetes
- 📋 Create and monitor swarm tasks
- 🔐 GitHub App authentication support
- 🤖 Operator deployment and management
- 📊 Real-time status monitoring
- 🔄 Seamless integration with Claude Flow

## Installation

```bash
# Install globally
npm install -g ./

# Or use directly
node claude-flow-k8s.js help
```

## Commands

### Swarm Management

```bash
# Deploy a new swarm
claude-flow-k8s swarm-deploy my-swarm hierarchical 8

# Check swarm status
claude-flow-k8s swarm-status
claude-flow-k8s swarm-status my-swarm
```

### Task Management

```bash
# Create a task
claude-flow-k8s task-create analyze-code "Analyze repository for optimization" my-swarm high

# Monitor task execution
claude-flow-k8s task-monitor analyze-code
```

### GitHub App Setup

```bash
# Configure GitHub App credentials
claude-flow-k8s github-app-setup /path/to/private-key.pem 1566739 75143529 Iv23liHKGCAQhZpOdb7a
```

### Operator Management

```bash
# Deploy the swarm operator
claude-flow-k8s operator-deploy latest swarm-system

# View operator logs
claude-flow-k8s operator-logs
claude-flow-k8s operator-logs -f  # Follow logs
```

## Examples

### Deploy a Complete Swarm for Go Development

```bash
# 1. Deploy the operator
claude-flow-k8s operator-deploy

# 2. Setup GitHub App credentials
claude-flow-k8s github-app-setup ~/Downloads/github-app.pem 123456 789012

# 3. Deploy a swarm
claude-flow-k8s swarm-deploy go-dev-swarm hierarchical 8

# 4. Create a task
claude-flow-k8s task-create build-app "Build hello world Go app and push to GitHub" go-dev-swarm high
```

### Monitor Swarm Activity

```bash
# Check all resources
claude-flow-k8s swarm-status

# Monitor specific task
claude-flow-k8s task-monitor build-app

# Watch operator logs
claude-flow-k8s operator-logs -f
```

## Integration with Claude Flow

This CLI extends Claude Flow with Kubernetes capabilities:

```javascript
// Use in Claude Flow workflows
const { k8sCommands } = require('./claude-flow-k8s');

// Deploy swarm programmatically
await k8sCommands['swarm-deploy'].handler(['my-swarm', 'mesh', '5']);

// Create task
await k8sCommands['task-create'].handler(['optimize', 'Optimize performance', 'my-swarm']);
```

## Architecture

```
Claude Flow K8s CLI
├── Swarm Management
│   ├── Deploy SwarmCluster CRDs
│   ├── Configure topology
│   └── Scale agents
├── Task Orchestration
│   ├── Create SwarmTask CRDs
│   ├── Monitor execution
│   └── Retrieve results
├── GitHub Integration
│   ├── GitHub App auth
│   ├── PAT support
│   └── Repository management
└── Operator Control
    ├── Deploy operator
    ├── Monitor health
    └── View logs
```

## Requirements

- Node.js 14+
- kubectl configured with cluster access
- Kubernetes 1.19+
- Swarm Operator deployed (or use `operator-deploy` command)

## Troubleshooting

### Common Issues

1. **CRDs not found**: Run `claude-flow-k8s operator-deploy` first
2. **GitHub auth fails**: Verify private key path and app IDs
3. **Task stuck**: Check operator logs with `claude-flow-k8s operator-logs`

### Debug Mode

```bash
# Enable verbose output
export DEBUG=claude-flow-k8s
claude-flow-k8s swarm-status
```

## Contributing

Contributions welcome! Please submit PRs to the [swarm-operator repository](https://github.com/claude-flow/swarm-operator).

## License

MIT