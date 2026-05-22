package render

import (
	"fmt"

	zelosv1alpha1 "github.com/ZelosAI/zelosai/api/v1alpha1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildHPA renders a HorizontalPodAutoscaler targeting the rendered Deployment.
// Returns nil when autoscaling is disabled.
func BuildHPA(owner metav1.Object, c Component, spec *zelosv1alpha1.AutoscalingSpec) *autoscalingv2.HorizontalPodAutoscaler {
	if spec == nil {
		return nil
	}
	target := int32(70)
	if spec.TargetCPUUtilization != nil {
		target = *spec.TargetCPUUtilization
	}
	depName := fmt.Sprintf("%s-%s", c.Name, owner.GetName())
	return &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      depName,
			Namespace: owner.GetNamespace(),
			Labels:    Labels(c, owner.GetName()),
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       depName,
			},
			MinReplicas: &spec.MinReplicas,
			MaxReplicas: spec.MaxReplicas,
			Metrics: []autoscalingv2.MetricSpec{{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: "cpu",
					Target: autoscalingv2.MetricTarget{
						Type:               autoscalingv2.UtilizationMetricType,
						AverageUtilization: &target,
					},
				},
			}},
		},
	}
}
