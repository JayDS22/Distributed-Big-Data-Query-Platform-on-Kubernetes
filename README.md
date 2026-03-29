# Distributed Big Data Query Platform (DBQP) on Kubernetes

A production-grade platform for deploying and managing Trino and Spark clusters on Kubernetes with disaggregated compute/storage architecture, custom CRDs, autoscaling, and full CI/CD automation.

---

## System Architecture

```mermaid
graph TB
    subgraph CLIENT["🖥️ Client Layer"]
        CLI["DBQP CLI Tool<br/>(Go Binary)"]
        API["REST API<br/>(kubectl proxy)"]
    end

    subgraph K8S["☸️ Kubernetes Control Plane"]
        direction TB
        APISERVER["kube-apiserver"]
        CRD["Custom Resource Definitions<br/>TrinoCluster | SparkCluster"]
        OPERATOR["DBQP Operator<br/>(Reconciliation Loop)"]
        HPA["Horizontal Pod Autoscaler"]
        CONFIGMAP["ConfigMaps<br/>(Engine Configs)"]
    end

    subgraph COMPUTE["⚡ Compute Layer (Stateless, Autoscaled)"]
        direction LR
        subgraph TRINO["Trino Cluster"]
            TC["Coordinator<br/>8GB / 4 vCPU"]
            TW1["Worker 1"]
            TW2["Worker 2"]
            TWN["Worker N"]
        end
        subgraph SPARK["Spark Cluster"]
            SM["Driver<br/>8GB / 4 vCPU"]
            SE1["Executor 1"]
            SE2["Executor 2"]
            SEN["Executor N"]
        end
    end

    subgraph STORAGE["💾 Disaggregated Storage Layer"]
        MINIO["MinIO S3<br/>(Object Storage)"]
        HIVE["Hive Metastore<br/>(Table Metadata)"]
        PV["Persistent Volumes<br/>(StatefulSets)"]
    end

    subgraph OBSERVE["📊 Observability"]
        PROM["Prometheus<br/>(Metrics)"]
        HEALTH["Health Checks<br/>(Liveness + Readiness)"]
        LOGS["Log Aggregation<br/>(stdout/stderr)"]
    end

    CLI --> APISERVER
    API --> APISERVER
    APISERVER --> CRD
    CRD --> OPERATOR
    OPERATOR --> TRINO
    OPERATOR --> SPARK
    HPA --> TW1 & TW2 & TWN
    HPA --> SE1 & SE2 & SEN
    CONFIGMAP --> TC & SM

    TC --> MINIO
    TC --> HIVE
    SM --> MINIO
    SM --> HIVE
    MINIO --> PV

    PROM --> TC & SM & TW1 & SE1
    HEALTH --> TC & SM

    classDef client fill:#1a1a2e,stroke:#e94560,color:#fff,stroke-width:2px
    classDef k8s fill:#0f3460,stroke:#533483,color:#fff,stroke-width:2px
    classDef compute fill:#16213e,stroke:#0f3460,color:#e94560,stroke-width:2px
    classDef storage fill:#1a1a2e,stroke:#533483,color:#fff,stroke-width:2px
    classDef observe fill:#0f3460,stroke:#e94560,color:#fff,stroke-width:2px

    class CLI,API client
    class APISERVER,CRD,OPERATOR,HPA,CONFIGMAP k8s
    class TC,TW1,TW2,TWN,SM,SE1,SE2,SEN compute
    class MINIO,HIVE,PV storage
    class PROM,HEALTH,LOGS observe
```

---

## Cluster Lifecycle & Autoscaling Flow

