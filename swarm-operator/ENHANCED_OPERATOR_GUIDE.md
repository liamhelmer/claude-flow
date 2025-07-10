# Enhanced Swarm Operator Guide

## 🚀 Overview

The Enhanced Swarm Operator v2.0.0 brings powerful cloud-native capabilities to AI swarm orchestration in Kubernetes, including:

- **Multi-Cloud Support**: Pre-installed kubectl, terraform, gcloud (with alpha), AWS CLI, and Azure CLI
- **Additional Secrets Management**: Mount multiple Kubernetes secrets with custom paths
- **Persistent Storage**: PVC support for stateful tasks with checkpoint/resume capability
- **Resource Management**: Fine-grained CPU, memory, and GPU controls
- **Custom Executors**: Use your own Docker images with pre-installed tools
- **Task Resumption**: Resume failed tasks from checkpoints

## 📋 Table of Contents

1. [Quick Start](#quick-start)
2. [Enhanced Features](#enhanced-features)
3. [Working Examples](#working-examples)
4. [Cloud Provider Setup](#cloud-provider-setup)
5. [Persistent Storage](#persistent-storage)
6. [Task Resumption](#task-resumption)
7. [Advanced Configuration](#advanced-configuration)
8. [Troubleshooting](#troubleshooting)

## Quick Start

### 1. Deploy the Enhanced Operator

```bash
# Create namespace
kubectl create namespace swarm-system

# Apply enhanced CRDs
kubectl apply -f deploy/crds/enhanced-swarmtask-crd.yaml
kubectl apply -f deploy/crds/swarmcluster-crd.yaml
kubectl apply -f deploy/crds/agent-crd.yaml

# Create RBAC
kubectl apply -f deploy/enhanced-rbac.yaml

# Deploy operator
kubectl apply -f deploy/enhanced-operator-deployment.yaml

# Verify deployment
kubectl get pods -n swarm-system
```

### 2. Build and Push Enhanced Executor Image

```bash
# Build the enhanced executor image
docker build -f build/Dockerfile.swarm-executor -t claudeflow/swarm-executor:2.0.0 .

# Push to registry (adjust for your registry)
docker push claudeflow/swarm-executor:2.0.0
```

### 3. Create Your First Enhanced Task

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: terraform-gcp-setup
  namespace: claude-flow-swarm
spec:
  task: |
    echo "🚀 Setting up GCP infrastructure with Terraform"
    
    # Initialize Terraform
    cd /workspace
    cat > main.tf << 'EOF'
    provider "google" {
      project = var.project_id
      region  = "us-central1"
    }
    
    resource "google_storage_bucket" "swarm_data" {
      name     = "${var.project_id}-swarm-data"
      location = "US"
    }
    EOF
    
    terraform init
    terraform plan
    terraform apply -auto-approve
    
    # Save state to persistent volume
    cp terraform.tfstate /swarm-state/
    /scripts/checkpoint.sh save "terraform-complete" '{"bucket_created": true}'
  executorImage: claudeflow/swarm-executor:2.0.0
  additionalSecrets:
  - name: gcp-credentials
    mountPath: /credentials/gcp
  persistentVolumes:
  - name: workspace
    mountPath: /workspace
    size: 10Gi
  - name: terraform-state
    mountPath: /swarm-state
    size: 5Gi
  resources:
    requests:
      cpu: "1"
      memory: "2Gi"
    limits:
      cpu: "2"
      memory: "4Gi"
```

## Enhanced Features

### 1. Pre-installed Tools

The enhanced executor image includes:

```bash
# Cloud CLIs
- kubectl (latest stable)
- terraform (v1.7.0)
- gcloud with ALL alpha/beta components
- aws cli v2
- azure cli

# Container Tools
- docker cli
- skaffold
- helm 3
- kustomize

# Development Tools
- go 1.22
- python 3 with pip
- node.js with npm/yarn
- git, jq, yq, curl, wget

# Kubernetes Tools
- k9s
- stern
- claude-flow cli

# Python Libraries
- PyJWT (GitHub App auth)
- google-cloud-* (GCP SDKs)
- boto3 (AWS SDK)
- azure-* (Azure SDKs)
- kubernetes client
```

### 2. Additional Secrets Mounting

Mount multiple secrets with custom paths:

```yaml
additionalSecrets:
- name: database-credentials
  mountPath: /secrets/db
- name: api-keys
  mountPath: /secrets/api
  items:
  - key: openai_key
    path: openai.txt
  - key: anthropic_key
    path: anthropic.txt
- name: ssh-keys
  mountPath: /root/.ssh
```

### 3. Persistent Volumes

Configure persistent storage for stateful tasks:

```yaml
persistentVolumes:
- name: workspace
  mountPath: /workspace
  storageClass: fast-ssd
  size: 100Gi
  accessMode: ReadWriteOnce
- name: cache
  mountPath: /cache
  size: 50Gi
- name: models
  mountPath: /models
  storageClass: standard
  size: 200Gi
  accessMode: ReadOnlyMany
```

## Working Examples

### Example 1: Multi-Cloud Infrastructure Deployment

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: multi-cloud-setup
  namespace: claude-flow-swarm
spec:
  task: |
    #!/bin/bash
    set -e
    
    echo "🌐 Deploying multi-cloud infrastructure"
    
    # Setup GCP
    echo "☁️ Configuring GCP..."
    gcloud auth activate-service-account --key-file=/credentials/gcp/key.json
    gcloud config set project $GCP_PROJECT_ID
    
    # Create GKE cluster
    gcloud container clusters create swarm-cluster \
      --zone us-central1-a \
      --num-nodes 3 \
      --machine-type n2-standard-4
    
    # Setup AWS
    echo "☁️ Configuring AWS..."
    aws configure set region us-east-1
    
    # Create EKS cluster using eksctl
    cat > cluster.yaml << EOF
    apiVersion: eksctl.io/v1alpha5
    kind: ClusterConfig
    metadata:
      name: swarm-cluster-aws
      region: us-east-1
    nodeGroups:
    - name: workers
      instanceType: t3.large
      desiredCapacity: 3
    EOF
    
    eksctl create cluster -f cluster.yaml
    
    # Save kubeconfigs
    mkdir -p /swarm-state/kubeconfigs
    gcloud container clusters get-credentials swarm-cluster --zone us-central1-a
    cp ~/.kube/config /swarm-state/kubeconfigs/gcp-cluster.yaml
    
    aws eks update-kubeconfig --name swarm-cluster-aws --region us-east-1
    cp ~/.kube/config /swarm-state/kubeconfigs/aws-cluster.yaml
    
    /scripts/checkpoint.sh save "clusters-created" '{"gcp": true, "aws": true}'
  executorImage: claudeflow/swarm-executor:2.0.0
  additionalSecrets:
  - name: gcp-credentials
    mountPath: /credentials/gcp
  - name: aws-credentials
    mountPath: /credentials/aws
  persistentVolumes:
  - name: state
    mountPath: /swarm-state
    size: 20Gi
  environment:
    GCP_PROJECT_ID: "my-project-123"
  resources:
    requests:
      cpu: "2"
      memory: "4Gi"
    limits:
      cpu: "4"
      memory: "8Gi"
  timeout: "2h"
```

### Example 2: Terraform Infrastructure with State Management

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: terraform-full-stack
  namespace: claude-flow-swarm
spec:
  task: |
    #!/bin/bash
    set -e
    
    cd /workspace
    
    # Create comprehensive Terraform configuration
    cat > variables.tf << 'EOF'
    variable "project_id" {
      description = "GCP Project ID"
      type        = string
    }
    
    variable "region" {
      description = "Default region"
      default     = "us-central1"
    }
    EOF
    
    cat > main.tf << 'EOF'
    terraform {
      required_providers {
        google = {
          source  = "hashicorp/google"
          version = "~> 5.0"
        }
        kubernetes = {
          source  = "hashicorp/kubernetes"
          version = "~> 2.0"
        }
      }
      
      backend "gcs" {
        bucket = "${var.project_id}-terraform-state"
        prefix = "swarm/state"
      }
    }
    
    provider "google" {
      project = var.project_id
      region  = var.region
    }
    
    # Create GKE cluster
    resource "google_container_cluster" "primary" {
      name     = "swarm-cluster"
      location = var.region
      
      initial_node_count = 3
      
      node_config {
        preemptible  = true
        machine_type = "e2-standard-4"
        
        oauth_scopes = [
          "https://www.googleapis.com/auth/cloud-platform"
        ]
      }
    }
    
    # Create Cloud SQL instance
    resource "google_sql_database_instance" "main" {
      name             = "swarm-db-instance"
      database_version = "POSTGRES_14"
      region           = var.region
      
      settings {
        tier = "db-f1-micro"
      }
    }
    
    # Create storage buckets
    resource "google_storage_bucket" "data" {
      name     = "${var.project_id}-swarm-data"
      location = "US"
    }
    
    resource "google_storage_bucket" "backups" {
      name     = "${var.project_id}-swarm-backups"
      location = "US"
      
      lifecycle_rule {
        condition {
          age = 30
        }
        action {
          type = "Delete"
        }
      }
    }
    EOF
    
    # Create terraform.tfvars from environment
    cat > terraform.tfvars << EOF
    project_id = "$GCP_PROJECT_ID"
    EOF
    
    # Initialize and apply
    terraform init
    terraform plan -out=tfplan
    terraform apply tfplan
    
    # Save outputs
    terraform output -json > /swarm-state/terraform-outputs.json
    
    # Configure kubectl for the new cluster
    gcloud container clusters get-credentials swarm-cluster --region $REGION
    
    # Deploy initial resources
    kubectl create namespace swarm-apps
    kubectl create secret generic db-credentials \
      --from-literal=username=swarm \
      --from-literal=password=$(openssl rand -base64 32) \
      -n swarm-apps
    
    /scripts/checkpoint.sh save "infrastructure-ready" '{"status": "complete"}'
  executorImage: claudeflow/swarm-executor:2.0.0
  additionalSecrets:
  - name: gcp-credentials
    mountPath: /credentials/gcp
  persistentVolumes:
  - name: workspace
    mountPath: /workspace
    size: 10Gi
  - name: state
    mountPath: /swarm-state
    size: 5Gi
  environment:
    GCP_PROJECT_ID: "my-swarm-project"
    REGION: "us-central1"
  resources:
    limits:
      cpu: "4"
      memory: "8Gi"
  timeout: "1h"
```

### Example 3: Kubernetes Multi-Cluster Management

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: k8s-multi-cluster-setup
  namespace: claude-flow-swarm
spec:
  task: |
    #!/bin/bash
    set -e
    
    echo "🎮 Setting up multi-cluster Kubernetes management"
    
    # Install additional tools
    echo "📦 Installing cluster management tools..."
    
    # Install kubectx/kubens
    curl -L https://github.com/ahmetb/kubectx/releases/latest/download/kubectx -o /usr/local/bin/kubectx
    curl -L https://github.com/ahmetb/kubectx/releases/latest/download/kubens -o /usr/local/bin/kubens
    chmod +x /usr/local/bin/kubectx /usr/local/bin/kubens
    
    # Setup kubeconfigs
    mkdir -p ~/.kube/configs
    
    # Merge all mounted kubeconfigs
    KUBECONFIG=""
    for config in /credentials/kubeconfigs/*; do
      if [ -f "$config" ]; then
        KUBECONFIG="${KUBECONFIG}:${config}"
      fi
    done
    export KUBECONFIG="${KUBECONFIG:1}"  # Remove leading colon
    
    # List all contexts
    echo "📋 Available clusters:"
    kubectl config get-contexts
    
    # Install Istio on all clusters
    echo "🕸️ Installing Istio service mesh..."
    curl -L https://istio.io/downloadIstio | ISTIO_VERSION=1.20.0 sh -
    cd istio-*/
    export PATH=$PWD/bin:$PATH
    
    for context in $(kubectl config get-contexts -o name); do
      echo "Installing Istio on cluster: $context"
      kubectl config use-context $context
      
      istioctl install --set values.pilot.env.PILOT_ENABLE_STATUS=true -y
      kubectl label namespace default istio-injection=enabled
    done
    
    # Setup multi-cluster mesh
    echo "🔗 Configuring multi-cluster mesh..."
    
    # Get cluster endpoints
    CLUSTER1_CONTEXT=$(kubectl config get-contexts -o name | head -1)
    CLUSTER2_CONTEXT=$(kubectl config get-contexts -o name | tail -1)
    
    # Create multi-cluster secret
    kubectl config use-context $CLUSTER1_CONTEXT
    kubectl create secret generic cluster2-secret \
      --from-file=$HOME/.kube/config \
      -n istio-system
    
    # Deploy sample application across clusters
    echo "🚀 Deploying distributed application..."
    
    kubectl config use-context $CLUSTER1_CONTEXT
    kubectl apply -f - <<EOF
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: frontend
    spec:
      replicas: 3
      selector:
        matchLabels:
          app: frontend
      template:
        metadata:
          labels:
            app: frontend
        spec:
          containers:
          - name: app
            image: nginx:latest
            ports:
            - containerPort: 80
    EOF
    
    kubectl config use-context $CLUSTER2_CONTEXT
    kubectl apply -f - <<EOF
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: backend
    spec:
      replicas: 3
      selector:
        matchLabels:
          app: backend
      template:
        metadata:
          labels:
            app: backend
        spec:
          containers:
          - name: app
            image: httpd:latest
            ports:
            - containerPort: 80
    EOF
    
    # Save cluster configuration
    mkdir -p /swarm-state/clusters
    kubectl config view --minify --flatten > /swarm-state/clusters/merged-config.yaml
    
    /scripts/checkpoint.sh save "multi-cluster-ready" '{"clusters": 2, "istio": true}'
  executorImage: claudeflow/swarm-executor:2.0.0
  additionalSecrets:
  - name: cluster1-kubeconfig
    mountPath: /credentials/kubeconfigs/cluster1
  - name: cluster2-kubeconfig  
    mountPath: /credentials/kubeconfigs/cluster2
  persistentVolumes:
  - name: state
    mountPath: /swarm-state
    size: 10Gi
  resources:
    requests:
      cpu: "2"
      memory: "4Gi"
```

### Example 4: Data Pipeline with Persistent Storage

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: data-pipeline
  namespace: claude-flow-swarm
spec:
  task: |
    #!/bin/bash
    set -e
    
    echo "📊 Running data pipeline with persistent storage"
    
    # Check if resuming
    if [ "$RESUME_FROM_STEP" != "" ]; then
      echo "📂 Resuming from step: $RESUME_FROM_STEP"
      case $RESUME_FROM_STEP in
        "data-downloaded")
          echo "Skipping download, data already present"
          ;;
        "data-processed")
          echo "Skipping processing, using processed data"
          ;;
      esac
    fi
    
    # Step 1: Download data
    if [ "$RESUME_FROM_STEP" == "" ] || [ "$RESUME_FROM_STEP" == "start" ]; then
      echo "⬇️ Downloading dataset..."
      mkdir -p /data/raw
      
      # Download from GCS
      gsutil -m cp -r gs://public-datasets/wikipedia/2024/* /data/raw/
      
      /scripts/checkpoint.sh save "data-downloaded" '{"size_gb": 150}'
    fi
    
    # Step 2: Process data
    if [ "$RESUME_FROM_STEP" != "data-processed" ]; then
      echo "⚙️ Processing data..."
      python3 << 'EOF'
    import os
    import json
    import pandas as pd
    from concurrent.futures import ProcessPoolExecutor
    
    def process_file(filepath):
        # Process individual file
        df = pd.read_json(filepath, lines=True)
        # Data transformations
        df['processed'] = True
        df['timestamp'] = pd.Timestamp.now()
        
        output_path = filepath.replace('/raw/', '/processed/')
        os.makedirs(os.path.dirname(output_path), exist_ok=True)
        df.to_parquet(output_path, compression='snappy')
        return len(df)
    
    # Process all files in parallel
    raw_files = []
    for root, dirs, files in os.walk('/data/raw'):
        for file in files:
            if file.endswith('.json'):
                raw_files.append(os.path.join(root, file))
    
    total_records = 0
    with ProcessPoolExecutor(max_workers=8) as executor:
        results = executor.map(process_file, raw_files)
        total_records = sum(results)
    
    print(f"Processed {total_records} records from {len(raw_files)} files")
    
    # Save metadata
    metadata = {
        'total_files': len(raw_files),
        'total_records': total_records,
        'completion_time': str(pd.Timestamp.now())
    }
    
    with open('/data/metadata.json', 'w') as f:
        json.dump(metadata, f)
    EOF
      
      /scripts/checkpoint.sh save "data-processed" '{"status": "complete"}'
    fi
    
    # Step 3: Upload to cloud storage
    echo "☁️ Uploading processed data..."
    
    # Upload to GCS
    gsutil -m cp -r /data/processed/* gs://$BUCKET_NAME/processed/
    
    # Upload to S3
    aws s3 sync /data/processed/ s3://$S3_BUCKET/processed/ --storage-class GLACIER
    
    # Create BigQuery dataset
    echo "📊 Creating BigQuery tables..."
    bq mk --dataset --location=US ${GCP_PROJECT_ID}:wikipedia_processed
    
    for file in /data/processed/*.parquet; do
      table_name=$(basename $file .parquet)
      bq load \
        --source_format=PARQUET \
        --autodetect \
        ${GCP_PROJECT_ID}:wikipedia_processed.${table_name} \
        $file
    done
    
    echo "✅ Data pipeline complete!"
    
    # Clean up raw data to save space
    rm -rf /data/raw
    
    /scripts/checkpoint.sh save "pipeline-complete" '{"status": "success"}'
  executorImage: claudeflow/swarm-executor:2.0.0
  resume: true  # Enable resumption
  additionalSecrets:
  - name: gcp-credentials
    mountPath: /credentials/gcp
  - name: aws-credentials
    mountPath: /credentials/aws
  persistentVolumes:
  - name: data
    mountPath: /data
    storageClass: fast-ssd
    size: 500Gi
  - name: state
    mountPath: /swarm-state
    size: 10Gi
  environment:
    GCP_PROJECT_ID: "my-data-project"
    BUCKET_NAME: "my-data-bucket"
    S3_BUCKET: "my-s3-bucket"
  resources:
    requests:
      cpu: "4"
      memory: "16Gi"
    limits:
      cpu: "8"
      memory: "32Gi"
  nodeSelector:
    workload-type: data-processing
  tolerations:
  - key: "data-processing"
    operator: "Equal"
    value: "true"
    effect: "NoSchedule"
```

### Example 5: ML Model Training with GPU

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: ml-training-job
  namespace: claude-flow-swarm
spec:
  task: |
    #!/bin/bash
    set -e
    
    echo "🤖 Starting ML training job"
    
    # Install ML dependencies
    pip install torch torchvision transformers datasets accelerate wandb
    
    # Login to Weights & Biases
    wandb login $WANDB_API_KEY
    
    # Training script
    python3 << 'EOF'
    import torch
    import torch.nn as nn
    from torch.utils.data import DataLoader
    from transformers import AutoModelForSequenceClassification, AutoTokenizer
    from datasets import load_dataset
    import wandb
    from accelerate import Accelerator
    import os
    
    # Initialize accelerator for distributed training
    accelerator = Accelerator()
    
    # Initialize wandb
    wandb.init(project="swarm-training", name=f"run-{os.environ.get('TASK_NAME')}")
    
    # Load model and tokenizer
    model_name = "bert-base-uncased"
    model = AutoModelForSequenceClassification.from_pretrained(model_name, num_labels=2)
    tokenizer = AutoTokenizer.from_pretrained(model_name)
    
    # Load dataset
    dataset = load_dataset("imdb", split="train[:10000]")
    
    def tokenize_function(examples):
        return tokenizer(examples["text"], padding="max_length", truncation=True)
    
    tokenized_dataset = dataset.map(tokenize_function, batched=True)
    
    # Create data loader
    train_dataloader = DataLoader(tokenized_dataset, batch_size=32, shuffle=True)
    
    # Prepare for distributed training
    model, train_dataloader = accelerator.prepare(model, train_dataloader)
    
    # Training loop
    optimizer = torch.optim.AdamW(model.parameters(), lr=5e-5)
    num_epochs = 3
    
    for epoch in range(num_epochs):
        model.train()
        total_loss = 0
        
        for batch in train_dataloader:
            outputs = model(**batch)
            loss = outputs.loss
            accelerator.backward(loss)
            
            optimizer.step()
            optimizer.zero_grad()
            
            total_loss += loss.item()
        
        avg_loss = total_loss / len(train_dataloader)
        print(f"Epoch {epoch + 1}/{num_epochs}, Loss: {avg_loss:.4f}")
        wandb.log({"loss": avg_loss, "epoch": epoch + 1})
        
        # Save checkpoint
        if accelerator.is_main_process:
            checkpoint_path = f"/models/checkpoint-epoch-{epoch + 1}"
            model.save_pretrained(checkpoint_path)
            tokenizer.save_pretrained(checkpoint_path)
            
            # Also save to GCS
            os.system(f"gsutil -m cp -r {checkpoint_path} gs://{os.environ['BUCKET_NAME']}/models/")
    
    # Save final model
    if accelerator.is_main_process:
        final_path = "/models/final-model"
        model.save_pretrained(final_path)
        tokenizer.save_pretrained(final_path)
        
        # Upload to model registry
        os.system(f"gsutil -m cp -r {final_path} gs://{os.environ['BUCKET_NAME']}/models/final/")
        
        print("✅ Training complete! Model saved.")
    
    wandb.finish()
    EOF
    
    /scripts/checkpoint.sh save "training-complete" '{"model": "bert-fine-tuned"}'
  executorImage: claudeflow/swarm-executor:2.0.0
  additionalSecrets:
  - name: wandb-credentials
    mountPath: /secrets/wandb
  - name: gcp-credentials
    mountPath: /credentials/gcp
  persistentVolumes:
  - name: models
    mountPath: /models
    storageClass: fast-ssd
    size: 100Gi
  - name: datasets
    mountPath: /datasets
    size: 200Gi
  environment:
    WANDB_API_KEY: "$(cat /secrets/wandb/api-key)"
    BUCKET_NAME: "my-model-bucket"
  resources:
    requests:
      cpu: "8"
      memory: "32Gi"
      "nvidia.com/gpu": "1"
    limits:
      cpu: "16"
      memory: "64Gi"
      "nvidia.com/gpu": "1"
  nodeSelector:
    accelerator: nvidia-tesla-v100
```

### Example 6: GitOps Deployment Pipeline

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: gitops-deployment
  namespace: claude-flow-swarm
spec:
  task: |
    #!/bin/bash
    set -e
    
    echo "🚀 GitOps deployment pipeline"
    
    # Clone repositories
    echo "📦 Cloning repositories..."
    mkdir -p /workspace/repos
    cd /workspace/repos
    
    # Clone application repo
    git clone https://github.com/$GITHUB_ORG/my-app.git
    cd my-app
    
    # Build and test application
    echo "🔨 Building application..."
    make build
    make test
    
    # Build Docker image
    echo "🐳 Building Docker image..."
    export IMAGE_TAG="gcr.io/$GCP_PROJECT_ID/my-app:$(git rev-parse --short HEAD)"
    docker build -t $IMAGE_TAG .
    
    # Push to registry
    echo "📤 Pushing to registry..."
    docker push $IMAGE_TAG
    
    # Update GitOps repo
    echo "📝 Updating GitOps manifests..."
    cd /workspace/repos
    git clone https://github.com/$GITHUB_ORG/gitops-config.git
    cd gitops-config
    
    # Update image tag in Kustomization
    cd apps/my-app/overlays/production
    kustomize edit set image my-app=$IMAGE_TAG
    
    # Update with yq for Helm values
    yq eval ".image.tag = \"$(git rev-parse --short HEAD)\"" -i values-prod.yaml
    
    # Commit and push changes
    git add .
    git commit -m "Update my-app to ${IMAGE_TAG}"
    git push origin main
    
    # Trigger ArgoCD sync
    echo "🔄 Triggering ArgoCD sync..."
    argocd app sync my-app-production --force
    argocd app wait my-app-production --health
    
    # Run smoke tests
    echo "🧪 Running smoke tests..."
    kubectl run smoke-test --image=curlimages/curl --rm -it --restart=Never -- \
      curl -f http://my-app.production.svc.cluster.local/health
    
    # Send notification
    curl -X POST $SLACK_WEBHOOK -H 'Content-type: application/json' \
      -d "{\"text\":\"✅ Deployment complete: my-app ${IMAGE_TAG}\"}"
    
    /scripts/checkpoint.sh save "deployment-complete" "{\"image\": \"$IMAGE_TAG\"}"
  executorImage: claudeflow/swarm-executor:2.0.0
  additionalSecrets:
  - name: github-token
    mountPath: /secrets/github
  - name: dockerhub-creds
    mountPath: /secrets/docker
  - name: argocd-creds
    mountPath: /secrets/argocd
  - name: slack-webhook
    mountPath: /secrets/slack
  persistentVolumes:
  - name: workspace
    mountPath: /workspace
    size: 50Gi
  - name: docker-storage
    mountPath: /var/lib/docker
    size: 100Gi
  environment:
    GITHUB_ORG: "my-org"
    GCP_PROJECT_ID: "my-project"
    ARGOCD_SERVER: "argocd.my-domain.com"
    SLACK_WEBHOOK: "$(cat /secrets/slack/webhook-url)"
  resources:
    requests:
      cpu: "4"
      memory: "8Gi"
```

### Example 7: Disaster Recovery Automation

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: disaster-recovery
  namespace: claude-flow-swarm
spec:
  task: |
    #!/bin/bash
    set -e
    
    echo "🚨 Disaster Recovery Automation"
    
    # Define recovery functions
    check_cluster_health() {
      local context=$1
      kubectl config use-context $context
      
      # Check node status
      unhealthy_nodes=$(kubectl get nodes -o json | jq -r '.items[] | select(.status.conditions[] | select(.type=="Ready" and .status!="True")) | .metadata.name')
      
      if [ -n "$unhealthy_nodes" ]; then
        echo "⚠️ Unhealthy nodes detected: $unhealthy_nodes"
        return 1
      fi
      
      # Check critical deployments
      kubectl get deployments -A -o json | jq -r '.items[] | select(.status.replicas != .status.readyReplicas) | "\(.metadata.namespace)/\(.metadata.name)"' > /tmp/unhealthy-deployments
      
      if [ -s /tmp/unhealthy-deployments ]; then
        echo "⚠️ Unhealthy deployments:"
        cat /tmp/unhealthy-deployments
        return 1
      fi
      
      return 0
    }
    
    backup_cluster_state() {
      local cluster=$1
      local backup_dir="/backups/${cluster}/$(date +%Y%m%d-%H%M%S)"
      
      echo "💾 Backing up cluster state for $cluster"
      mkdir -p $backup_dir
      
      # Backup all resources
      for resource in $(kubectl api-resources --verbs=list -o name | grep -v events); do
        echo "Backing up $resource..."
        kubectl get $resource --all-namespaces -o yaml > "$backup_dir/${resource}.yaml" 2>/dev/null || true
      done
      
      # Backup etcd
      kubectl exec -n kube-system etcd-master-0 -- etcdctl \
        --endpoints=https://127.0.0.1:2379 \
        --cacert=/etc/kubernetes/pki/etcd/ca.crt \
        --cert=/etc/kubernetes/pki/etcd/healthcheck-client.crt \
        --key=/etc/kubernetes/pki/etcd/healthcheck-client.key \
        snapshot save /tmp/etcd-snapshot.db
      
      kubectl cp kube-system/etcd-master-0:/tmp/etcd-snapshot.db $backup_dir/etcd-snapshot.db
      
      # Upload to GCS
      gsutil -m cp -r $backup_dir gs://$BACKUP_BUCKET/cluster-backups/
    }
    
    restore_critical_services() {
      local backup_path=$1
      
      echo "🔄 Restoring critical services from $backup_path"
      
      # Restore in order of priority
      local priority_resources=(
        "namespaces"
        "serviceaccounts"
        "clusterroles"
        "clusterrolebindings"
        "persistentvolumes"
        "persistentvolumeclaims"
        "configmaps"
        "secrets"
        "services"
        "deployments"
        "statefulsets"
        "daemonsets"
      )
      
      for resource in "${priority_resources[@]}"; do
        if [ -f "$backup_path/${resource}.yaml" ]; then
          echo "Restoring $resource..."
          kubectl apply -f "$backup_path/${resource}.yaml" || true
        fi
      done
    }
    
    # Main recovery logic
    echo "🔍 Checking cluster health..."
    
    PRIMARY_CLUSTER="gke_${GCP_PROJECT_ID}_us-central1_primary"
    DR_CLUSTER="gke_${GCP_PROJECT_ID}_us-east1_dr"
    
    if ! check_cluster_health $PRIMARY_CLUSTER; then
      echo "❌ Primary cluster unhealthy, initiating failover!"
      
      # Backup current state
      backup_cluster_state $PRIMARY_CLUSTER
      
      # Switch to DR cluster
      kubectl config use-context $DR_CLUSTER
      
      # Scale up DR cluster
      echo "⚡ Scaling up DR cluster..."
      gcloud container clusters resize dr-cluster --zone us-east1-b --num-nodes 5
      
      # Restore latest backup
      LATEST_BACKUP=$(gsutil ls gs://$BACKUP_BUCKET/cluster-backups/ | sort -r | head -1)
      gsutil -m cp -r $LATEST_BACKUP /tmp/restore/
      
      restore_critical_services /tmp/restore/
      
      # Update DNS to point to DR cluster
      echo "🌐 Updating DNS..."
      gcloud dns record-sets transaction start --zone=$DNS_ZONE
      gcloud dns record-sets transaction remove \
        --name=api.myapp.com. \
        --ttl=300 \
        --type=A \
        --zone=$DNS_ZONE \
        "$PRIMARY_IP"
      gcloud dns record-sets transaction add \
        --name=api.myapp.com. \
        --ttl=300 \
        --type=A \
        --zone=$DNS_ZONE \
        "$DR_IP"
      gcloud dns record-sets transaction execute --zone=$DNS_ZONE
      
      # Send alerts
      curl -X POST $PAGERDUTY_WEBHOOK -H 'Content-type: application/json' \
        -d '{"event_action": "trigger", "payload": {"summary": "Primary cluster failure - DR activated", "severity": "critical"}}'
      
      echo "✅ Disaster recovery complete! Services now running on DR cluster."
    else
      echo "✅ Primary cluster healthy"
      
      # Perform regular backup
      backup_cluster_state $PRIMARY_CLUSTER
    fi
    
    /scripts/checkpoint.sh save "dr-check-complete" '{"status": "success"}'
  executorImage: claudeflow/swarm-executor:2.0.0
  additionalSecrets:
  - name: gcp-credentials
    mountPath: /credentials/gcp
  - name: pagerduty-webhook
    mountPath: /secrets/pagerduty
  - name: kubeconfigs
    mountPath: /credentials/kubeconfigs
  persistentVolumes:
  - name: backups
    mountPath: /backups
    size: 1Ti
    storageClass: standard
  environment:
    GCP_PROJECT_ID: "my-production-project"
    BACKUP_BUCKET: "my-dr-backups"
    DNS_ZONE: "my-dns-zone"
    PRIMARY_IP: "35.1.2.3"
    DR_IP: "35.4.5.6"
    PAGERDUTY_WEBHOOK: "$(cat /secrets/pagerduty/webhook)"
  resources:
    requests:
      cpu: "4"
      memory: "16Gi"
  schedule: "0 */6 * * *"  # Run every 6 hours
```

## Cloud Provider Setup

### Google Cloud Platform

1. Create service account and download key:
```bash
gcloud iam service-accounts create swarm-executor \
  --display-name="Swarm Executor Service Account"

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:swarm-executor@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/editor"

gcloud iam service-accounts keys create key.json \
  --iam-account=swarm-executor@$PROJECT_ID.iam.gserviceaccount.com
```

2. Create Kubernetes secret:
```bash
kubectl create secret generic gcp-credentials \
  --from-file=key.json \
  --namespace=default
```

### AWS

1. Create IAM user and access keys:
```bash
aws iam create-user --user-name swarm-executor
aws iam attach-user-policy --user-name swarm-executor \
  --policy-arn arn:aws:iam::aws:policy/PowerUserAccess
aws iam create-access-key --user-name swarm-executor
```

2. Create credentials file:
```bash
cat > credentials << EOF
[default]
aws_access_key_id = YOUR_ACCESS_KEY
aws_secret_access_key = YOUR_SECRET_KEY
EOF

cat > config << EOF
[default]
region = us-east-1
output = json
EOF
```

3. Create Kubernetes secret:
```bash
kubectl create secret generic aws-credentials \
  --from-file=credentials \
  --from-file=config \
  --namespace=default
```

### Azure

1. Create service principal:
```bash
az ad sp create-for-rbac --name swarm-executor \
  --role Contributor \
  --scopes /subscriptions/$SUBSCRIPTION_ID
```

2. Create Kubernetes secret:
```bash
kubectl create secret generic azure-credentials \
  --from-literal=client_id=$CLIENT_ID \
  --from-literal=client_secret=$CLIENT_SECRET \
  --from-literal=tenant_id=$TENANT_ID \
  --from-literal=subscription_id=$SUBSCRIPTION_ID \
  --namespace=default
```

## Persistent Storage

### Storage Classes

Create custom storage classes for different performance tiers:

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: fast-ssd
provisioner: kubernetes.io/gce-pd
parameters:
  type: pd-ssd
  replication-type: regional-pd
allowVolumeExpansion: true
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: standard
provisioner: kubernetes.io/gce-pd
parameters:
  type: pd-standard
  replication-type: none
allowVolumeExpansion: true
```

### Volume Management

The operator automatically:
- Creates PVCs for each persistent volume defined
- Reuses existing PVCs if they match the task name
- Cleans up PVCs based on retention policies

## Task Resumption

### How It Works

1. Tasks save checkpoints during execution:
```bash
/scripts/checkpoint.sh save "step-name" '{"progress": 50}'
```

2. When a task fails, set `resume: true` to retry:
```yaml
spec:
  resume: true
```

3. The task resumes from the last checkpoint:
- Workspace state is restored
- Environment variables indicate resume point
- Task logic can skip completed steps

### Best Practices

1. **Save checkpoints frequently** for long-running tasks
2. **Use meaningful checkpoint names** that indicate progress
3. **Store minimal state** in checkpoint data
4. **Clean up old checkpoints** to save storage

## Advanced Configuration

### Resource Management

```yaml
resources:
  requests:
    cpu: "2"
    memory: "4Gi"
  limits:
    cpu: "4"
    memory: "8Gi"
    "nvidia.com/gpu": "1"  # For GPU workloads
```

### Node Selection

```yaml
nodeSelector:
  workload-type: compute-intensive
  disk: ssd
```

### Tolerations

```yaml
tolerations:
- key: "gpu"
  operator: "Equal"
  value: "true"
  effect: "NoSchedule"
- key: "preemptible"
  operator: "Exists"
  effect: "NoSchedule"
```

### Environment Variables

```yaml
environment:
  PROJECT_ID: "my-project"
  REGION: "us-central1"
  DEBUG: "true"
```

## Monitoring and Observability

### Prometheus Metrics

The operator exposes metrics at `:8080/metrics`:
- Task counts by status
- Job duration histograms
- Resource utilization
- Checkpoint save/restore stats

### Logging

View operator logs:
```bash
kubectl logs -n swarm-system deployment/swarm-operator -f
```

View task executor logs:
```bash
kubectl logs job/swarm-job-<task-name> -f
```

### Health Checks

- Liveness: `:8081/healthz`
- Readiness: `:8081/readyz`

## Troubleshooting

### Common Issues

1. **PVC Creation Fails**
   - Check storage class exists
   - Verify quota limits
   - Check node availability

2. **Cloud Authentication Fails**
   - Verify secret exists and is mounted
   - Check credential permissions
   - Test with cloud CLI in pod

3. **Task Timeout**
   - Increase timeout in task spec
   - Add more checkpoints
   - Check resource limits

4. **Resume Not Working**
   - Verify checkpoint was saved
   - Check PVC is accessible
   - Review checkpoint loading logs

### Debug Mode

Enable debug logging:
```yaml
environment:
  DEBUG: "true"
  LOG_LEVEL: "debug"
```

### Pod Debugging

```bash
# Execute into a running task
kubectl exec -it job/swarm-job-<task-name> -- /bin/bash

# Check checkpoint files
ls -la /swarm-state/

# View environment
env | grep -E "(TASK|SWARM|RESUME)"
```

## Security Best Practices

1. **Use least-privilege service accounts**
2. **Rotate credentials regularly**
3. **Encrypt secrets at rest**
4. **Use private container registries**
5. **Enable pod security policies**
6. **Audit cloud API usage**

## Performance Optimization

1. **Use appropriate storage classes** for workload types
2. **Enable volume expansion** for growing datasets
3. **Use node affinity** for data locality
4. **Leverage spot/preemptible instances** for cost savings
5. **Monitor resource utilization** and adjust limits

## Conclusion

The Enhanced Swarm Operator provides a powerful platform for running complex, stateful tasks in Kubernetes with full cloud provider integration. With persistent storage, task resumption, and comprehensive tooling, it enables reliable execution of long-running workflows while maintaining cost efficiency and operational excellence.