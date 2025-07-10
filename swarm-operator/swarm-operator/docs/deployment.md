# Swarm Operator Deployment Guide

This guide provides detailed instructions for deploying the swarm-operator to various Kubernetes environments.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Local Development Deployment](#local-development-deployment)
3. [Production Deployment](#production-deployment)
4. [Configuration Options](#configuration-options)
5. [Verification](#verification)
6. [Troubleshooting](#troubleshooting)

## Prerequisites

Before deploying swarm-operator, ensure you have:

- Kubernetes cluster (v1.24+)
- kubectl configured to access your cluster
- Helm 3.x installed
- Docker (for building images)
- Make (for using Makefile targets)

## Local Development Deployment

### 1. Set Up Local Cluster

We support both Kind and Minikube for local development:

```bash
# Using Kind (recommended)
./scripts/local-setup.sh

# Using Minikube
USE_KIND=false ./scripts/local-setup.sh
```

This script will:
- Install necessary tools if missing
- Create a multi-node cluster
- Install metrics-server
- Set up NGINX ingress controller
- Create swarm-operator namespace

### 2. Build Operator Image

```bash
# Build the operator image
make docker-build IMG=swarm-operator:latest

# For Kind clusters
make docker-load IMG=swarm-operator:latest

# For Minikube clusters
minikube image load swarm-operator:latest
```

### 3. Install CRDs

```bash
# Install Custom Resource Definitions
kubectl apply -f config/crd/bases/
```

### 4. Deploy Operator

```bash
# Deploy using the convenience script
./scripts/deploy-operator.sh

# Or manually with Helm
helm install swarm-operator deploy/helm/swarm-operator \
  --namespace swarm-operator \
  --create-namespace \
  --set image.repository=swarm-operator \
  --set image.tag=latest \
  --set image.pullPolicy=IfNotPresent
```

## Production Deployment

### 1. Build and Push Image

```bash
# Build production image
make docker-build IMG=your-registry.io/swarm-operator:v1.0.0

# Push to registry
make docker-push IMG=your-registry.io/swarm-operator:v1.0.0
```

### 2. Create Values File

Create a `values-production.yaml` file:

```yaml
# Production values for swarm-operator
replicaCount: 3

image:
  repository: your-registry.io/swarm-operator
  tag: v1.0.0
  pullPolicy: IfNotPresent

imagePullSecrets:
  - name: registry-secret

resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80

monitoring:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 30s
    namespace: monitoring

nodeSelector:
  node-role.kubernetes.io/worker: "true"

tolerations:
  - key: "workload"
    operator: "Equal"
    value: "system"
    effect: "NoSchedule"

affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        labelSelector:
          matchExpressions:
          - key: app.kubernetes.io/name
            operator: In
            values:
            - swarm-operator
        topologyKey: kubernetes.io/hostname
```

### 3. Deploy to Production

```bash
# Create namespace
kubectl create namespace swarm-operator

# Create image pull secret if needed
kubectl create secret docker-registry registry-secret \
  --docker-server=your-registry.io \
  --docker-username=your-username \
  --docker-password=your-password \
  --namespace swarm-operator

# Install CRDs
kubectl apply -f config/crd/bases/

# Deploy with Helm
helm upgrade --install swarm-operator deploy/helm/swarm-operator \
  --namespace swarm-operator \
  --values values-production.yaml \
  --wait
```

## Configuration Options

### Operator Configuration

Key configuration options via Helm values:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of operator replicas | `1` |
| `image.repository` | Container image repository | `swarm-operator` |
| `image.tag` | Container image tag | `latest` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `autoscaling.enabled` | Enable HPA | `false` |
| `monitoring.enabled` | Enable Prometheus metrics | `true` |
| `leaderElection.enabled` | Enable leader election | `true` |

### Environment Variables

The operator supports these environment variables:

```yaml
env:
  - name: MAX_CONCURRENT_RECONCILES
    value: "10"
  - name: RECONCILE_INTERVAL
    value: "30s"
  - name: METRICS_ADDR
    value: ":8080"
  - name: PROBE_ADDR
    value: ":8081"
  - name: ENABLE_WEBHOOKS
    value: "true"
```

## Verification

### 1. Check Operator Status

```bash
# Check deployment
kubectl get deployment -n swarm-operator

# Check pods
kubectl get pods -n swarm-operator

# Check logs
kubectl logs -n swarm-operator -l app.kubernetes.io/name=swarm-operator
```

### 2. Verify CRDs

```bash
# List CRDs
kubectl get crds | grep swarm.cloudflow.io

# Describe CRDs
kubectl describe crd swarms.swarm.cloudflow.io
```

### 3. Create Test Swarm

```bash
# Apply test swarm
kubectl apply -f examples/basic-swarm.yaml

# Check swarm status
kubectl get swarms -A

# Describe swarm
kubectl describe swarm test-swarm
```

### 4. Run E2E Tests

```bash
# Run comprehensive tests
./scripts/run-tests.sh
```

## Troubleshooting

### Common Issues

#### 1. Operator Pod CrashLoopBackOff

Check logs:
```bash
kubectl logs -n swarm-operator -l app.kubernetes.io/name=swarm-operator --previous
```

Common causes:
- Missing CRDs
- Insufficient permissions (check RBAC)
- Invalid configuration

#### 2. Swarms Stuck in Pending

Check operator logs and events:
```bash
kubectl describe swarm <swarm-name>
kubectl get events --field-selector involvedObject.name=<swarm-name>
```

#### 3. Webhook Certificate Issues

If using webhooks, ensure cert-manager is installed:
```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
```

### Debug Mode

Enable debug logging:
```bash
helm upgrade swarm-operator deploy/helm/swarm-operator \
  --set env[0].name=LOG_LEVEL \
  --set env[0].value=debug
```

### Metrics and Monitoring

Access metrics:
```bash
# Port-forward to operator
kubectl port-forward -n swarm-operator svc/swarm-operator-metrics 8080:8080

# View metrics
curl http://localhost:8080/metrics
```

## Advanced Deployment Scenarios

### Multi-Region Deployment

For multi-region deployments, use federation:

```yaml
# federation-values.yaml
federation:
  enabled: true
  regions:
    - name: us-east
      endpoint: https://us-east.k8s.example.com
    - name: eu-west
      endpoint: https://eu-west.k8s.example.com
```

### High Availability Setup

For HA deployments:

1. Use at least 3 replicas
2. Enable pod disruption budgets
3. Configure anti-affinity rules
4. Use leader election
5. Set up cross-AZ node groups

### GitOps Integration

For ArgoCD or Flux:

```yaml
# argocd-app.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: swarm-operator
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/your-org/swarm-operator
    targetRevision: main
    path: deploy/helm/swarm-operator
    helm:
      valueFiles:
      - values-production.yaml
  destination:
    server: https://kubernetes.default.svc
    namespace: swarm-operator
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

## Security Considerations

1. **RBAC**: Review and restrict permissions in `deploy/helm/swarm-operator/templates/rbac.yaml`
2. **Network Policies**: Implement network policies to restrict traffic
3. **Pod Security**: Use Pod Security Standards
4. **Secret Management**: Use external secret operators for sensitive data
5. **Image Scanning**: Scan images for vulnerabilities before deployment

## Backup and Recovery

1. **CRD Backup**: Regularly backup CRD instances
2. **Operator State**: Use persistent volumes for stateful components
3. **Disaster Recovery**: Document recovery procedures

## Next Steps

- Review [Quickstart Guide](quickstart.md) for basic usage
- Check [Examples](../examples/) for various configurations
- Read [Troubleshooting Guide](troubleshooting.md) for common issues
- Join our community for support and discussions