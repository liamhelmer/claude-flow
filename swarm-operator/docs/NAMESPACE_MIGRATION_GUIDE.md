# Namespace Migration Guide

## Overview

This guide helps migrate existing Swarm Operator deployments from using the default namespace to the proper namespace structure.

## Namespace Structure

### New Namespace Layout

- **`claude-flow-swarm`**: General swarm agents, tasks, and associated resources
- **`claude-flow-hivemind`**: Hive-mind components and consensus operations
- **`swarm-system`**: Operator deployment and cluster-wide configuration

## Migration Steps

### Step 1: Create New Namespaces

```bash
# Create the required namespaces
kubectl create namespace claude-flow-swarm
kubectl create namespace claude-flow-hivemind
kubectl create namespace swarm-system
```

### Step 2: Migrate Secrets

#### List Existing Secrets

```bash
# List secrets in default namespace
kubectl get secrets -n default | grep -E 'github|api|credentials|token'
```

#### Migrate GitHub App Key

```bash
# Export existing secret
kubectl get secret github-app-key -n default -o yaml > github-app-key.yaml

# Update namespace in the file
sed -i '' 's/namespace: default/namespace: claude-flow-swarm/' github-app-key.yaml

# Apply to new namespace
kubectl apply -f github-app-key.yaml

# Verify
kubectl get secret github-app-key -n claude-flow-swarm

# Delete from default (after verification)
kubectl delete secret github-app-key -n default
```

#### Migrate Other Secrets

```bash
# Function to migrate a secret
migrate_secret() {
  local secret_name=$1
  local target_namespace=$2
  
  kubectl get secret $secret_name -n default -o yaml | \
    sed 's/namespace: default/namespace: '$target_namespace'/' | \
    kubectl apply -f -
}

# Migrate API keys
migrate_secret "openai-api-key" "claude-flow-swarm"
migrate_secret "anthropic-api-key" "claude-flow-swarm"

# Migrate cloud credentials
migrate_secret "aws-credentials" "claude-flow-swarm"
migrate_secret "gcp-credentials" "claude-flow-swarm"
migrate_secret "azure-credentials" "claude-flow-swarm"
```

### Step 3: Migrate SwarmCluster Resources

#### Export Existing SwarmClusters

```bash
# List SwarmClusters
kubectl get swarmclusters -n default

# Export each SwarmCluster
for cluster in $(kubectl get swarmclusters -n default -o name); do
  kubectl get $cluster -n default -o yaml > ${cluster##*/}.yaml
done
```

#### Update SwarmCluster Definitions

Edit each exported YAML file:

1. Change namespace:
   ```yaml
   metadata:
     namespace: claude-flow-swarm  # or claude-flow-hivemind for consensus clusters
   ```

2. Update secret references if using cross-namespace:
   ```yaml
   spec:
     githubApp:
       privateKeyRef:
         name: github-app-key
         # Add namespace if secret is in different namespace
         namespace: swarm-system
   ```

3. Add namespace configuration:
   ```yaml
   spec:
     namespaceConfig:
       swarmNamespace: claude-flow-swarm
       hiveMindNamespace: claude-flow-hivemind
       createNamespaces: true
   ```

#### Apply Updated SwarmClusters

```bash
# Apply to new namespace
kubectl apply -f <cluster-name>.yaml

# Verify
kubectl get swarmclusters -n claude-flow-swarm
```

### Step 4: Migrate SwarmTasks

```bash
# Export tasks
kubectl get swarmtasks -n default -o yaml > swarmtasks-backup.yaml

# Update namespaces
sed -i '' 's/namespace: default/namespace: claude-flow-swarm/g' swarmtasks-backup.yaml

# Apply to new namespace
kubectl apply -f swarmtasks-backup.yaml

# Verify
kubectl get swarmtasks -n claude-flow-swarm
```

### Step 5: Update Operator Deployment

#### Update Operator Configuration

```bash
# Edit operator deployment
kubectl edit deployment swarm-operator -n swarm-system

# Add/update args:
args:
- --leader-elect
- --swarm-namespace=claude-flow-swarm
- --hivemind-namespace=claude-flow-hivemind
- --watch-namespaces=claude-flow-swarm,claude-flow-hivemind
```

#### Update RBAC

```bash
# Grant operator access to new namespaces
kubectl create rolebinding swarm-operator-swarm \
  --clusterrole=swarm-operator \
  --serviceaccount=swarm-system:swarm-operator \
  -n claude-flow-swarm

kubectl create rolebinding swarm-operator-hivemind \
  --clusterrole=swarm-operator \
  --serviceaccount=swarm-system:swarm-operator \
  -n claude-flow-hivemind
```

