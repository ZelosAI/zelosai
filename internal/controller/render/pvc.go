package render

import (
	"fmt"

	zelosv1alpha1 "github.com/ZelosAI/zelosai/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildPVC renders a PersistentVolumeClaim for the component when persistence is enabled.
// Returns nil when persistence is disabled.
func BuildPVC(owner metav1.Object, c Component, spec *zelosv1alpha1.PersistenceSpec) *corev1.PersistentVolumeClaim {
	if spec == nil || !spec.Enabled {
		return nil
	}
	size := spec.Size
	if size == "" {
		size = "1Gi"
	}
	modes := spec.AccessModes
	if len(modes) == 0 {
		modes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", c.Name, owner.GetName()),
			Namespace: owner.GetNamespace(),
			Labels:    Labels(c, owner.GetName()),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      modes,
			StorageClassName: spec.StorageClassName,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(size),
				},
			},
		},
	}
}
