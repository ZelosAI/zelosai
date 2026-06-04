package render

import (
	"fmt"
	"path/filepath"
	"sort"

	zelosv1alpha1 "github.com/ZelosAI/zelosai/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// DeploymentInput is everything needed to build a Deployment for a Zelos component.
type DeploymentInput struct {
	Owner       metav1.Object
	Component   Component
	Workload    zelosv1alpha1.WorkloadSpec
	ConfigMap   string // name of the operator-rendered ConfigMap with envFrom config
	TelemetryCM string // name of the suite-wide OTel env ConfigMap
	PullSecret  string // standard ghcr pull secret name
}

// BuildDeployment renders the Deployment honoring the standard container contract.
func BuildDeployment(in DeploymentInput) *appsv1.Deployment {
	labels := Labels(in.Component, in.Owner.GetName())
	selector := SelectorLabels(in.Component, in.Owner.GetName())

	replicas := int32(1)
	if in.Workload.Replicas != nil {
		replicas = *in.Workload.Replicas
	}

	resources := in.Workload.Resources
	if resources.Requests == nil {
		resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		}
	}

	envFrom := []corev1.EnvFromSource{}
	if in.TelemetryCM != "" {
		envFrom = append(envFrom, corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: in.TelemetryCM},
			},
		})
	}
	if in.ConfigMap != "" {
		envFrom = append(envFrom, corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: in.ConfigMap},
			},
		})
	}

	// Standard pod-info downward env (used as resource attributes by OTel SDK).
	env := []corev1.EnvVar{
		{Name: "OTEL_SERVICE_NAME", Value: in.Component.Name},
		{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
		{Name: "POD_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
		{Name: fmt.Sprintf("%s_PORT", in.Component.EnvPrefix), Value: fmt.Sprintf("%d", in.Component.Port)},
		{Name: fmt.Sprintf("%s_STATE_DIR", in.Component.EnvPrefix), Value: in.Component.PersistentStateDir},
	}

	// Secret file mounts → env vars carrying their paths.
	volumes, volumeMounts, secretEnvs := SecretVolumes(in.Component, in.Workload.SecretRefs)
	env = append(env, secretEnvs...)

	// Persistence volume mount.
	if in.Workload.Persistence != nil && in.Workload.Persistence.Enabled {
		volumes = append(volumes, corev1.Volume{
			Name: "state",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: fmt.Sprintf("%s-%s", in.Component.Name, in.Owner.GetName()),
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "state",
			MountPath: in.Component.PersistentStateDir,
		})
	}

	image := in.Component.DefaultImageRepo
	if in.Workload.Image.Repository != "" {
		image = in.Workload.Image.Repository
	}
	tag := "develop"
	if in.Workload.Image.Tag != "" {
		tag = in.Workload.Image.Tag
	}
	pullPolicy := corev1.PullIfNotPresent
	if in.Workload.Image.PullPolicy != "" {
		pullPolicy = in.Workload.Image.PullPolicy
	}

	pullSecrets := append([]corev1.LocalObjectReference{}, in.Workload.ImagePullSecrets...)
	if in.PullSecret != "" {
		pullSecrets = append(pullSecrets, corev1.LocalObjectReference{Name: in.PullSecret})
	}

	sort.SliceStable(env, func(i, j int) bool { return env[i].Name < env[j].Name })

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", in.Component.Name, in.Owner.GetName()),
			Namespace: in.Owner.GetNamespace(),
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: selector},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					ServiceAccountName: fmt.Sprintf("zelos-%s", in.Component.Name),
					ImagePullSecrets:   pullSecrets,
					NodeSelector:       in.Workload.NodeSelector,
					Tolerations:        in.Workload.Tolerations,
					Volumes:            volumes,
					Containers: []corev1.Container{{
						Name:            in.Component.Name,
						Image:           ImageRef(image, tag),
						ImagePullPolicy: pullPolicy,
						Ports: []corev1.ContainerPort{
							{Name: "http", ContainerPort: in.Component.Port, Protocol: corev1.ProtocolTCP},
						},
						Env:          env,
						EnvFrom:      envFrom,
						VolumeMounts: volumeMounts,
						Resources:    resources,
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/healthz",
									Port: intstr.FromString("http"),
								},
							},
							InitialDelaySeconds: 5,
							PeriodSeconds:       10,
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/readyz",
									Port: intstr.FromString("http"),
								},
							},
							InitialDelaySeconds: 3,
							PeriodSeconds:       5,
						},
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: boolPtr(false),
							ReadOnlyRootFilesystem:   boolPtr(true),
							RunAsNonRoot:             boolPtr(true),
							Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
						},
					}},
				},
			},
		},
	}
	return dep
}

// JoinPath is a tiny helper for tests; not used by builders directly.
func JoinPath(parts ...string) string { return filepath.Join(parts...) }

func boolPtr(b bool) *bool { return &b }
