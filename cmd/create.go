package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"dbqp/pkg/cluster"
)

var (
	engine   string
	workers  int
	memory   string
	cpu      string
	image    string
	storageBucket string
)

var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Trino or Spark cluster",
	Long:  "Create a new distributed query engine cluster on Kubernetes with specified configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get Kubernetes client from context
		clientset := cmd.Context().Value("clientset").(kubernetes.Interface)
		namespace := cmd.Context().Value("namespace").(string)

		if engine == "" {
			return fmt.Errorf("--engine flag is required (trino or spark)")
		}

		if engine != "trino" && engine != "spark" {
			return fmt.Errorf("invalid engine: %s (must be 'trino' or 'spark')", engine)
		}

		log.Printf("Creating %s cluster in namespace %s\n", engine, namespace)

		clusterConfig := &cluster.Config{
			Engine:        engine,
			Workers:       workers,
			Memory:        memory,
			CPU:           cpu,
			Image:         image,
			StorageBucket: storageBucket,
		}

		if engine == "trino" {
			return createTrinoCluster(cmd.Context(), clientset, namespace, clusterConfig)
		} else {
			return createSparkCluster(cmd.Context(), clientset, namespace, clusterConfig)
		}
	},
}

func createTrinoCluster(ctx context.Context, clientset kubernetes.Interface, namespace string, cfg *cluster.Config) error {
	// Create CRD instance
	trinoCluster := cluster.NewTrinoCluster(cfg)

	log.Printf("Deploying Trino cluster with %d workers\n", cfg.Workers)
	log.Printf("  Memory per worker: %s\n", cfg.Memory)
	log.Printf("  CPU per worker: %s\n", cfg.CPU)
	log.Printf("  Storage bucket: %s\n", cfg.StorageBucket)

	// Here we would create the actual CRD resource via the Operator
	// For now, we create ConfigMaps and Services that the Operator will manage
	cm, err := cluster.GenerateTrinoConfigMap(cfg)
	if err != nil {
		return fmt.Errorf("failed to generate Trino config: %w", err)
	}

	_, err = clientset.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create ConfigMap: %w", err)
	}

	log.Printf("✓ Trino cluster '%s' created successfully\n", trinoCluster.Name)
	log.Printf("  Access coordinator at: %s.%s.svc.cluster.local:8080\n", 
		trinoCluster.Name+"-coordinator", namespace)

	return nil
}

func createSparkCluster(ctx context.Context, clientset kubernetes.Interface, namespace string, cfg *cluster.Config) error {
	sparkCluster := cluster.NewSparkCluster(cfg)

	log.Printf("Deploying Spark cluster with %d workers\n", cfg.Workers)
	log.Printf("  Memory per worker: %s\n", cfg.Memory)
	log.Printf("  CPU per worker: %s\n", cfg.CPU)

	cm, err := cluster.GenerateSparkConfigMap(cfg)
	if err != nil {
		return fmt.Errorf("failed to generate Spark config: %w", err)
	}

	_, err = clientset.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create ConfigMap: %w", err)
	}

	log.Printf("✓ Spark cluster '%s' created successfully\n", sparkCluster.Name)
	log.Printf("  Access master at: %s.%s.svc.cluster.local:7077\n",
		sparkCluster.Name+"-master", namespace)

	return nil
}

func init() {
	CreateCmd.Flags().StringVar(&engine, "engine", "", "Query engine to deploy (trino or spark)")
	CreateCmd.Flags().IntVar(&workers, "workers", 3, "Number of worker nodes")
	CreateCmd.Flags().StringVar(&memory, "memory", "8Gi", "Memory per worker node")
	CreateCmd.Flags().StringVar(&cpu, "cpu", "4", "CPU cores per worker node")
	CreateCmd.Flags().StringVar(&image, "image", "", "Custom container image (optional)")
	CreateCmd.Flags().StringVar(&storageBucket, "storage-bucket", "s3://data", "S3/MinIO bucket URI")

	CreateCmd.MarkFlagRequired("engine")
}