```mermaid
sequenceDiagram
    participant User as 👤 User (CLI)
    participant API as ☸️ K8s API
    participant Op as 🔄 DBQP Operator
    participant HPA as 📈 HPA Controller
    participant Pods as 🟢 Worker Pods
    participant S3 as 💾 MinIO S3

    User->>API: dbqp create --engine trino --workers 5
    API->>Op: TrinoCluster CR created
    Op->>Op: Reconcile: desired=5, current=0
    Op->>Pods: Create Coordinator Pod
    Op->>Pods: Create Worker Pods (1..5)
    Pods->>S3: Mount S3 catalog (Hive Metastore)
    Op->>API: Status: RUNNING, workers=5

    Note over User,S3: Cluster is live, executing queries

    User->>API: dbqp scale --min 2 --max 10 --cpu 70%
    API->>HPA: Create HPA (targetCPU=70%)

    Note over HPA,Pods: Traffic spike detected

    HPA->>HPA: Avg CPU = 85% > target 70%
    HPA->>Pods: Scale 5 → 8 workers
    Pods-->>HPA: 8 pods running, CPU stabilized

    Note over HPA,Pods: Traffic drops overnight

    HPA->>HPA: Avg CPU = 20% < target 70%
    HPA->>Pods: Scale 8 → 2 workers (minReplicas)
    Pods-->>HPA: 2 pods running, cost optimized

    User->>API: dbqp delete trino-1234567890
    API->>Op: TrinoCluster CR deleted
    Op->>Pods: Graceful shutdown (all pods)
    Op->>API: Status: DELETED
```

---

## Disaggregated Compute/Storage Architecture

```mermaid
graph LR
    subgraph COMPUTE["⚡ Compute Tier<br/>(Ephemeral, Autoscaled)"]
        direction TB
        T["Trino Coordinator"]
        TW["Trino Workers<br/>× N (HPA managed)"]
        S["Spark Driver"]
        SW["Spark Executors<br/>× N (Dynamic Alloc)"]
    end

    subgraph NETWORK["🔗 Network Layer"]
        SVC_T["ClusterIP Service<br/>trino-coordinator:8080"]
        SVC_S["ClusterIP Service<br/>spark-master:7077"]
        SVC_H["ClusterIP Service<br/>hive-metastore:9083"]
    end

    subgraph STORAGE["💾 Storage Tier<br/>(Persistent, Independent)"]
        direction TB
        MINIO["MinIO S3<br/>Data Lake<br/>(Parquet/ORC/CSV)"]
        HIVE["Hive Metastore<br/>Schema Registry<br/>(MySQL backend)"]
        PROM["Prometheus TSDB<br/>Metrics Store<br/>(15d retention)"]
    end

    T --> SVC_T
    TW --> T
    S --> SVC_S
    SW --> S

    T -->|"S3A connector<br/>read/write data"| MINIO
    S -->|"Hadoop S3A<br/>read/write data"| MINIO
    T -->|"Thrift protocol<br/>schema lookups"| HIVE
    S -->|"Thrift protocol<br/>schema lookups"| HIVE
    T -->|"/metrics endpoint"| PROM
    S -->|"/metrics endpoint"| PROM

    classDef compute fill:#0d1117,stroke:#58a6ff,color:#c9d1d9,stroke-width:2px
    classDef network fill:#161b22,stroke:#8b949e,color:#c9d1d9,stroke-width:1px
    classDef storage fill:#0d1117,stroke:#3fb950,color:#c9d1d9,stroke-width:2px

    class T,TW,S,SW compute
    class SVC_T,SVC_S,SVC_H network
    class MINIO,HIVE,PROM storage
```

---

## CI/CD Pipeline

```mermaid
graph LR
    subgraph TRIGGER["🔀 Trigger"]
        PR["Pull Request"]
        PUSH["Push to main"]
    end

    subgraph BUILD["🔨 Build & Test"]
        GO["Go Build<br/>+ Unit Tests"]
        PY["Python Lint<br/>+ Pytest"]
        DOCKER["Docker Build<br/>(Multi-stage)"]
    end

    subgraph SECURITY["🔒 Security Scan"]
        TRIVY["Trivy<br/>(Container CVEs)"]
        GOSEC["GoSec<br/>(Static Analysis)"]
    end

    subgraph VALIDATE["✅ Integration"]
        KIND["Kind Cluster<br/>(Ephemeral)"]
        CRD_T["Apply CRDs"]
        E2E["E2E Tests<br/>(Create/Scale/Delete)"]
    end

    subgraph DEPLOY["🚀 Deploy"]
        GHCR["Push to GHCR"]
        RELEASE["GitHub Release<br/>+ Changelog"]
    end

    PR --> GO & PY
    PUSH --> GO & PY
    GO --> DOCKER
    PY --> DOCKER
    DOCKER --> TRIVY & GOSEC
    TRIVY --> KIND
    GOSEC --> KIND
    KIND --> CRD_T --> E2E
    E2E --> GHCR --> RELEASE

    classDef trigger fill:#1f2937,stroke:#f59e0b,color:#fbbf24,stroke-width:2px
    classDef build fill:#1f2937,stroke:#3b82f6,color:#93c5fd,stroke-width:2px
    classDef security fill:#1f2937,stroke:#ef4444,color:#fca5a5,stroke-width:2px
    classDef validate fill:#1f2937,stroke:#8b5cf6,color:#c4b5fd,stroke-width:2px
    classDef deploy fill:#1f2937,stroke:#10b981,color:#6ee7b7,stroke-width:2px

    class PR,PUSH trigger
    class GO,PY,DOCKER build
    class TRIVY,GOSEC security
    class KIND,CRD_T,E2E validate
    class GHCR,RELEASE deploy
```

