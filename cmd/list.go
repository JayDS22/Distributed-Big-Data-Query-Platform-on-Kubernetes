package cmd

import (
	"context"
	"fmt"
	"log"
	"text/tabwriter"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all clusters",
	Long:  "Display all Trino and Spark clusters in the namespace",
	RunE: func(cmd *cobra.Command, args []string) error {
		clientset := cmd.Context().Value("clientset").(kubernetes.Interface)
		namespace := cmd.Context().Value("namespace").(string)

		log.Printf("Listing clusters in namespace '%s'\n\n", namespace)

		// Get all StatefulSets and Deployments with cluster labels
		statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list StatefulSets: %w", err)
		}

		deployments, err := clientset.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list Deployments: %w", err)
		}

		w := tabwriter.NewWriter(fmt.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "CLUSTER\tTYPE\tCOORDINATOR\tWORKERS\tSTATUS\tAGE")

		// Print StatefulSets (workers)
		for _, sts := range statefulsets.Items {
			if sts.Labels != nil && sts.Labels["type"] == "worker" {
				clusterName := sts.Labels["cluster"]
				fmt.Fprintf(w, "%s\t%s\t%s\t%d/%d\t%s\t%v\n",
					clusterName,
					"Spark/Trino",
					"1",
					sts.Status.ReadyReplicas,
					*sts.Spec.Replicas,
					statusFromReplicas(sts.Status.ReadyReplicas, *sts.Spec.Replicas),
					sts.CreationTimestamp.Time)
			}
		}

		// Print Deployments (coordinators)
		for _, deploy := range deployments.Items {
			if deploy.Labels != nil && deploy.Labels["type"] == "coordinator" {
				fmt.Fprintf(w, "%s\t%s\t%d/%d\t%s\t%s\t%v\n",
					deploy.Labels["cluster"],
					"Coordinator",
					"-",
					deploy.Status.ReadyReplicas,
					*deploy.Spec.Replicas,
					statusFromReplicas(deploy.Status.ReadyReplicas, *deploy.Spec.Replicas),
					deploy.CreationTimestamp.Time)
			}
		}

		w.Flush()

		if len(statefulsets.Items) == 0 && len(deployments.Items) == 0 {
			fmt.Println("No clusters found in namespace")
		}

		return nil
	},
}

func statusFromReplicas(ready, desired int32) string {
	if ready == desired {
		return "Ready"
	} else if ready > 0 {
		return "Partial"
	}
	return "Pending"
}

func init() {
	// No additional flags
}
