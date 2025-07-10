# kubectl-swarm

A kubectl plugin for managing AI agent swarms in Kubernetes.

## Features

- 🚀 **Create and manage swarms** with different topologies (mesh, hierarchical, ring, star)
- 📈 **Scale agents** dynamically based on workload
- 📋 **Submit and monitor tasks** with priority and dependency management
- 📊 **Real-time status monitoring** with detailed agent health information
- 📝 **Aggregated log viewing** from all agents with filtering
- 🔍 **Debug tools** for diagnosing swarm issues
- 🎯 **Interactive mode** with user-friendly prompts
- 📄 **Multiple output formats** (table, JSON, YAML)
- 🔧 **Tab completion** for all major shells

## Installation

### Prerequisites

- Kubernetes cluster (1.19+)
- kubectl installed and configured
- Swarm CRDs installed in your cluster (see [swarm-operator](../README.md))

### Install with Krew (Recommended)

```bash
kubectl krew install swarm
```

### Install with Homebrew

```bash
brew tap claude-flow/tap
brew install kubectl-swarm
```

### Install with Script

```bash
curl -sSL https://raw.githubusercontent.com/claude-flow/kubectl-swarm/main/install.sh | bash
```

### Install from Source

```bash
git clone https://github.com/claude-flow/kubectl-swarm.git
cd kubectl-swarm
make install
```

## Usage

### Create a Swarm

```bash
# Create a swarm with default settings
kubectl swarm create my-swarm

# Create with specific topology and agent limits
kubectl swarm create my-swarm --topology hierarchical --max-agents 10 --min-agents 3

# Interactive mode with prompts
kubectl swarm create --interactive
```

### Scale Agents

```bash
# Scale to specific number of agents
kubectl swarm scale my-swarm --replicas 8

# Scale with auto-adjustment enabled
kubectl swarm scale my-swarm --replicas 10 --auto-adjust

# Scale multiple swarms
kubectl swarm scale swarm1 swarm2 --replicas 5
```

### View Status

```bash
# List all swarms
kubectl swarm status

# Get detailed status of a specific swarm
kubectl swarm status my-swarm --detailed

# Watch status updates in real-time
kubectl swarm status --watch

# Output in different formats
kubectl swarm status -o json
kubectl swarm status -o yaml
```

### Submit Tasks

```bash
# Submit a task
kubectl swarm task submit my-swarm --task "Analyze codebase for security vulnerabilities"

# Submit with high priority
kubectl swarm task submit my-swarm --task "Critical bug fix" --priority high

# Submit with dependencies
kubectl swarm task submit my-swarm --task "Deploy app" --depends-on task-123,task-456

# List tasks
kubectl swarm task list my-swarm

# Check task status
kubectl swarm task status task-789

# Cancel a task
kubectl swarm task cancel task-789
```

### View Logs

```bash
# View logs from all agents
kubectl swarm logs my-swarm

# Follow logs in real-time
kubectl swarm logs my-swarm --follow

# Filter by agent type
kubectl swarm logs my-swarm --agent-type researcher,coder

# Filter by task
kubectl swarm logs my-swarm --task task-123

# Show timestamps
kubectl swarm logs my-swarm --timestamps

# Tail specific number of lines
kubectl swarm logs my-swarm --tail 100
```

### Debug Issues

```bash
# Run comprehensive diagnostics
kubectl swarm debug my-swarm

# Debug specific components
kubectl swarm debug my-swarm --component agents

# Verbose output
kubectl swarm debug my-swarm --verbose

# Run diagnostic tests
kubectl swarm debug my-swarm --run-tests

# Export debug report
kubectl swarm debug my-swarm --export debug-report.yaml
```

### Delete Swarms

```bash
# Delete a swarm
kubectl swarm delete my-swarm

# Delete multiple swarms
kubectl swarm delete swarm1 swarm2

# Delete without confirmation
kubectl swarm delete my-swarm --force

# Delete all swarms in namespace
kubectl swarm delete --all
```

## Shell Completion

Enable tab completion for your shell:

### Bash

```bash
source <(kubectl swarm completion bash)

# To persist across sessions:
echo 'source <(kubectl swarm completion bash)' >> ~/.bashrc
```

### Zsh

```bash
source <(kubectl swarm completion zsh)

# To persist across sessions:
echo 'source <(kubectl swarm completion zsh)' >> ~/.zshrc
```

### Fish

```bash
kubectl swarm completion fish | source

# To persist across sessions:
kubectl swarm completion fish > ~/.config/fish/completions/kubectl-swarm.fish
```

### PowerShell

```powershell
kubectl swarm completion powershell | Out-String | Invoke-Expression

# To persist across sessions, add the above line to your PowerShell profile
```

## Configuration

kubectl-swarm respects standard kubectl configuration:

- Uses current context from `~/.kube/config`
- Supports `--namespace` flag
- Supports `--kubeconfig` flag
- Supports `KUBECONFIG` environment variable

## Examples

### Complete Workflow Example

```bash
# 1. Create a swarm for distributed code analysis
kubectl swarm create code-analyzer --topology mesh --max-agents 8

# 2. Check swarm is ready
kubectl swarm status code-analyzer --detailed

# 3. Submit analysis task
kubectl swarm task submit code-analyzer \
  --task "Analyze project for security vulnerabilities, code quality, and performance issues" \
  --priority high

# 4. Monitor task progress
kubectl swarm task list code-analyzer
kubectl swarm logs code-analyzer --follow

# 5. Scale up if needed
kubectl swarm scale code-analyzer --replicas 12

# 6. Debug if issues arise
kubectl swarm debug code-analyzer --verbose

# 7. Clean up when done
kubectl swarm delete code-analyzer
```

### Advanced Task Management

```bash
# Submit complex task with dependencies
TASK1=$(kubectl swarm task submit my-swarm --task "Analyze requirements" -o json | jq -r '.metadata.name')
TASK2=$(kubectl swarm task submit my-swarm --task "Design architecture" --depends-on $TASK1 -o json | jq -r '.metadata.name')
TASK3=$(kubectl swarm task submit my-swarm --task "Generate code" --depends-on $TASK2 -o json | jq -r '.metadata.name')

# Monitor task chain
watch kubectl swarm task list my-swarm
```

## Troubleshooting

### Common Issues

1. **"No swarms found"**
   - Check you're in the correct namespace: `kubectl config get-contexts`
   - List swarms in all namespaces: `kubectl swarm status --all-namespaces`

2. **"Failed to create swarm"**
   - Ensure CRDs are installed: `kubectl get crd swarms.swarm.io`
   - Check RBAC permissions: `kubectl auth can-i create swarms`

3. **"No agents found"**
   - Check swarm status: `kubectl swarm status my-swarm --detailed`
   - View controller logs: `kubectl logs -n swarm-system deployment/swarm-controller`

4. **"Task stuck in pending"**
   - Check agent availability: `kubectl swarm debug my-swarm --component agents`
   - Review task dependencies: `kubectl swarm task status task-id -o yaml`

### Debug Mode

Enable debug logging:

```bash
export KUBECTL_SWARM_DEBUG=true
kubectl swarm status
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.