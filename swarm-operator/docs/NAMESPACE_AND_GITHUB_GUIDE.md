# Namespace Management and GitHub Token Integration Guide

## Overview

The Enhanced Swarm Operator v3.0.0 introduces sophisticated namespace management and GitHub App token generation with repository-scoped access control. This guide covers configuration, deployment, and best practices.

## Table of Contents

1. [Namespace Configuration](#namespace-configuration)
2. [GitHub App Integration](#github-app-integration)
3. [Token Lifecycle Management](#token-lifecycle-management)
4. [Security Best Practices](#security-best-practices)
5. [Examples](#examples)
6. [Troubleshooting](#troubleshooting)

## Namespace Configuration

### Default Namespaces

The operator uses two default namespaces to separate concerns:

- **`claude-flow-swarm`**: General swarm agents and tasks
- **`claude-flow-hivemind`**: Hive-mind components and consensus operations

### Operator Configuration

Configure the operator to watch specific namespaces:

```bash
# Deploy operator with namespace configuration
helm install swarm-operator ./helm/swarm-operator \
  --set namespaces.swarm="claude-flow-swarm" \
  --set namespaces.hivemind="claude-flow-hivemind" \
  --set namespaces.watch="claude-flow-swarm,claude-flow-hivemind"
```

Or using command-line flags:

```bash
/manager \
  --swarm-namespace=claude-flow-swarm \
  --hivemind-namespace=claude-flow-hivemind \
  --watch-namespaces=claude-flow-swarm,claude-flow-hivemind
```

### SwarmCluster Namespace Configuration

Override default namespaces per cluster:

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: my-cluster
spec:
  namespaceConfig:
    swarmNamespace: my-custom-swarm      # Override swarm namespace
    hiveMindNamespace: my-custom-hivemind # Override hive-mind namespace
    allowedNamespaces:                    # Additional allowed namespaces
    - production-tasks
    - staging-tasks
    createNamespaces: true                # Auto-create if missing
```

### Task Namespace Assignment

Tasks can specify their target namespace:

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: my-task
spec:
  namespace: production-tasks  # Explicit namespace
  type: development           # Or use type-based defaults
```

Default namespace assignment by task type:
- `hivemind`, `consensus` → `claude-flow-hivemind`
- All others → `claude-flow-swarm`

## GitHub App Integration

### Prerequisites

1. Create a GitHub App with the following permissions:
   - **Repository permissions**:
     - Contents: Read & Write
     - Pull requests: Read & Write
     - Issues: Read & Write
     - Actions: Read
     - Metadata: Read

2. Install the GitHub App on your organization/repositories

3. Note the App ID and download the private key

### Configuration

#### 1. Store the Private Key

```bash
kubectl create secret generic github-app-key \
  --from-file=private-key=path/to/private-key.pem \
  -n claude-flow-swarm
```

#### 2. Configure SwarmCluster

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: github-enabled-cluster
spec:
  githubApp:
    appID: 123456  # Your GitHub App ID
    privateKeyRef:
      name: github-app-key
      key: private-key
    installationID: 789012  # Optional, auto-discovered if not set
    tokenTTL: "1h"         # Token lifetime (default: 1h)
```

### Repository-Scoped Access

#### Per-Task Repository Access

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: code-analysis-task
spec:
  swarmCluster: github-enabled-cluster
  
  # Repositories this task can access
  repositories:
  - "my-org/frontend-app"
  - "my-org/backend-api"
  - "my-org/shared-libs"
  
  subtasks:
  - name: analyze-code
    type: code-review
    parameters:
      # GITHUB_TOKEN env var will be available
      # with access ONLY to specified repositories
```

#### Per-Agent Repository Access

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmAgent
metadata:
  name: specialized-coder
spec:
  type: coder
  clusterRef: github-enabled-cluster
  
  # Agent-specific repository access
  allowedRepositories:
  - "my-org/microservice-a"
  - "my-org/microservice-b"
```

## Token Lifecycle Management

### Token Generation Flow

1. **Task/Agent Creation**: When a SwarmTask or SwarmAgent is created with repository requirements
2. **Token Request**: Controller requests a scoped installation token from GitHub
3. **Secret Creation**: Token stored in Kubernetes secret with metadata
4. **Environment Injection**: Token injected as `GITHUB_TOKEN` environment variable
5. **Automatic Rotation**: Tokens refreshed before expiration

### Token Secret Structure

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: task-name-github-token
  namespace: claude-flow-swarm
  labels:
    swarm.claudeflow.io/type: github-token
  annotations:
    swarm.claudeflow.io/expires-at: "2024-01-10T15:00:00Z"
    swarm.claudeflow.io/repositories: "org/repo1,org/repo2"
    swarm.claudeflow.io/rotated-at: "2024-01-10T14:00:00Z"
type: Opaque
data:
  token: <base64-encoded-token>
```

### Automatic Rotation

Tokens are automatically rotated:
- 5 minutes before expiration
- On repository list changes
- On manual trigger via annotation

```bash
# Force token rotation
kubectl annotate swarmagent my-agent \
  swarm.claudeflow.io/rotate-token=true
```

## Security Best Practices

### 1. Principle of Least Privilege

- Only grant access to required repositories
- Use separate GitHub Apps for different environments
- Regularly audit repository access

### 2. Private Key Management

```yaml
# Use sealed secrets or external secret operators
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  name: github-app-key
spec:
  encryptedData:
    private-key: <encrypted-key>
```

### 3. Network Policies

Restrict agent network access:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: swarm-agent-github
  namespace: claude-flow-swarm
spec:
  podSelector:
    matchLabels:
      swarm.claudeflow.io/component: agent
  policyTypes:
  - Egress
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 443  # HTTPS to GitHub
```

### 4. RBAC Configuration

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: swarm-operator-github
  namespace: claude-flow-swarm
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["create", "update", "get", "list", "delete"]
  resourceNames: ["*-github-token"]  # Limit to token secrets
```

## Examples

### Example 1: CI/CD Pipeline Task

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: deploy-pipeline
  namespace: claude-flow-swarm
spec:
  swarmCluster: prod-cluster
  type: deployment
  
  repositories:
  - "my-org/app-frontend"
  - "my-org/app-backend"
  - "my-org/infrastructure"
  
  subtasks:
  - name: run-tests
    type: testing
    requiredCapabilities: ["docker", "jest", "pytest"]
    
  - name: build-images
    type: build
    requiredCapabilities: ["docker", "buildkit"]
    
  - name: deploy
    type: deployment
    requiredCapabilities: ["kubectl", "helm"]
```

### Example 2: Multi-Repo Code Analysis

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: security-audit
spec:
  swarmCluster: security-cluster
  priority: high
  
  repositories:
  - "my-org/payment-service"
  - "my-org/auth-service"
  - "my-org/api-gateway"
  
  githubApp:
    tokenTTL: "30m"  # Short-lived token for security
  
  subtasks:
  - name: dependency-scan
    type: security
    parameters:
      tools: ["snyk", "dependabot", "trivy"]
      
  - name: code-analysis
    type: analysis
    parameters:
      tools: ["sonarqube", "semgrep"]
```

### Example 3: Hive-Mind Consensus with GitHub

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: review-cluster
spec:
  topology: mesh
  queenMode: distributed
  strategy: consensus
  
  namespaceConfig:
    hiveMindNamespace: claude-flow-hivemind
    swarmNamespace: claude-flow-swarm
  
  githubApp:
    appID: 123456
    privateKeyRef:
      name: github-app-key
      key: private-key
  
  hiveMind:
    enabled: true
    consensusThreshold: 0.75
---
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: pr-review-consensus
spec:
  swarmCluster: review-cluster
  type: hivemind  # Will run in hivemind namespace
  
  repositories:
  - "my-org/critical-service"
  
  parameters:
    pr_number: "123"
    review_type: "security,performance,architecture"
```

## Troubleshooting

### Issue: Namespace Not Found

```bash
# Check if namespace exists
kubectl get namespace claude-flow-swarm

# Check operator logs
kubectl logs -n swarm-system deployment/swarm-operator | grep namespace

# Verify operator is watching namespace
kubectl get deployment -n swarm-system swarm-operator -o yaml | grep watch-namespaces
```

### Issue: GitHub Token Not Generated

```bash
# Check SwarmTask/Agent status
kubectl describe swarmtask my-task

# Check for token secret
kubectl get secrets -l swarm.claudeflow.io/type=github-token

# Verify GitHub App configuration
kubectl get secret github-app-key -o yaml

# Check operator logs for GitHub errors
kubectl logs -n swarm-system deployment/swarm-operator | grep -i github
```

### Issue: Repository Access Denied

```bash
# Verify token permissions
TOKEN_SECRET=$(kubectl get swarmtask my-task -o jsonpath='{.status.githubTokenSecret}')
kubectl get secret $TOKEN_SECRET -o jsonpath='{.metadata.annotations}'

# Test token manually
TOKEN=$(kubectl get secret $TOKEN_SECRET -o jsonpath='{.data.token}' | base64 -d)
curl -H "Authorization: token $TOKEN" https://api.github.com/installation/repositories
```

### Common Error Messages

| Error | Cause | Solution |
|-------|-------|----------|
| `namespace is not in allowed list` | Namespace not watched by operator | Add to `--watch-namespaces` |
| `failed to generate GitHub token: 404` | GitHub App not installed | Install app on repository |
| `repository not in allowed list` | Token doesn't have repo access | Add to task `repositories` list |
| `token expired` | TTL exceeded | Increase `tokenTTL` or wait for rotation |

## Best Practices Summary

1. **Use Dedicated Namespaces**: Separate swarm and hive-mind workloads
2. **Scope Repository Access**: Only grant access to required repositories
3. **Set Appropriate TTLs**: Balance security with performance (avoid too frequent rotations)
4. **Monitor Token Usage**: Track token creation/rotation metrics
5. **Implement Network Policies**: Restrict egress to GitHub only
6. **Use GitOps**: Store configurations in Git with proper secret management
7. **Regular Audits**: Review repository access and token usage

## Migration Guide

For existing deployments:

1. **Update CRDs**: Apply new CRD definitions with namespace fields
2. **Update Operator**: Deploy new operator version with namespace flags
3. **Create Namespaces**: Ensure target namespaces exist
4. **Update Resources**: Add `namespaceConfig` to SwarmClusters
5. **Configure GitHub**: Add `githubApp` configuration and secrets
6. **Test Migration**: Verify agents spawn in correct namespaces

## Conclusion

The namespace management and GitHub token integration features provide:
- **Security**: Repository-scoped access with automatic rotation
- **Organization**: Logical separation of workload types
- **Flexibility**: Per-cluster and per-task configuration
- **Automation**: No manual token management required

For questions or issues, please refer to the [project repository](https://github.com/claude-flow/swarm-operator).