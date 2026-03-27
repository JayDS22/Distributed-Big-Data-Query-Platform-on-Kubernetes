package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	logPod       string
	logTail      int64
	logFollow    bool
	logTimestamp bool
)

var LogsCmd = &cobra.Command{
	Use:   "logs [cluster-name]",
	Short: "Fetch logs from cluster pods",
	Long:  "Stream or retrieve logs from coordinator/worker pods for debugging",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		clientset := cmd.Context().Value("clientset").(kubernetes.Interface)
		namespace := cmd.Context().Value("namespace").(string)

		clusterName := args[0]

		log.Printf("Fetching logs for cluster '%s'\n", clusterName)

		return fetchClusterLogs(clientset, namespace, clusterName)
	},
}

func fetchClusterLogs(clientset kubernetes.Interface, namespace, clusterName string) error {
	// List all pods for this cluster
	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: "cluster=" + clusterName,
	})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return fmt.Errorf("no pods found for cluster '%s'", clusterName)
	}

	log.Printf("Found %d pods\n\n", len(pods.Items))

	// Get logs from each pod
	for _, pod := range pods.Items {
		fmt.Printf("\n=== Pod: %s ===\n", pod.Name)

		logOpts := &corev1.PodLogOptions{
			TailLines:  &logTail,
			Timestamps: logTimestamp,
		}

		req := clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, logOpts)
		podLogs, err := req.Stream(context.Background())
		if err != nil {
			log.Printf("Warning: failed to get logs from %s: %v\n", pod.Name, err)
			continue
		}
		defer podLogs.Close()

		_, err = io.Copy(os.Stdout, podLogs)
		if err != nil {
			log.Printf("Warning: failed to read logs: %v\n", err)
		}
	}

	return nil
}

func init() {
	LogsCmd.Flags().StringVar(&logPod, "pod", "", "Specific pod name (optional)")
	LogsCmd.Flags().Int64Var(&logTail, "tail", 100, "Number of lines to retrieve")
	LogsCmd.Flags().BoolVar(&logFollow, "follow", false, "Follow logs (stream)")
	LogsCmd.Flags().BoolVar(&logTimestamp, "timestamps", true, "Include timestamps in logs")
}
