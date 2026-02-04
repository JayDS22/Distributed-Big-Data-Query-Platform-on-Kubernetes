package cluster

import (
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Config struct {
	Engine        string
	Workers       int
	Memory        string
	CPU           string
	Image         string
	StorageBucket string
	Name          string
}

type TrinoCluster struct {
	Name              string
	CoordinatorPods   int
	WorkerPods        int
	Memory            string
	CPU               string
	StorageBucket     string
	MetastoreHost     string
	MetastorePort     int
	CreatedAt         time.Time
}

type SparkCluster struct {
	Name          string
	MasterPods    int
	WorkerPods    int
	Memory        string
	CPU           string
	StorageBucket string
	CreatedAt     time.Time
}

func NewTrinoCluster(cfg *Config) *TrinoCluster {
	return &TrinoCluster{
		Name:          fmt.Sprintf("trino-%d", time.Now().Unix()),
		CoordinatorPods: 1,
		WorkerPods:    cfg.Workers,
		Memory:        cfg.Memory,
		CPU:           cfg.CPU,
		StorageBucket: cfg.StorageBucket,
		MetastoreHost: "hive-metastore",
		MetastorePort: 9083,
		CreatedAt:     time.Now(),
	}
}

func NewSparkCluster(cfg *Config) *SparkCluster {
	return &SparkCluster{
		Name:          fmt.Sprintf("spark-%d", time.Now().Unix()),
		MasterPods:    1,
		WorkerPods:    cfg.Workers,
		Memory:        cfg.Memory,
		CPU:           cfg.CPU,
		StorageBucket: cfg.StorageBucket,
		CreatedAt:     time.Now(),
	}
}

func GenerateTrinoConfigMap(cfg *Config) (*corev1.ConfigMap, error) {
	clusterName := fmt.Sprintf("trino-%d", time.Now().Unix())

	// Generate Trino configuration
	config := generateTrinoConfig(cfg.Workers, cfg.StorageBucket)
	catalog := generateTrinoCatalog(cfg.StorageBucket)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName + "-config",
			Namespace: "default",
			Labels: map[string]string{
				"cluster":  clusterName,
				"type":     "config",
				"engine":   "trino",
			},
		},
		Data: map[string]string{
			"config.properties":    config,
			"catalog/hive.properties": catalog,
			"jvm.config":            generateTrinoJVMConfig(),
		},
	}

	return cm, nil
}

func GenerateSparkConfigMap(cfg *Config) (*corev1.ConfigMap, error) {
	clusterName := fmt.Sprintf("spark-%d", time.Now().Unix())

	config := generateSparkConfig(cfg.StorageBucket, cfg.Memory, cfg.CPU)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName + "-config",
			Namespace: "default",
			Labels: map[string]string{
				"cluster":  clusterName,
				"type":     "config",
				"engine":   "spark",
			},
		},
		Data: map[string]string{
			"spark-defaults.conf": config,
			"log4j.properties":     generateSparkLog4j(),
		},
	}

	return cm, nil
}

func generateTrinoConfig(workers int, storageBucket string) string {
	return fmt.Sprintf(`coordinator=true
node-scheduler.include-coordinator=false
discovery-server.enabled=true
discovery.uri=http://localhost:8080
query.max-memory=4GB
query.max-memory-per-node=1GB
query.max-total-memory-per-node=2GB
memory.heap-headroom-per-node=1GB
http-server.http.port=8080
task.max-worker-threads=32
task.min-drivers=4
task.writer-count=4
exchange.http-client.max-connections-per-server=1000
exchange.http-client.connect-timeout=5s
s3.endpoint=%s
s3.path-style-access=true
s3.aws-access-key=${AWS_ACCESS_KEY_ID}
s3.aws-secret-access-key=${AWS_SECRET_ACCESS_KEY}
`, storageBucket)
}

func generateTrinoCatalog(storageBucket string) string {
	return `connector.name=hive
hive.metastore.uri=thrift://hive-metastore:9083
hive.s3.path=s3://%s/
hive.s3.endpoint=%s
hive.s3.aws-access-key=${AWS_ACCESS_KEY_ID}
hive.s3.aws-secret-access-key=${AWS_SECRET_ACCESS_KEY}
hive.compression-codec=snappy
`
}