---

## CRD Schema: TrinoCluster

```mermaid
classDiagram
    class TrinoCluster {
        +apiVersion: dbqp.io/v1alpha1
        +kind: TrinoCluster
        +metadata: ObjectMeta
        +spec: TrinoClusterSpec
        +status: TrinoClusterStatus
    }

    class TrinoClusterSpec {
        +workers: int
        +coordinator: CoordinatorSpec
        +workerTemplate: WorkerSpec
        +storage: StorageSpec
        +autoscaling: AutoscalingSpec
    }

    class CoordinatorSpec {
        +memory: 8Gi
        +cpu: 4
        +config: map[string]string
    }

    class WorkerSpec {
        +memory: 8Gi
        +cpu: 4
        +nodeSelector: map
        +tolerations: []Toleration
    }

    class StorageSpec {
        +s3Endpoint: string
        +bucket: string
        +credentials: SecretRef
    }

    class AutoscalingSpec {
        +enabled: bool
        +minReplicas: 2
        +maxReplicas: 10
        +targetCPUPercent: 70
    }

    class TrinoClusterStatus {
        +phase: Running|Pending|Failed
        +readyWorkers: int
        +coordinatorReady: bool
        +lastReconciled: timestamp
        +conditions: []Condition
    }

    TrinoCluster --> TrinoClusterSpec
    TrinoCluster --> TrinoClusterStatus
    TrinoClusterSpec --> CoordinatorSpec
    TrinoClusterSpec --> WorkerSpec
    TrinoClusterSpec --> StorageSpec
    TrinoClusterSpec --> AutoscalingSpec
```

---

## Performance Benchmarks (TPC-H Scale 10)

```mermaid
xychart-beta
    title "Query Latency: Trino vs Spark (seconds, lower is better)"
    x-axis ["Q1", "Q3", "Q5", "Q6", "Q10", "Q12", "Q14", "Q19", "Q22"]
    y-axis "Latency (seconds)" 0 --> 8
    bar [4.2, 2.8, 3.5, 1.2, 3.8, 2.9, 1.8, 4.1, 1.9]
    bar [6.1, 4.5, 5.2, 2.1, 5.7, 4.3, 3.0, 5.9, 3.1]
```

| Query | Trino (s) | Spark (s) | Speedup |
|-------|-----------|-----------|---------|
| Q1 (Pricing Summary) | 4.2 | 6.1 | 1.45× |
| Q3 (Shipping Priority) | 2.8 | 4.5 | 1.61× |
| Q5 (Local Supplier Volume) | 3.5 | 5.2 | 1.49× |
| Q6 (Forecasting Revenue) | 1.2 | 2.1 | 1.75× |
| Q10 (Returned Items) | 3.8 | 5.7 | 1.50× |
| Q12 (Shipping Modes) | 2.9 | 4.3 | 1.48× |
| Q14 (Promotion Effect) | 1.8 | 3.0 | 1.67× |
| Q19 (Discounted Revenue) | 4.1 | 5.9 | 1.44× |
| Q22 (Global Sales Opp.) | 1.9 | 3.1 | 1.63× |

**Cluster Config:** 5 workers, 8GB RAM / 4 vCPU each, MinIO S3 storage, Hive Metastore

---

## Features

### Core Components

