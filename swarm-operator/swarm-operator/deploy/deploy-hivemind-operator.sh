#!/bin/bash
# Deploy Enhanced Swarm Operator with Hive-Mind Support

set -e

NAMESPACE=${NAMESPACE:-swarm-system}
REGISTRY=${REGISTRY:-claudeflow}
VERSION=${VERSION:-3.0.0}

echo "🐝 Deploying Enhanced Swarm Operator v${VERSION}"
echo "📍 Namespace: ${NAMESPACE}"
echo "🏗️ Registry: ${REGISTRY}"

# Create namespace if it doesn't exist
echo "Creating namespace..."
kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -

# Apply CRDs
echo "Applying CRDs..."
kubectl apply -f deploy/crds/swarmcluster-crd.yaml
kubectl apply -f deploy/crds/swarmagent-crd.yaml
kubectl apply -f deploy/crds/swarmmemory-crd.yaml
kubectl apply -f deploy/crds/swarmtask-crd.yaml

# Wait for CRDs to be established
echo "Waiting for CRDs to be ready..."
kubectl wait --for condition=established --timeout=60s crd/swarmclusters.swarm.claudeflow.io
kubectl wait --for condition=established --timeout=60s crd/swarmagents.swarm.claudeflow.io
kubectl wait --for condition=established --timeout=60s crd/swarmmemories.swarm.claudeflow.io
kubectl wait --for condition=established --timeout=60s crd/swarmtasks.swarm.claudeflow.io

# Build and push operator image
echo "Building operator image..."
docker build -f build/Dockerfile -t ${REGISTRY}/swarm-operator:${VERSION} .
docker push ${REGISTRY}/swarm-operator:${VERSION}

# Build and push executor image
echo "Building executor image..."
docker build -f build/Dockerfile.swarm-executor -t ${REGISTRY}/swarm-executor:${VERSION} build/
docker push ${REGISTRY}/swarm-executor:${VERSION}

# Build and push hive-mind image
echo "Building hive-mind image..."
docker build -f build/Dockerfile.hivemind -t ${REGISTRY}/hivemind:${VERSION} build/
docker push ${REGISTRY}/hivemind:${VERSION}

# Create default namespaces
echo "🔧 Creating default namespaces..."
kubectl create namespace claude-flow-swarm --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace claude-flow-hivemind --dry-run=client -o yaml | kubectl apply -f -

# Apply operator deployment with namespace configuration
echo "📦 Deploying enhanced Swarm Operator..."
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: swarm-operator
  namespace: ${NAMESPACE}
  labels:
    app: swarm-operator
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
        image: ${REGISTRY}/swarm-operator:${VERSION}
        command:
        - /manager
        args:
        - --leader-elect
        - --swarm-namespace=claude-flow-swarm
        - --hivemind-namespace=claude-flow-hivemind
        - --watch-namespaces=claude-flow-swarm,claude-flow-hivemind
        env:
        - name: WATCH_NAMESPACE
          value: "" # Watch multiple namespaces
        ports:
        - containerPort: 8080
          name: metrics
        - containerPort: 8081
          name: health
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
        livenessProbe:
          httpGet:
            path: /healthz
            port: health
          initialDelaySeconds: 15
          periodSeconds: 20
        readinessProbe:
          httpGet:
            path: /readyz
            port: health
          initialDelaySeconds: 5
          periodSeconds: 10
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: swarm-operator
  namespace: ${NAMESPACE}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: swarm-operator
rules:
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list", "watch", "create"]
- apiGroups: [""]
  resources: ["secrets", "configmaps", "services", "persistentvolumeclaims"]
  verbs: ["*"]
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets"]
  verbs: ["*"]
- apiGroups: ["batch"]
  resources: ["jobs"]
  verbs: ["*"]
- apiGroups: ["autoscaling"]
  resources: ["horizontalpodautoscalers"]
  verbs: ["*"]
- apiGroups: ["swarm.claudeflow.io"]
  resources: ["*"]
  verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: swarm-operator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: swarm-operator
subjects:
- kind: ServiceAccount
  name: swarm-operator
  namespace: ${NAMESPACE}
---
apiVersion: v1
kind: Service
metadata:
  name: swarm-operator-metrics
  namespace: ${NAMESPACE}
  labels:
    app: swarm-operator
spec:
  selector:
    app: swarm-operator
  ports:
  - name: metrics
    port: 8080
    targetPort: metrics
EOF

# Wait for operator to be ready
echo "Waiting for operator to be ready..."
kubectl -n ${NAMESPACE} wait --for=condition=available --timeout=300s deployment/swarm-operator

# Create default agent configuration
echo "Creating default agent configuration..."
kubectl -n ${NAMESPACE} apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: swarm-operator-agent-config
data:
  agent-defaults.yaml: |
    # Default agent configuration
    resources:
      requests:
        cpu: "200m"
        memory: "512Mi"
      limits:
        cpu: "2000m"
        memory: "4Gi"
    
    # Agent type specific overrides
    coordinator:
      resources:
        requests:
          cpu: "500m"
          memory: "1Gi"
      maxConcurrentTasks: 10
      
    analyst:
      resources:
        requests:
          cpu: "1000m"
          memory: "2Gi"
        limits:
          cpu: "4000m"
          memory: "8Gi"
      maxConcurrentTasks: 5
      
    optimizer:
      resources:
        requests:
          cpu: "2000m"
          memory: "4Gi"
          nvidia.com/gpu: "1"
      maxConcurrentTasks: 2
EOF

# Create monitoring resources if prometheus operator is installed
if kubectl api-resources | grep -q servicemonitors.monitoring.coreos.com; then
  echo "Creating ServiceMonitor for Prometheus..."
  kubectl -n ${NAMESPACE} apply -f - <<EOF
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: swarm-operator-metrics
  labels:
    app: swarm-operator
spec:
  selector:
    matchLabels:
      app: swarm-operator
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: swarm-agent-metrics
  labels:
    app: swarm-agent
spec:
  selector:
    matchLabels:
      component: agent
  endpoints:
  - port: metrics
    interval: 15s
    path: /metrics
EOF
fi

# Create sample secrets for cloud providers
echo "Creating sample cloud provider secrets..."
kubectl -n ${NAMESPACE} apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: cloud-credentials-sample
type: Opaque
stringData:
  gcp-key.json: |
    {
      "type": "service_account",
      "project_id": "your-project",
      "private_key_id": "key-id",
      "private_key": "-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----\n"
    }
  aws-credentials: |
    [default]
    aws_access_key_id = YOUR_ACCESS_KEY
    aws_secret_access_key = YOUR_SECRET_KEY
  azure-config: |
    {
      "clientId": "your-client-id",
      "clientSecret": "your-client-secret",
      "subscriptionId": "your-subscription-id",
      "tenantId": "your-tenant-id"
    }
EOF

echo "✅ Enhanced Swarm Operator deployed successfully!"
echo ""
echo "Next steps:"
echo "1. Deploy a SwarmCluster:"
echo "   kubectl apply -f examples/hivemind-cluster.yaml"
echo ""
echo "2. Check operator logs:"
echo "   kubectl -n ${NAMESPACE} logs deployment/swarm-operator -f"
echo ""
echo "3. Monitor swarm status:"
echo "   kubectl get swarmclusters,swarmagents,swarmtasks"
echo ""
echo "4. Enable autoscaling:"
echo "   kubectl apply -f examples/autoscaling-cluster.yaml"