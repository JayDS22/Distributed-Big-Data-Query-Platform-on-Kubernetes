# Distributed Big Data Query Platform (DBQP) on Kubernetes

A production-grade platform for deploying and managing Trino and Spark clusters on Kubernetes with disaggregated compute/storage architecture.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                      DBQP CLI Tool                              │
│  (Go: create, scale, delete, status, benchmark, logs commands)  │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│              Kubernetes Cluster Management                       │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────────┐│
│  │  Trino Cluster  │  │  Spark Cluster  │  │  Custom Operator ││
│  │  - Coordinator  │  │  - Master       │  │  (CRDs)          ││
│  │  - Workers      │  │  - Workers      │  │  - TrinoCluster  ││
│  │  - Services     │  │  - Services     │  │  - SparkCluster  ││
│  └─────────────────┘  └─────────────────┘  └──────────────────┘│
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────────┐│
│  │ HPA (Scaling)   │  │ Liveness Probes │  │ Health Checks    ││
│  │ ConfigMaps      │  │ Readiness Probes│  │ Monitoring       ││
│  │ StatefulSets    │  │ Resource Limits │  │ Logging          ││
│  └─────────────────┘  └─────────────────┘  └──────────────────┘│
└─────────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│        Disaggregated Storage Layer (S3/MinIO)                   │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────┐  ┌──────────────────┐  ┌───────────────┐ │
│  │   MinIO S3       │  │ Hive Metastore   │  │ Prometheus    │ │
│  │   (Data)         │  │ (Metadata)       │  │ (Monitoring)  │ │
│  └──────────────────┘  └──────────────────┘  └───────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## Features

### Core Components

1. **Golang CLI Tool** - Interactive command-line interface
   - Create/scale/delete/monitor clusters
   - Run benchmarks (TPC-H, TPC-DS)
   - Stream logs and capture metrics
   - Kubernetes API integration

2. **Kubernetes Operator** - Automates cluster lifecycle
   - Custom Resource Definitions (CRDs)
   - Pod lifecycle management
   - Horizontal Pod Autoscaling (HPA)
   - Rolling upgrades and health checks

3. **Disaggregated Architecture**
   - Compute nodes (Trino/Spark workers)
   - Storage layer (MinIO S3-compatible)
   - Hive Metastore for table metadata
   - Independent scaling of compute/storage

4. **Python Benchmarking Suite**
   - TPC-H and TPC-DS query execution
   - Performance metrics collection
   - Results export (JSON, CSV)
   - Visualization with matplotlib

5. **Shell Automation**
   - Cluster provisioning setup
   - Health check diagnostics
   - Log collection utilities
   - Environment cleanup

6. **CI/CD Pipeline (GitHub Actions)**
   - Go/Python testing and linting
   - Docker image building
   - Kubernetes integration tests
   - Security scanning (Trivy, GoSec)

## Quick Start

### Prerequisites

- Kubernetes cluster (v1.24+)
- `kubectl` configured
- Docker (for building images)
- Go 1.21+
- Python 3.9+

### Installation

```bash
# Clone repository
git clone https://github.com/yourusername/dbqp.git
cd dbqp

# Build CLI
go build -o dbqp .

# Setup cluster
./scripts/setup-cluster.sh kind default

# Verify health
./scripts/health-check.sh default
```

## Usage

### Create a Trino Cluster

```bash
# Create 5-worker Trino cluster
./dbqp create --engine trino --workers 5 --memory 8Gi --cpu 4

# Create Spark cluster
./dbqp create --engine spark --workers 3 --memory 8Gi --cpu 4
```

### Scale a Cluster

```bash
# Enable autoscaling for workers
./dbqp scale --name trino-1234567890 --min 2 --max 10 --cpu-percent 70
```

### Monitor Cluster Status

```bash
# Check cluster health
./dbqp status trino-1234567890

# List all clusters
./dbqp list

# Stream logs from cluster
./dbqp logs trino-1234567890 --tail 100 --follow
```

### Run Benchmarks

```bash
# Run TPC-H Q1 on Trino
./dbqp benchmark --engine trino --query tpch-q1 --scale 10 --iterations 3 --output json

# Compare Trino vs Spark
./dbqp benchmark --engine spark --query tpch-q22 --scale 1
```

### Cleanup

```bash
# Delete a cluster
./dbqp delete trino-1234567890

# Full environment cleanup
./scripts/cleanup.sh default --force
```

## Configuration

### Trino Configuration

Edit `k8s/trino-deployment.yaml` to customize:
- Coordinator/worker resource allocation
- Query execution limits
- S3/MinIO endpoint configuration
- Catalog settings

