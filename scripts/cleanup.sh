#!/bin/bash

# cleanup.sh: Clean up all DBQP resources
# Usage: ./cleanup.sh [namespace] [--force]

set -euo pipefail

NAMESPACE=${1:-default}
FORCE=${2:-false}

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*"
}

log "=== DBQP Cleanup ==="
log "Namespace: $NAMESPACE"

if [ "$FORCE" != "--force" ]; then
    log ""
    log "This will delete all DBQP resources in namespace '$NAMESPACE'"
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log "Cleanup cancelled"
        exit 0
    fi
fi

# Delete clusters
log "Deleting Trino clusters..."
kubectl delete trinoclusters --all -n $NAMESPACE --ignore-not-found 2>/dev/null || true
sleep 5

log "Deleting Spark clusters..."
kubectl delete sparkclusters --all -n $NAMESPACE --ignore-not-found 2>/dev/null || true
sleep 5

# Delete Helm releases
log "Deleting Helm releases..."
helm list -n $NAMESPACE --short 2>/dev/null | xargs -I {} helm uninstall {} -n $NAMESPACE || true

# Delete StatefulSets and Deployments
log "Deleting workloads..."
kubectl delete statefulset,deployment,daemonset --all -n $NAMESPACE --ignore-not-found 2>/dev/null || true

# Delete PVCs
log "Deleting persistent volumes..."
kubectl delete pvc --all -n $NAMESPACE --ignore-not-found 2>/dev/null || true

# Delete Services
log "Deleting services..."
kubectl delete service --all -n $NAMESPACE --ignore-not-found 2>/dev/null || true

# Delete ConfigMaps and Secrets
log "Deleting configuration..."
kubectl delete configmap,secret --all -n $NAMESPACE --ignore-not-found 2>/dev/null || true

# Wait for cleanup
log "Waiting for resource cleanup..."
sleep 10

# Verify cleanup
POD_COUNT=$(kubectl get pods -n $NAMESPACE --no-headers 2>/dev/null | wc -l || echo "0")
PVC_COUNT=$(kubectl get pvc -n $NAMESPACE --no-headers 2>/dev/null | wc -l || echo "0")

if [ "$POD_COUNT" -eq 0 ] && [ "$PVC_COUNT" -eq 0 ]; then
    log "✓ Cleanup completed successfully"
    log "Namespace '$NAMESPACE' is empty"
else
    log "Warning: Some resources may still exist"
    log "Pods: $POD_COUNT"
    log "PVCs: $PVC_COUNT"
fi

log ""
log "To delete the namespace completely, run:"
log "  kubectl delete namespace $NAMESPACE"
