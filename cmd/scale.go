package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	clusterName string
	minReplicas int32
	maxReplicas int32
	cpuPercent  int32
)

var ScaleCmd = &cobra.Command{
	Use:   "scale",
	Short: "Configure autoscaling for a cluster",
	Long:  "Set up Horizontal Pod Autoscaler (HPA) to automatically scale workers based on CPU/memory",
	RunE: func(cmd *cobra.Command, args []string) error {
		clientset := cmd.Context().Value("clientset").(kubernetes.Interface)
		namespace := cmd.Context().Value("namespace").(string)

		if clusterName == "" {
			return fmt.Errorf("--name flag is required")
		}

		log.Printf("Configuring autoscaling for cluster '%s' in namespace %s\n", clusterName, namespace)
		log.Printf("  Min replicas: %d\n", minReplicas)
		log.Printf("  Max replicas: %d\n", maxReplicas)
		log.Printf("  Target CPU: %d%%\n", cpuPercent)

		hpa := &autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName + "-hpa",
				Namespace: namespace,
			},
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
					APIVersion: "apps/v1",
					Kind:       "StatefulSet",
					Name:       clusterName + "-workers",
				},
				MinReplicas: &minReplicas,
				MaxReplicas: maxReplicas,
				Metrics: []autoscalingv2.MetricSpec{
					{
						Type: autoscalingv2.ResourceMetricSourceType,
						Resource: &autoscalingv2.ResourceMetricSource{
							Name: corev1.ResourceCPU,
							Target: autoscalingv2.MetricTarget{
								Type:               autoscalingv2.UtilizationMetricType,
								AverageUtilization: &cpuPercent,
							},
						},
					},
				},
			},
		}

		_, err := clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).Create(context.Background(), hpa, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create HPA: %w", err)
		}

		log.Printf("✓ Autoscaling enabled for cluster '%s'\n", clusterName)
		log.Printf("  HPA will maintain CPU utilization at ~%d%%\n", cpuPercent)

		return nil
	},
}

func init() {
	ScaleCmd.Flags().StringVar(&clusterName, "name", "", "Cluster name to scale")
	ScaleCmd.Flags().Int32Var(&minReplicas, "min", 2, "Minimum number of replicas")
	ScaleCmd.Flags().Int32Var(&maxReplicas, "max", 10, "Maximum number of replicas")
	ScaleCmd.Flags().Int32Var(&cpuPercent, "cpu-percent", 70, "Target CPU utilization percentage")

	ScaleCmd.MarkFlagRequired("name")
}
