package render

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// BuildService renders the ClusterIP Service exposing the component's HTTP port.
func BuildService(owner metav1.Object, c Component) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", c.Name, owner.GetName()),
			Namespace: owner.GetNamespace(),
			Labels:    Labels(c, owner.GetName()),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: SelectorLabels(c, owner.GetName()),
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Port:       c.Port,
				TargetPort: intstr.FromString("http"),
				Protocol:   corev1.ProtocolTCP,
			}},
		},
	}
}