**Golang CLI Tool** — Interactive command-line interface for creating, scaling, deleting, and monitoring clusters. Runs TPC-H/TPC-DS benchmarks, streams logs, and integrates directly with the Kubernetes API.

**Kubernetes Operator** — Custom controller that watches TrinoCluster and SparkCluster CRDs, manages pod lifecycle, configures HPA for autoscaling, and handles rolling upgrades with health checks.

**Disaggregated Architecture** — Compute nodes (Trino/Spark workers) scale independently from the persistent storage layer (MinIO S3 + Hive Metastore). Compute is ephemeral and autoscaled; storage is durable and shared.

**Python Benchmarking Suite** — TPC-H and TPC-DS query execution with metrics collection, results export (JSON/CSV), and matplotlib visualization.

**CI/CD Pipeline (GitHub Actions)** — Go/Python testing and linting, Docker multi-stage builds, Trivy and GoSec security scanning, Kind cluster integration tests, and automated releases.

---

## Quick Start

### Prerequisites

- Kubernetes cluster (v1.24+) or Kind for local development
- `kubectl` configured
- Docker (for building images)
- Go 1.21+ and Python 3.9+

### Installation

```bash
# Clone and build
git clone https://github.com/JayDS22/Distributed-Big-Data-Query-Platform-on-Kubernetes.git
cd Distributed-Big-Data-Query-Platform-on-Kubernetes
go build -o dbqp .

# Setup local cluster with Kind
./setup-cluster.sh kind default

# Verify health
./health-check.sh default
```

### Usage

```bash
# Create a 5-worker Trino cluster
./dbqp create --engine trino --workers 5 --memory 8Gi --cpu 4

# Enable autoscaling (scale between 2-10 workers based on CPU)
./dbqp scale --name trino-1234567890 --min 2 --max 10 --cpu-percent 70

# Run TPC-H benchmark
./dbqp benchmark --engine trino --query tpch-q1 --scale 10 --iterations 3 --output json

# Monitor cluster
./dbqp status trino-1234567890
./dbqp logs trino-1234567890 --tail 100 --follow

# Cleanup
./dbqp delete trino-1234567890
```

### Helm Deployment

```bash
# Deploy Trino with Helm
helm install my-trino k8s/helm/trino \
  --set workers=3 \
  --set memory=8Gi \
  --set storageBucket=s3://data

# Or manual deployment
kubectl apply -f crd.yaml
kubectl apply -f trino-deployment.yaml
```

---

## Project Structure

```
dbqp/
├── main.go                   # CLI entrypoint
├── create.go                 # Cluster creation logic
├── scale.go                  # HPA configuration
├── delete.go                 # Graceful cluster teardown
├── status.go                 # Cluster health reporting
├── benchmark.go              # TPC-H/TPC-DS runner
├── logs.go                   # Log streaming
├── config.go                 # Cluster configuration
├── crd.yaml                  # TrinoCluster + SparkCluster CRDs
├── trino-deployment.yaml     # Coordinator + Worker manifests
├── benchmark_runner.py       # Python benchmarking suite
├── setup-cluster.sh          # Kind cluster provisioning
├── health-check.sh           # Diagnostic health checks
├── cleanup.sh                # Environment teardown
├── cicd.yml                  # GitHub Actions workflow
├── Dockerfile                # Multi-stage container build
└── go.mod                    # Go module dependencies
```

---

## Performance Tuning

**Memory** — Set `query.max-memory` in Trino config and adjust heap sizes per engine. Monitor utilization via `kubectl top pods`.

**CPU** — Configure task parallelism and worker thread count. Resource limits in CRD spec prevent noisy-neighbor issues.

**Storage** — Enable S3 caching, use 64MB block sizes, and apply Snappy compression for scan-heavy workloads.

**Autoscaling** — HPA targets 70% CPU utilization by default. Set `minReplicas` based on baseline traffic to avoid cold-start latency during scale-up events.

---

## Roadmap

- Web UI Dashboard (React)
- Query result caching layer
- Cost optimization recommendations
- Multi-cloud deployment (AWS EKS / GCP GKE)
- Query federation across Trino + Spark clusters
- ML-based query optimization

---

## License

MIT License — see LICENSE file
