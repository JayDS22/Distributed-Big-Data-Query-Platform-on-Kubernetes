package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var force bool

var DeleteCmd = &cobra.Command{
	Use:   "delete [cluster-name]",
	Short: "Delete a Trino or Spark cluster",
	Long:  "Remove a cluster and all associated resources from Kubernetes",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		clientset := cmd.Context().Value("clientset").(kubernetes.Interface)
		namespace := cmd.Context().Value("namespace").(string)

		clusterName := args[0]

		log.Printf("Deleting cluster '%s' from namespace %s\n", clusterName, namespace)

		deleteOptions := metav1.DeleteOptions{}
		if force {
			grace := int64(0)
			deleteOptions.GracePeriodSeconds = &grace
		}

		// Delete StatefulSet (workers)
		err := clientset.AppsV1().StatefulSets(namespace).Delete(context.Background(), clusterName+"-workers", deleteOptions)
		if err != nil && !isNotFound(err) {
			log.Printf("Warning: failed to delete StatefulSet: %v\n", err)
		} else if err == nil {
			log.Printf("✓ Deleted worker StatefulSet\n")
		}

		// Delete Coordinator/Master Deployment
		err = clientset.AppsV1().Deployments(namespace).Delete(context.Background(), clusterName+"-coordinator", deleteOptions)
		if err != nil && !isNotFound(err) {
			log.Printf("Warning: failed to delete coordinator deployment: %v\n", err)
		} else if err == nil {
			log.Printf("✓ Deleted coordinator deployment\n")
		}

		// Delete Services
		err = clientset.CoreV1().Services(namespace).Delete(context.Background(), clusterName+"-service", deleteOptions)
		if err != nil && !isNotFound(err) {
			log.Printf("Warning: failed to delete service: %v\n", err)
		} else if err == nil {
			log.Printf("✓ Deleted service\n")
		}

		// Delete ConfigMaps
		err = clientset.CoreV1().ConfigMaps(namespace).Delete(context.Background(), clusterName+"-config", deleteOptions)
		if err != nil && !isNotFound(err) {
			log.Printf("Warning: failed to delete ConfigMap: %v\n", err)
		} else if err == nil {
			log.Printf("✓ Deleted ConfigMap\n")
		}

		// Delete HPA if exists
		err = clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).Delete(context.Background(), clusterName+"-hpa", deleteOptions)
		if err != nil && !isNotFound(err) {
			log.Printf("Warning: failed to delete HPA: %v\n", err)
		} else if err == nil {
			log.Printf("✓ Deleted HPA\n")
		}

		log.Printf("✓ Cluster '%s' deleted successfully\n", clusterName)

		return nil
	},
}

func isNotFound(err error) bool {
	return err != nil && err.Error() == "not found"
}

func init() {
	DeleteCmd.Flags().BoolVar(&force, "force", false, "Force delete without graceful shutdown")
}
