package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"dbqp/cmd"
	"dbqp/pkg/config"
	"dbqp/pkg/k8s"
)

var (
	kubeconfig string
	namespace  string
)

var rootCmd = &cobra.Command{
	Use:   "dbqp",
	Short: "Distributed Big Data Query Platform CLI",
	Long: `DBQP is a command-line tool for managing Trino and Spark clusters on Kubernetes.
It provides commands to create, scale, delete, monitor, and benchmark distributed query engines.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize Kubernetes client and store in context
		cfg, err := loadKubeConfig(kubeconfig)
		if err != nil {
			return fmt.Errorf("failed to load kubeconfig: %w", err)
		}

		clientset, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return fmt.Errorf("failed to create Kubernetes clientset: %w", err)
		}

		// Store in context for child commands
		ctx := context.Background()
		ctx = context.WithValue(ctx, "clientset", clientset)
		ctx = context.WithValue(ctx, "namespace", namespace)
		ctx = context.WithValue(ctx, "kubeconfig", kubeconfig)

		return nil
	},
}

func loadKubeConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig file (defaults to in-cluster config)")
	rootCmd.PersistentFlags().StringVar(&namespace, "namespace", "default", "Kubernetes namespace for deployments")

	// Add subcommands
	rootCmd.AddCommand(cmd.CreateCmd)
	rootCmd.AddCommand(cmd.ScaleCmd)
	rootCmd.AddCommand(cmd.DeleteCmd)
	rootCmd.AddCommand(cmd.StatusCmd)
	rootCmd.AddCommand(cmd.BenchmarkCmd)
	rootCmd.AddCommand(cmd.LogsCmd)
	rootCmd.AddCommand(cmd.ListCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Command failed: %v", err)
		os.Exit(1)
	}
}
