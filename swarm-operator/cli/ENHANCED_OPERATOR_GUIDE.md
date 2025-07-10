# Enhanced Swarm Operator Guide

## Overview

The Enhanced Swarm Operator extends the Claude Flow Swarm Operator with powerful features for production workloads:

- **Enhanced Executor Image**: Includes kubectl, terraform, google-cloud-sdk with all components, AWS CLI, Azure CLI, and more
- **Multiple Secret Mounting**: Mount any number of Kubernetes secrets into task containers
- **Persistent Volume Claims**: Attach persistent storage for state management and task resumption
- **Task Resumption**: Resume failed tasks from checkpoints
- **Cloud Provider Integration**: Built-in support for GCP, AWS, and Azure credentials
- **Resource Management**: Fine-grained control over CPU and memory allocation

## Quick Start

### 1. Build and Deploy the Enhanced Executor Image

```bash
# Build the enhanced executor image
docker build -f Dockerfile.swarm-executor -t claude-flow/swarm-executor:latest .

# Push to your registry (replace with your registry)
docker tag claude-flow/swarm-executor:latest gcr.io/your-project/swarm-executor:latest
docker push gcr.io/your-project/swarm-executor:latest
```

### 2. Deploy the Enhanced CRDs

```bash
# Apply the enhanced SwarmTask CRD
kubectl apply -f enhanced-swarmtask-crd.yaml
```

### 3. Deploy the Enhanced Operator

```bash
# Create namespace
kubectl create namespace swarm-system

# Deploy the operator with enhanced features
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: swarm-operator
  namespace: swarm-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: swarm-operator
  template:
    metadata:
      labels:
        app: swarm-operator
    spec:
      serviceAccountName: swarm-operator
      containers:
      - name: operator
        image: claude-flow/swarm-operator:enhanced-v0.5.0
        env:
        - name: EXECUTOR_IMAGE
          value: "gcr.io/your-project/swarm-executor:latest"
        - name: ENABLE_PERSISTENCE
          value: "true"
        - name: DEFAULT_STORAGE_CLASS
          value: "standard"
        - name: NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
EOF
```

## Working Examples

### Example 1: Basic Task with Cloud Tools

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: cloud-infrastructure-task
spec:
  task: "Deploy infrastructure across multiple cloud providers"
  priority: high
  strategy: adaptive
  timeout: 1h
  config:
    executorImage: gcr.io/your-project/swarm-executor:latest
    resources:
      requests:
        cpu: "500m"
        memory: "1Gi"
      limits:
        cpu: "2"
        memory: "4Gi"
```

### Example 2: Task with Multiple Secrets

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: multi-secret-task
spec:
  task: "Deploy application with multiple authentication requirements"
  priority: medium
  config:
    additionalSecrets:
    - name: database-credentials
      mountPath: /secrets/db
    - name: api-keys
      mountPath: /secrets/api
    - name: ssl-certificates
      mountPath: /certs
      optional: true
    environment:
    - name: DATABASE_URL
      valueFrom:
        secretKeyRef:
          name: database-credentials
          key: connection-string
```

### Example 3: Task with Persistent Storage

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: stateful-analysis-task
spec:
  task: "Analyze large dataset with checkpoint support"
  priority: high
  resume: true  # Enable resumption on failure
  config:
    persistentVolumes:
    - name: workspace
      mountPath: /workspace
      storageClass: fast-ssd
      size: 100Gi
    - name: cache
      mountPath: /cache
      size: 50Gi
    resources:
      requests:
        cpu: "4"
        memory: "16Gi"
```

### Example 4: Complex Multi-Cloud Deployment

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: multi-cloud-deployment
spec:
  task: "Deploy application stack across GCP, AWS, and Azure"
  priority: critical
  strategy: parallel
  timeout: 2h
  config:
    executorImage: gcr.io/your-project/swarm-executor:latest
    additionalSecrets:
    - name: terraform-backend
      mountPath: /terraform/backend
    - name: deployment-configs
      mountPath: /configs
    persistentVolumes:
    - name: terraform-state
      mountPath: /terraform/state
      storageClass: regional-ssd
      size: 20Gi
    resources:
      requests:
        cpu: "2"
        memory: "8Gi"
      limits:
        cpu: "4"
        memory: "16Gi"
    environment:
    - name: TF_VAR_project_id
      value: "my-gcp-project"
    - name: TF_VAR_aws_region
      value: "us-west-2"
    - name: TF_VAR_azure_location
      value: "westus2"
```

## Setting Up Cloud Provider Credentials

### Google Cloud Platform

```bash
# Create GCP service account key secret
kubectl create secret generic gcp-credentials \
  --from-file=key.json=/path/to/service-account-key.json \
  --namespace=default
```

### AWS

```bash
# Create AWS credentials secret
kubectl create secret generic aws-credentials \
  --from-literal=access-key-id=YOUR_ACCESS_KEY \
  --from-literal=secret-access-key=YOUR_SECRET_KEY \
  --from-literal=region=us-west-2 \
  --namespace=default
```

### Azure

```bash
# Create Azure service principal secret
kubectl create secret generic azure-credentials \
  --from-literal=client-id=YOUR_CLIENT_ID \
  --from-literal=client-secret=YOUR_CLIENT_SECRET \
  --from-literal=tenant-id=YOUR_TENANT_ID \
  --namespace=default
```

## Task Scripts ConfigMap

Create a ConfigMap with your task execution scripts:

```bash
kubectl create configmap swarm-task-scripts --from-file=task.sh --namespace=default
```

