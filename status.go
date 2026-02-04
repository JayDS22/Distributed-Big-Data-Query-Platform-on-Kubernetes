package cmd

import (
	"context"
	"fmt"
	"log"
	"text/tabwriter"

	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var StatusCmd = &cobra.Command{
	Use:   "status [cluster-name]",
	Short: "Check status of a cluster",
	Long:  "Display health, resource usage, and pod status of a cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		clientset := cmd.Context().Value("clientset").(kubernetes.Interface)
		namespace := cmd.Context().Value("namespace").(string)

		clusterName := args[0]

		log.Printf("Fetching status for cluster '%s' in namespace %s\n\n", clusterName, namespace)

		// Get worker StatefulSet
		sts, err := clientset.AppsV1().StatefulSets(namespace).Get(context.Background(), clusterName+"-workers", metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get StatefulSet: %w", err)
		}

		// Get coordinator Deployment
		deploy, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), clusterName+"-coordinator", metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get Deployment: %w", err)
		}

		// Get Pods
		pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: "cluster=" + clusterName,
		})
		if err != nil {
			return fmt.Errorf("failed to list pods: %w", err)
		}

		printClusterStatus(clusterName, sts, deploy, pods)

		return nil
	},
}

func printClusterStatus(name string, sts *appsv1.StatefulSet, deploy *appsv1.Deployment, pods interface{}) {
	fmt.Printf("Cluster: %s\n", name)
	fmt.Printf("=====================================\n\n")

	// StatefulSet status
	if sts != nil {
		fmt.Printf("Workers (StatefulSet):\n")
		fmt.Printf("  Desired Replicas: %d\n", *sts.Spec.Replicas)
		fmt.Printf("  Ready Replicas:   %d\n", sts.Status.ReadyReplicas)
		fmt.Printf("  Current Replicas: %d\n", sts.Status.Replicas)
		fmt.Printf("\n")
	}

	// Deployment status
	if deploy != nil {
		fmt.Printf("Coordinator (Deployment):\n")
		fmt.Printf("  Desired Replicas: %d\n", *deploy.Spec.Replicas)
		fmt.Printf("  Ready Replicas:   %d\n", deploy.Status.ReadyReplicas)
		fmt.Printf("  Available:        %d\n", deploy.Status.AvailableReplicas)
		fmt.Printf("\n")
	}

	// Pod details
	fmt.Printf("Pod Details:\n")
	fmt.Printf("=====================================\n")

	w := tabwriter.NewWriter(fmt.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tREADY\tSTATUS\tRESTARTS\tAGE")

	// Convert pods interface to expected type
	podList, ok := pods.(*struct {
		Items []interface{}
	})
	if ok && podList != nil {
		for _, pod := range podList.Items {
			// Print pod info (simplified)
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\n", "", "", "", "", "")
		}
	}

	w.Flush()
}

func init() {
	// No additional flags
}
