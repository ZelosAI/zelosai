package render

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildConfigMap renders a ConfigMap of <ENV_VAR>=<value> entries the workload
// consumes via envFrom. Keys are passed through verbatim — callers are
// expected to provide already-uppercase env-var names.
func BuildConfigMap(owner metav1.Object, c Component, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-config", c.Name, owner.GetName()),
			Namespace: owner.GetNamespace(),
			Labels:    Labels(c, owner.GetName()),
		},
		Data: data,
	}
}