Example `task.sh`:
```bash
#!/bin/bash
set -e

echo "Starting enhanced swarm task..."
echo "Task: $TASK_NAME"
echo "Priority: $PRIORITY"

# Check available tools
echo "Checking installed tools..."
command -v kubectl && kubectl version --client
command -v terraform && terraform version
command -v gcloud && gcloud version
command -v aws && aws --version
command -v az && az version

# Your task logic here
case "$TASK_NAME" in
  *"infrastructure"*)
    echo "Running infrastructure deployment..."
    # Terraform commands
    ;;
  *"kubernetes"*)
    echo "Managing Kubernetes resources..."
    # kubectl commands
    ;;
  *"analysis"*)
    echo "Running analysis tasks..."
    # Data processing commands
    ;;
  *)
    echo "Running generic task..."
    ;;
esac

echo "Task completed successfully!"
```

## Task Resumption

To enable task resumption, the executor saves checkpoints to persistent storage:

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: resumable-task
spec:
  task: "Long-running data processing with checkpoints"
  resume: true
  config:
    persistentVolumes:
    - name: checkpoint
      mountPath: /swarm-state
      size: 10Gi
    environment:
    - name: CHECKPOINT_DIR
      value: "/swarm-state/checkpoints"
```

Your task script should implement checkpoint logic:

```bash
#!/bin/bash
CHECKPOINT_FILE="/swarm-state/checkpoint.json"

# Check for existing checkpoint
if [ -f "$CHECKPOINT_FILE" ]; then
  echo "Resuming from checkpoint..."
  LAST_STEP=$(jq -r '.step' "$CHECKPOINT_FILE")
else
  LAST_STEP="0"
fi

# Process with checkpointing
for step in {1..10}; do
  if [ "$step" -le "$LAST_STEP" ]; then
    echo "Skipping completed step $step"
    continue
  fi
  
  echo "Processing step $step..."
  # Your processing logic here
  
  # Save checkpoint
  echo "{\"step\": \"$step\", \"timestamp\": \"$(date -Iseconds)\"}" > "$CHECKPOINT_FILE"
done
```

## Monitoring and Debugging

### View Task Status

```bash
# List all tasks
kubectl get swarmtasks

# Get detailed task information
kubectl describe swarmtask <task-name>

# View task logs
kubectl logs job/swarm-job-<task-name>
```

### Monitor Persistent Volumes

```bash
# List PVCs created by tasks
kubectl get pvc -l managed-by=swarm-operator

# Check PVC usage
kubectl exec -it <pod-name> -- df -h /workspace
```

### Debug Failed Tasks

```bash
# Check task status
kubectl get swarmtask <task-name> -o yaml

# View job details
kubectl describe job swarm-job-<task-name>

# Check pod events
kubectl get events --field-selector involvedObject.name=<pod-name>
```

## Best Practices

1. **Resource Allocation**: Always specify resource requests and limits appropriate for your workload
2. **Secret Management**: Use Kubernetes secrets for sensitive data, never hardcode credentials
3. **Persistent Storage**: Use fast storage classes for I/O intensive workloads
4. **Checkpointing**: Implement checkpointing for long-running tasks to enable resumption
5. **Monitoring**: Set up proper monitoring and alerting for task execution
6. **Image Management**: Keep executor images updated with security patches
7. **Network Policies**: Implement network policies to restrict task communication as needed

## Advanced Configuration

### Custom Executor Image

You can extend the base executor image with additional tools:

```dockerfile
FROM claude-flow/swarm-executor:latest

# Add custom tools
RUN apt-get update && apt-get install -y \
    your-custom-tool \
    && rm -rf /var/lib/apt/lists/*

# Add custom scripts
COPY custom-scripts/ /usr/local/bin/
```

### Task Templates

Create reusable task templates:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: task-templates
data:
  terraform-deploy.yaml: |
    apiVersion: swarm.claudeflow.io/v1alpha1
    kind: SwarmTask
    metadata:
      generateName: terraform-deploy-
    spec:
      task: "Deploy infrastructure with Terraform"
      priority: high
      config:
        persistentVolumes:
        - name: tfstate
          mountPath: /terraform
          size: 10Gi
        resources:
          requests:
            cpu: "1"
            memory: "2Gi"
```

## Troubleshooting

### Common Issues

1. **Task Stuck in Pending**
   - Check if executor image is accessible
   - Verify resource requests can be satisfied
   - Check for pod scheduling issues

2. **Secret Mount Failures**
   - Ensure secrets exist in the correct namespace
   - Verify secret keys match the configuration

3. **PVC Creation Failures**
   - Check if storage class exists
   - Verify sufficient storage quota
   - Ensure PVC access modes are supported

4. **Task Resume Not Working**
   - Verify PVC is properly mounted
   - Check checkpoint file permissions
   - Ensure resume flag is set to true

## Security Considerations

1. **RBAC**: Implement proper RBAC rules for the operator and tasks
2. **Pod Security**: Use Pod Security Standards/Policies
3. **Secret Encryption**: Enable encryption at rest for secrets
4. **Network Isolation**: Use NetworkPolicies to isolate tasks
5. **Image Scanning**: Regularly scan executor images for vulnerabilities
6. **Audit Logging**: Enable audit logging for task operations

## Conclusion

The Enhanced Swarm Operator provides a robust platform for running complex, stateful tasks in Kubernetes with support for multiple cloud providers, persistent storage, and advanced configuration options. Follow the examples and best practices to get the most out of your swarm deployments.