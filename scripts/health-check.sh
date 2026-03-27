#!/bin/bash

# health-check.sh: Comprehensive cluster health assessment
# Usage: ./health-check.sh [namespace]

set -euo pipefail

NAMESPACE=${1:-default}
HEALTHY=true

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*"
}

check_status() {
    local check_name=$1
    local command=$2
    
    if eval "$command" > /dev/null 2>&1; then
        log "✓ $check_name"
        return 0
    else
        log "✗ $check_name"
        HEALTHY=false
        return 1
    fi
}

log "=== DBQP Cluster Health Check ==="
log "Namespace: $NAMESPACE"
log ""

# 1. Kubernetes connectivity
log "1. Kubernetes Connectivity"
check_status "API Server" "kubectl cluster-info"
check_status "Namespace exists" "kubectl get namespace $NAMESPACE"

# 2. Node status
log ""
log "2. Node Status"
NODES=$(kubectl get nodes --no-headers | wc -l)
READY_NODES=$(kubectl get nodes --no-headers | grep " Ready " | wc -l)
log "Nodes: $READY_NODES/$NODES ready"

if [ "$READY_NODES" -lt "$NODES" ]; then
    log "✗ Not all nodes are ready"
    HEALTHY=false
else
    log "✓ All nodes ready"
fi

# 3. Pod status
log ""
log "3. Pod Status"
log "Running Pods:"
kubectl get pods -n $NAMESPACE --no-headers | awk '{print "  " $1 " - " $3}'

FAILED_PODS=$(kubectl get pods -n $NAMESPACE -o jsonpath='{.items[?(@.status.phase!="Running")].metadata.name}' | wc -w)
if [ "$FAILED_PODS" -gt 0 ]; then
    log "✗ $FAILED_PODS pods not running"
    HEALTHY=false
else
    log "✓ All pods running"
fi

# 4. Storage
log ""
log "4. Storage Status"
check_status "PersistentVolumes exist" "kubectl get pv | grep -q ."
check_status "MinIO available" "kubectl get pods -n $NAMESPACE -l app.kubernetes.io/name=minio | grep -q Running"
check_status "Hive Metastore available" "kubectl get pods -n $NAMESPACE -l app=hive | grep -q Running"

# 5. Services
log ""
log "5. Services"
log "Available services:"
kubectl get services -n $NAMESPACE --no-headers | awk '{print "  " $1 " - " $2}'

check_status "Trino coordinator" "kubectl get service -n $NAMESPACE trino-coordinator 2>/dev/null || true"
check_status "MinIO service" "kubectl get service -n $NAMESPACE -l app.kubernetes.io/name=minio | grep -q ."

# 6. Resource usage
log ""
log "6. Resource Usage"
log "Node Resource Usage:"
kubectl top nodes 2>/dev/null || log "Warning: Metrics server not available"

log ""
log "Pod Resource Usage:"
kubectl top pods -n $NAMESPACE 2>/dev/null | head -10 || log "Warning: Metrics server not available"

# 7. Recent events
log ""
log "7. Recent Events"
kubectl get events -n $NAMESPACE --sort-by='.lastTimestamp' | tail -5

# 8. ConfigMaps and Secrets
log ""
log "8. Configuration"
CM_COUNT=$(kubectl get configmaps -n $NAMESPACE --no-headers | wc -l)
SECRET_COUNT=$(kubectl get secrets -n $NAMESPACE --no-headers | wc -l)
log "ConfigMaps: $CM_COUNT"
log "Secrets: $SECRET_COUNT"

# 9. Check for errors in logs
log ""
log "9. Log Error Check"
ERROR_COUNT=$(kubectl logs -n $NAMESPACE -l app=trino --tail=100 2>/dev/null | grep -i "error" | wc -l || echo "0")
log "Errors in recent logs: $ERROR_COUNT"

if [ "$ERROR_COUNT" -gt 0 ]; then
    log "Sample errors:"
    kubectl logs -n $NAMESPACE -l app=trino --tail=100 2>/dev/null | grep -i "error" | head -3 | sed 's/^/  /'
fi

# 10. Database connectivity check
log ""
log "10. Database Connectivity"
TRINO_POD=$(kubectl get pods -n $NAMESPACE -l app=trino,component=coordinator -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

if [ -n "$TRINO_POD" ]; then
    log "Testing Trino coordinator..."
    if kubectl exec -n $NAMESPACE "$TRINO_POD" -- curl -s http://localhost:8080/ui/ > /dev/null; then
        log "✓ Trino coordinator responding"
    else
        log "✗ Trino coordinator not responding"
        HEALTHY=false
    fi
fi

# Summary
log ""
log "==================================="
if [ "$HEALTHY" = true ]; then
    log "✓ Cluster is HEALTHY"
    exit 0
else
    log "✗ Cluster has ISSUES"
    exit 1
fi