### Step 6: Cleanup Default Namespace

After verifying all resources are working in new namespaces:

```bash
# Delete SwarmClusters from default
kubectl delete swarmclusters --all -n default

# Delete SwarmTasks from default
kubectl delete swarmtasks --all -n default

# Delete SwarmAgents from default
kubectl delete swarmagents --all -n default

# Delete secrets (after double-checking)
kubectl delete secret github-app-key -n default
# ... other secrets
```

## Verification Steps

### 1. Check Resource Status

```bash
# Check SwarmClusters
kubectl get swarmclusters -n claude-flow-swarm
kubectl get swarmclusters -n claude-flow-hivemind

# Check SwarmTasks
kubectl get swarmtasks -n claude-flow-swarm

# Check SwarmAgents
kubectl get swarmagents -n claude-flow-swarm
```

### 2. Verify Secret Access

```bash
# Test secret access
kubectl auth can-i get secrets \
  --as=system:serviceaccount:claude-flow-swarm:swarm-agent \
  -n claude-flow-swarm
```

### 3. Check Operator Logs

```bash
# Check for errors
kubectl logs -n swarm-system deployment/swarm-operator --tail=100
```

### 4. Test GitHub Token Generation

```bash
# Create a test task
cat <<EOF | kubectl apply -f -
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: github-test
  namespace: claude-flow-swarm
spec:
  swarmCluster: <your-cluster-name>
  repositories:
  - "org/repo"
  subtasks:
  - name: test
    type: git
EOF

# Check if token was created
kubectl get secrets -n claude-flow-swarm | grep github-token
```

## Rollback Plan

If issues occur during migration:

### 1. Keep Backup

```bash
# Backup all resources before migration
kubectl get swarmclusters,swarmtasks,swarmagents,secrets \
  -n default -o yaml > default-namespace-backup.yaml
```

### 2. Restore if Needed

```bash
# Restore from backup
kubectl apply -f default-namespace-backup.yaml

# Revert operator configuration
kubectl edit deployment swarm-operator -n swarm-system
# Remove namespace flags
```

## Common Issues

### Issue: Resources Not Found

**Symptom**: `No resources found in claude-flow-swarm namespace`

**Solution**: Ensure resources were applied to correct namespace:
```bash
kubectl get <resource> -A | grep <name>
```

### Issue: Secret Access Denied

**Symptom**: `secrets "github-app-key" is forbidden`

**Solution**: Check RBAC and secret location:
```bash
# Check if secret exists in namespace
kubectl get secret github-app-key -n claude-flow-swarm

# Check RBAC
kubectl auth can-i get secrets \
  --as=system:serviceaccount:<namespace>:<serviceaccount> \
  -n claude-flow-swarm
```

### Issue: Token Generation Fails

**Symptom**: GitHub tokens not being created

**Solution**: Check cross-namespace access:
```bash
# If secret is in different namespace, ensure RBAC allows access
kubectl create role github-secret-reader \
  --verb=get --resource=secrets \
  --resource-name=github-app-key \
  -n swarm-system

kubectl create rolebinding allow-operator-read-github \
  --role=github-secret-reader \
  --serviceaccount=swarm-system:swarm-operator \
  -n swarm-system
```

## Best Practices Going Forward

1. **Never use default namespace** for production workloads
2. **Use namespace labels** for organization:
   ```bash
   kubectl label namespace claude-flow-swarm environment=production
   kubectl label namespace claude-flow-swarm managed-by=swarm-operator
   ```

3. **Implement Network Policies** for namespace isolation
4. **Use ResourceQuotas** to prevent resource exhaustion:
   ```yaml
   apiVersion: v1
   kind: ResourceQuota
   metadata:
     name: swarm-quota
     namespace: claude-flow-swarm
   spec:
     hard:
       requests.cpu: "100"
       requests.memory: 200Gi
       persistentvolumeclaims: "10"
   ```

5. **Regular Audits**: Check for resources in default namespace:
   ```bash
   kubectl get all -n default | grep -E 'swarm|claude-flow'
   ```

## Summary

Migrating from the default namespace to proper namespaces improves:
- **Security**: Better isolation and RBAC
- **Organization**: Clear separation of concerns
- **Scalability**: Easier to manage multiple environments
- **Compliance**: Meets production best practices

Always test migrations in a non-production environment first!