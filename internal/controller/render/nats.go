package render

import (
	"fmt"

	zelosv1alpha1 "github.com/ZelosAI/zelosai/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// NATSName returns the StatefulSet/Service name for the operator-installed NATS substrate.
func NATSName(backplaneName string) string { return fmt.Sprintf("zelos-backplane-nats-%s", backplaneName) }

// NATSURL is the cluster-internal URL workloads should use to reach NATS.
func NATSURL(backplaneName, namespace string) string {
	return fmt.Sprintf("nats://%s.%s.svc:4222", NATSName(backplaneName), namespace)
}

// BuildNATSStatefulSet renders a single-replica NATS server with JetStream enabled.
func BuildNATSStatefulSet(owner metav1.Object, spec zelosv1alpha1.ZelosBackplaneSpec) *appsv1.StatefulSet {
	labels := map[string]string{
		"app.kubernetes.io/name":       "nats",
		"app.kubernetes.io/instance":   owner.GetName(),
		"app.kubernetes.io/managed-by": "zelosai",
		"app.kubernetes.io/part-of":    "zelos",
		"zelos.zelosai.io/component":   "zelosbackplane",
	}
	selector := map[string]string{
		"app.kubernetes.io/name":     "nats",
		"app.kubernetes.io/instance": owner.GetName(),
	}
	replicas := int32(1)
	if spec.Replicas != nil {
		replicas = *spec.Replicas
	}

	size := "1Gi"
	if spec.Persistence != nil && spec.Persistence.Size != "" {
		size = spec.Persistence.Size
	}
	var storageClass *string
	if spec.Persistence != nil {
		storageClass = spec.Persistence.StorageClassName
	}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NATSName(owner.GetName()),
			Namespace: owner.GetNamespace(),
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: NATSName(owner.GetName()),
			Replicas:    &replicas,
			Selector:    &metav1.LabelSelector{MatchLabels: selector},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "nats",
						Image: "nats:2.10-alpine",
						Args:  []string{"-js", "-sd", "/data"},
						Ports: []corev1.ContainerPort{
							{Name: "client", ContainerPort: 4222, Protocol: corev1.ProtocolTCP},
							{Name: "monitor", ContainerPort: 8222, Protocol: corev1.ProtocolTCP},
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
						VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/data"}},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{Path: "/healthz", Port: intstr.FromString("monitor")},
							},
							InitialDelaySeconds: 10,
							PeriodSeconds:       10,
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{Path: "/healthz?js-enabled=true", Port: intstr.FromString("monitor")},
							},
							InitialDelaySeconds: 5,
							PeriodSeconds:       5,
						},
					}},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
				ObjectMeta: metav1.ObjectMeta{Name: "data"},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					StorageClassName: storageClass,
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(size)},
					},
				},
			}},
		},
	}
}

// BuildNATSService renders the headless Service that fronts the NATS StatefulSet.
func BuildNATSService(owner metav1.Object) *corev1.Service {
	selector := map[string]string{
		"app.kubernetes.io/name":     "nats",
		"app.kubernetes.io/instance": owner.GetName(),
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NATSName(owner.GetName()),
			Namespace: owner.GetNamespace(),
			Labels: map[string]string{
				"app.kubernetes.io/name":       "nats",
				"app.kubernetes.io/instance":   owner.GetName(),
				"app.kubernetes.io/managed-by": "zelosai",
				"app.kubernetes.io/part-of":    "zelos",
			},
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: corev1.ClusterIPNone,
			Selector:  selector,
			Ports: []corev1.ServicePort{
				{Name: "client", Port: 4222, TargetPort: intstr.FromString("client"), Protocol: corev1.ProtocolTCP},
				{Name: "monitor", Port: 8222, TargetPort: intstr.FromString("monitor"), Protocol: corev1.ProtocolTCP},
			},
		},
	}
}
