# Secrets and Tokens Management Guide

## Overview

This guide provides comprehensive instructions for managing secrets, tokens, and API keys in the Swarm Operator with proper namespace isolation.

## Table of Contents

1. [Namespace Overview](#namespace-overview)
2. [GitHub App Tokens](#github-app-tokens)
3. [API Tokens and Secrets](#api-tokens-and-secrets)
4. [Cloud Provider Credentials](#cloud-provider-credentials)
5. [Best Practices](#best-practices)
6. [Troubleshooting](#troubleshooting)

## Namespace Overview

The Swarm Operator uses namespace isolation for security and organization:

- **`claude-flow-swarm`**: General swarm agents, tasks, and associated secrets
- **`claude-flow-hivemind`**: Hive-mind components and consensus-related secrets
- **`swarm-system`**: Operator deployment and cluster-wide secrets

### Creating Namespaces

```bash
# Create the required namespaces
kubectl create namespace claude-flow-swarm
kubectl create namespace claude-flow-hivemind
kubectl create namespace swarm-system
```

## GitHub App Tokens

### 1. GitHub App Private Key

Store the GitHub App private key in the appropriate namespace:

```bash
# For swarm-related GitHub operations
kubectl create secret generic github-app-key \
  --from-file=private-key=path/to/private-key.pem \
  -n claude-flow-swarm

# For hive-mind consensus on GitHub
kubectl create secret generic github-app-key \
  --from-file=private-key=path/to/private-key.pem \
  -n claude-flow-hivemind
```

### 2. GitHub App Configuration

Configure the SwarmCluster with proper namespace references:

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: github-enabled-cluster
  namespace: claude-flow-swarm  # Deploy in swarm namespace
spec:
  githubApp:
    appID: 123456
    privateKeyRef:
      name: github-app-key
      key: private-key
      # namespace field is optional - defaults to the SwarmCluster's namespace
    tokenTTL: "1h"
```

### 3. Cross-Namespace Access

If the secret is in a different namespace:

```yaml
spec:
  githubApp:
    privateKeyRef:
      name: github-app-key
      key: private-key
      namespace: swarm-system  # Explicitly reference another namespace
```

### 4. Generated Token Secrets

The operator creates GitHub tokens in the same namespace as the resource:

```yaml
# Automatically created by the operator
apiVersion: v1
kind: Secret
metadata:
  name: task-name-github-token
  namespace: claude-flow-swarm  # Same as the SwarmTask
  labels:
    swarm.claudeflow.io/type: github-token
    swarm.claudeflow.io/task: task-name
  annotations:
    swarm.claudeflow.io/expires-at: "2025-01-10T15:00:00Z"
    swarm.claudeflow.io/repositories: "org/repo1,org/repo2"
type: Opaque
data:
  token: <base64-encoded-token>
```

## API Tokens and Secrets

### 1. External API Keys

Store API keys in the appropriate namespace based on usage:

```bash
# OpenAI API key for swarm agents
kubectl create secret generic openai-api-key \
  --from-literal=api-key=sk-... \
  -n claude-flow-swarm

# Anthropic API key for hive-mind consensus
kubectl create secret generic anthropic-api-key \
  --from-literal=api-key=sk-ant-... \
  -n claude-flow-hivemind

# Shared API keys (accessible by operator)
kubectl create secret generic shared-api-keys \
  --from-literal=slack-token=xoxb-... \
  --from-literal=discord-token=... \
  -n swarm-system
```

### 2. Database Credentials

For SQLite memory stores and other databases:

```bash
# PostgreSQL credentials for memory store
kubectl create secret generic postgres-credentials \
  --from-literal=username=swarmuser \
  --from-literal=password=secure-password \
  --from-literal=database=swarmdb \
  -n claude-flow-swarm

# Redis credentials for distributed memory
kubectl create secret generic redis-credentials \
  --from-literal=password=redis-password \
  -n claude-flow-swarm
```

### 3. Using Secrets in SwarmTasks

Reference secrets in SwarmTask specifications:

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: api-integration-task
  namespace: claude-flow-swarm
spec:
  swarmCluster: my-cluster
  
  # Environment variables from secrets
  env:
  - name: OPENAI_API_KEY
    valueFrom:
      secretKeyRef:
        name: openai-api-key
        key: api-key
        # Defaults to task's namespace (claude-flow-swarm)
  
  - name: SLACK_TOKEN
    valueFrom:
      secretKeyRef:
        name: shared-api-keys
        key: slack-token
        namespace: swarm-system  # Explicit namespace
```

## Cloud Provider Credentials

### 1. AWS Credentials

```bash
# Create AWS credentials secret
kubectl create secret generic aws-credentials \
  --from-literal=aws-access-key-id=AKIAIOSFODNN7EXAMPLE \
  --from-literal=aws-secret-access-key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY \
  --from-literal=aws-region=us-west-2 \
  -n claude-flow-swarm

# Or from credentials file
kubectl create secret generic aws-credentials \
  --from-file=credentials=/path/to/.aws/credentials \
  -n claude-flow-swarm
```

### 2. Google Cloud Platform

```bash
# Create GCP service account key secret
kubectl create secret generic gcp-credentials \
  --from-file=key.json=/path/to/service-account-key.json \
  -n claude-flow-swarm
```

### 3. Azure Credentials

```bash
# Create Azure service principal secret
kubectl create secret generic azure-credentials \
  --from-literal=azure-client-id=00000000-0000-0000-0000-000000000000 \
  --from-literal=azure-client-secret=secure-secret \
  --from-literal=azure-tenant-id=00000000-0000-0000-0000-000000000000 \
  --from-literal=azure-subscription-id=00000000-0000-0000-0000-000000000000 \
  -n claude-flow-swarm
```

### 4. Multi-Cloud SwarmCluster Configuration

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: multi-cloud-cluster
  namespace: claude-flow-swarm
spec:
  cloudProviders:
    aws:
      credentialsRef:
        name: aws-credentials
        # Defaults to cluster's namespace
    gcp:
      credentialsRef:
        name: gcp-credentials
        key: key.json
    azure:
      credentialsRef:
        name: azure-credentials
        namespace: swarm-system  # Can reference other namespaces
```

## Best Practices

### 1. Namespace Isolation

- **Never use the default namespace** for production secrets
- Keep secrets in the same namespace as the resources that use them
- Use explicit namespace references when cross-namespace access is required

### 2. Secret Naming Conventions

Follow consistent naming patterns:

```bash
# Pattern: <provider>-<type>-<environment>
github-app-key-prod
openai-api-key-dev
aws-credentials-staging
postgres-credentials-prod
```

### 3. RBAC for Secret Access

Configure proper RBAC for namespace access:

```yaml
# Allow swarm agents to read secrets in their namespace
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: swarm-agent-secrets
  namespace: claude-flow-swarm
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list"]
  resourceNames: 
  - "github-app-key"
  - "openai-api-key"
  - "aws-credentials"
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: swarm-agent-secrets
  namespace: claude-flow-swarm
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: swarm-agent-secrets
subjects:
- kind: ServiceAccount
  name: swarm-agent
  namespace: claude-flow-swarm
```

### 4. Secret Rotation

Implement regular rotation for sensitive credentials:

```bash
# Rotate GitHub App token
kubectl delete secret github-app-key -n claude-flow-swarm
kubectl create secret generic github-app-key \
  --from-file=private-key=path/to/new-private-key.pem \
  -n claude-flow-swarm

# Force token regeneration
kubectl annotate swarmtask my-task \
  swarm.claudeflow.io/rotate-token=true \
  -n claude-flow-swarm
```

### 5. Encryption at Rest

Ensure Kubernetes secrets encryption is enabled:

```yaml
# encryption-config.yaml
apiVersion: apiserver.config.k8s.io/v1
kind: EncryptionConfiguration
resources:
  - resources:
    - secrets
    providers:
    - aescbc:
        keys:
        - name: key1
          secret: <base64-encoded-secret>
    - identity: {}
```

## Troubleshooting

### Issue: Secret Not Found

```bash
# Check if secret exists in the correct namespace
kubectl get secrets -n claude-flow-swarm
kubectl get secrets -n claude-flow-hivemind
kubectl get secrets -n swarm-system

# Verify secret contents (be careful with sensitive data)
kubectl get secret github-app-key -n claude-flow-swarm -o yaml
```

### Issue: Cross-Namespace Access Denied

```bash
# Check RBAC permissions
kubectl auth can-i get secrets -n swarm-system --as=system:serviceaccount:claude-flow-swarm:swarm-operator

# Grant cross-namespace access if needed
kubectl create rolebinding swarm-operator-secrets \
  --clusterrole=view \
  --serviceaccount=claude-flow-swarm:swarm-operator \
  -n swarm-system
```

### Issue: Token Expired

```bash
# Check token expiration
kubectl get secret task-name-github-token -n claude-flow-swarm -o jsonpath='{.metadata.annotations.swarm\.claudeflow\.io/expires-at}'

# Force regeneration
kubectl delete secret task-name-github-token -n claude-flow-swarm
# The controller will recreate it
```

### Common Namespace Mistakes

1. **Wrong**: Creating secrets in default namespace
   ```bash
   kubectl create secret generic github-app-key --from-file=private-key=key.pem
   ```

2. **Correct**: Specify the appropriate namespace
   ```bash
   kubectl create secret generic github-app-key --from-file=private-key=key.pem -n claude-flow-swarm
   ```

3. **Wrong**: Hardcoding namespace in cross-namespace references
   ```yaml
   privateKeyRef:
     name: github-app-key
     namespace: default  # Never use default
   ```

4. **Correct**: Use proper namespace references
   ```yaml
   privateKeyRef:
     name: github-app-key
     namespace: swarm-system  # Or omit for same namespace
   ```

## Migration from Default Namespace

If you have existing secrets in the default namespace:

```bash
# List secrets in default namespace
kubectl get secrets -n default

# Copy secret to proper namespace
kubectl get secret github-app-key -n default -o yaml | \
  sed 's/namespace: default/namespace: claude-flow-swarm/' | \
  kubectl apply -f -

# Verify and delete from default
kubectl get secret github-app-key -n claude-flow-swarm
kubectl delete secret github-app-key -n default
```

## Summary

Proper namespace management for secrets and tokens is crucial for security and organization. Always:

1. Use `claude-flow-swarm` for swarm-related secrets
2. Use `claude-flow-hivemind` for hive-mind consensus secrets
3. Use `swarm-system` for operator and cluster-wide secrets
4. Never use the `default` namespace for production secrets
5. Implement proper RBAC for secret access
6. Regularly rotate sensitive credentials
7. Monitor token expiration and automate rotation