### Spark Configuration

Modify spark configuration in `python/benchmark_runner.py`:
- Driver/executor memory
- Parallelism settings
- Dynamic allocation thresholds

## Kubernetes Deployment

### Using Helm

```bash
# Deploy Trino cluster
helm install my-trino k8s/helm/trino \
  --set workers=3 \
  --set memory=8Gi \
  --set storageBucket=s3://data

# Deploy Spark cluster
helm install my-spark k8s/helm/spark \
  --set workers=3 \
  --set memory=8Gi
```

### Manual Deployment

```bash
# Apply CRDs
kubectl apply -f k8s/crd.yaml

# Deploy Trino
kubectl apply -f k8s/trino-deployment.yaml

# Deploy MinIO
kubectl apply -f k8s/minio-deployment.yaml

# Deploy Hive Metastore
kubectl apply -f k8s/metastore-deployment.yaml
```

## Monitoring

### View Metrics

```bash
# Port forward to Prometheus
kubectl port-forward svc/prometheus-server 9090:80

# Access Trino UI
kubectl port-forward svc/trino-coordinator 8080:8080

# Access Spark UI
kubectl port-forward svc/spark-master 4040:4040
```

### Health Checks

```bash
# Comprehensive cluster diagnostic
./scripts/health-check.sh default

# Pod resource usage
kubectl top pods -n default

# Recent events
kubectl get events -n default --sort-by='.lastTimestamp'
```

## Performance Tuning

### Memory Management
- Set `query.max-memory` in Trino config
- Adjust heap sizes for both engines
- Monitor memory utilization

### CPU Optimization
- Configure task parallelism
- Tune worker thread count
- Set appropriate resource limits

### Storage Optimization
- Enable S3 caching
- Configure block size (64MB recommended)
- Use compression (Snappy/Gzip)

## Troubleshooting

### Cluster Stuck in Pending

```bash
# Check pod events
kubectl describe pod <pod-name> -n default

# Check resource availability
kubectl describe nodes

# Check PVC status
kubectl get pvc -n default
```

### High Memory Usage

```bash
# Monitor memory usage
kubectl top pods -n default

# Check GC logs
kubectl logs <trino-pod> | grep "GC"

# Adjust heap size in deployment
```

### Query Failures

```bash
# Check query logs
./dbqp logs <cluster-name> --tail 500

# Verify Metastore connectivity
kubectl exec <coordinator-pod> -- curl hive-metastore:9083

# Check S3 credentials
kubectl get secret s3-credentials -o yaml
```

## CI/CD Pipeline

The GitHub Actions workflow automatically:

1. **Builds** - Go binaries, Docker images
2. **Tests** - Unit tests (Go/Python), integration tests
3. **Scans** - Security vulnerabilities (Trivy, GoSec)
4. **Deploys** - Docker images to registry
5. **Reports** - Coverage and status to GitHub

## Project Structure

```
dbqp/
├── cmd/                      # CLI commands
│   ├── create.go
│   ├── scale.go
│   ├── delete.go
│   ├── status.go
│   └── benchmark.go
├── pkg/
│   ├── cluster/             # Cluster configuration
│   └── k8s/                 # Kubernetes helpers
├── python/
│   ├── benchmark_runner.py  # TPC benchmarking
│   └── requirements.txt
├── k8s/
│   ├── crd.yaml             # CRD definitions
│   ├── trino-deployment.yaml
│   └── helm/                # Helm charts
├── scripts/
│   ├── setup-cluster.sh
│   ├── health-check.sh
│   └── cleanup.sh
├── .github/
│   └── workflows/
│       └── cicd.yml
├── main.go
├── go.mod
└── Dockerfile
```

## Performance Benchmarks

Example results from TPC-H Scale 10:

| Query | Trino | Spark | Speedup |
|-------|-------|-------|---------|
| Q1    | 4.2s  | 6.1s  | 1.45x   |
| Q3    | 2.8s  | 4.5s  | 1.61x   |
| Q5    | 3.5s  | 5.2s  | 1.49x   |
| Q22   | 1.9s  | 3.1s  | 1.63x   |

## Contributing

1. Fork repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

## License

MIT License - see LICENSE file

## Support

- Issues: GitHub Issues
- Documentation: See `/docs` directory
- Community: GitHub Discussions

## Roadmap

- [ ] Web UI Dashboard (React/Streamlit)
- [ ] Query result caching
- [ ] Cost optimization recommendations
- [ ] Multi-cloud deployment
- [ ] Query federation between clusters
- [ ] ML-based query optimization
- [ ] Federated learning support

## Contact

For questions or support, contact: support@dbqp.dev