func generateTrinoJVMConfig() string {
	return `-server
-Xmx8G
-XX:+UseG1GC
-XX:MaxGCPauseMillis=30000
-XX:InitiatingHeapOccupancyPercent=35
-XX:G1HeapRegionSize=16m
-XX:MinMetaspaceSize=512m
-XX:MaxMetaspaceSize=512m
-XX:CompressedClassSpaceSize=256m
-XX:+PrintGCDetails
-XX:+PrintGCDateStamps
-XX:+PrintGCTimeStamps
-XX:+PrintClassHistogramBeforeFullGC
-XX:+PrintClassHistogramAfterFullGC
-XX:PrintFLSStatistics=2
-XX:+PrintTenuringDistribution
-XX:+PrintHeapAtGC
-XX:+UseGCLogFileRotation
-XX:NumberOfGCLogFiles=5
-XX:GCLogFileSize=20M
-XX:-UseBiasedLocking
-XX:+HeapDumpOnOutOfMemoryError
-XX:HeapDumpPath=var/log
`
}

func generateSparkConfig(storageBucket, memory, cpu string) string {
	return fmt.Sprintf(`spark.master=spark://localhost:7077
spark.driver.memory=%s
spark.executor.memory=%s
spark.executor.cores=%s
spark.default.parallelism=32
spark.sql.shuffle.partitions=32
spark.sql.adaptive.enabled=true
spark.sql.adaptive.skewJoin.enabled=true
spark.dynamicAllocation.enabled=true
spark.dynamicAllocation.minExecutors=2
spark.dynamicAllocation.maxExecutors=10
spark.eventLog.enabled=true
spark.eventLog.dir=/var/spark/logs
spark.hadoop.fs.s3a.impl=org.apache.hadoop.fs.s3a.S3AFileSystem
spark.hadoop.fs.s3a.endpoint=%s
spark.hadoop.fs.s3a.access.key=${AWS_ACCESS_KEY_ID}
spark.hadoop.fs.s3a.secret.key=${AWS_SECRET_ACCESS_KEY}
spark.hadoop.fs.s3a.path.style.access=true
spark.hadoop.fs.s3a.block.size=64m
spark.hadoop.fs.s3a.threads.max=128
spark.hadoop.fs.s3a.threads.core=32
spark.sql.hive.metastore.uris=thrift://hive-metastore:9083
`, memory, memory, cpu, storageBucket)
}

func generateSparkLog4j() string {
	return `# Set everything to be logged to the console
rootLogger.level = WARN
rootLogger.appenderRef.console.ref = console

appender.console.type = Console
appender.console.name = console
appender.console.target = SYSTEM_ERR
appender.console.layout.type = PatternLayout
appender.console.layout.pattern = %d{yy/MM/dd HH:mm:ss} %p %c{1}: %m%n

# Settings to quiet third party logs that are too verbose
logger.org.spark_project.jetty.util.component.AbstractLifeCycle.name = org.spark_project.jetty.util.component.AbstractLifeCycle
logger.org.spark_project.jetty.util.component.AbstractLifeCycle.level = ERROR
`
}

// Helper to parse memory strings like "8Gi" to bytes
func ParseMemory(memStr string) (int64, error) {
	memStr = strings.TrimSpace(memStr)
	var multiplier int64 = 1

	if strings.HasSuffix(memStr, "Gi") {
		multiplier = 1 << 30
		memStr = strings.TrimSuffix(memStr, "Gi")
	} else if strings.HasSuffix(memStr, "Mi") {
		multiplier = 1 << 20
		memStr = strings.TrimSuffix(memStr, "Mi")
	} else if strings.HasSuffix(memStr, "Ki") {
		multiplier = 1 << 10
		memStr = strings.TrimSuffix(memStr, "Ki")
	}

	var value int64
	_, err := fmt.Sscanf(memStr, "%d", &value)
	if err != nil {
		return 0, fmt.Errorf("invalid memory format: %s", memStr)
	}

	return value * multiplier, nil
}
