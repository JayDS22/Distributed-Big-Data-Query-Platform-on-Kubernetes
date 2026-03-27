#!/bin/bash

# setup-cluster.sh: Initialize Kubernetes cluster with all prerequisites
# Usage: ./setup-cluster.sh [cluster-type] [namespace]

set -euo pipefail

CLUSTER_TYPE=${1:-kind}
NAMESPACE=${2:-default}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*"
}

error() {
    log "ERROR: $*"
    exit 1
}

log "=== Setting up DBQP Cluster ==="
log "Type: $CLUSTER_TYPE"
log "Namespace: $NAMESPACE"

# 1. Create cluster if needed
if [ "$CLUSTER_TYPE" = "kind" ]; then
    log "Creating kind cluster..."
    if ! kind get clusters | grep -q dbqp; then
        kind create cluster --name dbqp --config - << EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 8080
    hostPort: 8080
    protocol: TCP
  - containerPort: 7077
    hostPort: 7077
    protocol: TCP
- role: worker
- role: worker
- role: worker
EOF
        log "✓ Kind cluster created"
    else
        log "✓ Kind cluster already exists"
    fi
fi

# 2. Create namespace
log "Creating namespace..."
kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
log "✓ Namespace created"

# 3. Create S3 credentials secret
log "Creating S3 credentials secret..."
if [ -z "${AWS_ACCESS_KEY_ID:-}" ]; then
    log "Warning: AWS credentials not set. Using defaults for MinIO."
    AWS_ACCESS_KEY_ID="minioadmin"
    AWS_SECRET_ACCESS_KEY="minioadmin"
fi

kubectl create secret generic s3-credentials \
    --from-literal=access-key="$AWS_ACCESS_KEY_ID" \
    --from-literal=secret-key="$AWS_SECRET_ACCESS_KEY" \
    -n $NAMESPACE \
    --dry-run=client -o yaml | kubectl apply -f -
log "✓ S3 credentials secret created"

# 4. Install Helm repos
log "Setting up Helm repositories..."
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add trinodb https://trinodb.github.io/charts
helm repo update
log "✓ Helm repos updated"

# 5. Install MinIO for S3-compatible storage
log "Installing MinIO..."
helm upgrade --install minio bitnami/minio \
    --namespace $NAMESPACE \
    --set auth.username=minioadmin \
    --set auth.password=minioadmin \
    --set service.type=LoadBalancer \
    --set persistence.size=100Gi \
    --wait \
    --timeout 5m
log "✓ MinIO installed"

# 6. Install Hive Metastore
log "Installing Hive Metastore..."
helm upgrade --install hive-metastore bitnami/hive \
    --namespace $NAMESPACE \
    --wait \
    --timeout 5m
log "✓ Hive Metastore installed"

# 7. Apply CRDs
log "Installing Custom Resource Definitions..."
kubectl apply -f "$SCRIPT_DIR/../k8s/crd.yaml" -n $NAMESPACE
log "✓ CRDs installed"

# 8. Create sample MinIO bucket
log "Creating MinIO bucket..."
sleep 10  # Wait for MinIO to be ready
MINIO_POD=$(kubectl get pods -n $NAMESPACE -l app.kubernetes.io/name=minio -o jsonpath='{.items[0].metadata.name}')

kubectl exec -it -n $NAMESPACE "$MINIO_POD" -- \
    mc config host add local http://localhost:9000 minioadmin minioadmin && \
    kubectl exec -it -n $NAMESPACE "$MINIO_POD" -- \
    mc mb local/data || log "Warning: Bucket may already exist"
log "✓ MinIO bucket created"

# 9. Install Prometheus for monitoring (optional)
log "Installing Prometheus for monitoring..."
helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
    --namespace $NAMESPACE \
    --set prometheus.prometheusSpec.retention=7d \
    --wait \
    --timeout 5m 2>/dev/null || log "Warning: Prometheus installation failed (optional)"

log "=== Cluster setup complete ==="
log ""
log "Next steps:"
log "1. Verify cluster status: kubectl get nodes -n $NAMESPACE"
log "2. Check pods: kubectl get pods -n $NAMESPACE"
log "3. Create cluster: dbqp create --engine trino --workers 3"
log ""
