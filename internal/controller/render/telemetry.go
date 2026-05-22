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

// TelemetryCollectorName is the standard name of the operator-installed OTel collector.
const TelemetryCollectorName = "zelos-otel-collector"

// TelemetryEnvConfigMapName is the ConfigMap of OTel env vars every workload mounts via envFrom.
func TelemetryEnvConfigMapName(platformName string) string {
	return fmt.Sprintf("zelos-otel-env-%s", platformName)
}

// TelemetryEndpoint resolves the OTLP endpoint the operator should inject.
// Returns externalEndpoint when set, otherwise the in-cluster collector URL.
func TelemetryEndpoint(spec zelosv1alpha1.TelemetrySpec, namespace string) string {
	if spec.ExternalEndpoint != "" {
		return spec.ExternalEndpoint
	}
	return fmt.Sprintf("http://%s.%s.svc:4317", TelemetryCollectorName, namespace)
}

// BuildTelemetryEnvConfigMap builds the suite-wide ConfigMap with OTEL_* env vars
// every workload mounts via envFrom.
func BuildTelemetryEnvConfigMap(owner metav1.Object, spec zelosv1alpha1.TelemetrySpec) *corev1.ConfigMap {
	enabled := true
	if spec.Enabled != nil {
		enabled = *spec.Enabled
	}
	logLevel := spec.LogLevel
	if logLevel == "" {
		logLevel = "info"
	}

	data := map[string]string{
		"OTEL_LOG_LEVEL":              logLevel,
		"OTEL_RESOURCE_ATTRIBUTES":    fmt.Sprintf("deployment.environment=%s,k8s.namespace.name=%s,zelos.platform=%s", owner.GetNamespace(), owner.GetNamespace(), owner.GetName()),
		"OTEL_EXPORTER_OTLP_PROTOCOL": "grpc",
	}
	if enabled {
		data["OTEL_EXPORTER_OTLP_ENDPOINT"] = TelemetryEndpoint(spec, owner.GetNamespace())
		data["OTEL_LOGS_EXPORTER"] = "otlp"
		data["OTEL_METRICS_EXPORTER"] = "otlp"
		data["OTEL_TRACES_EXPORTER"] = "otlp"
	} else {
		data["OTEL_LOGS_EXPORTER"] = "console"
		data["OTEL_METRICS_EXPORTER"] = "none"
		data["OTEL_TRACES_EXPORTER"] = "none"
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TelemetryEnvConfigMapName(owner.GetName()),
			Namespace: owner.GetNamespace(),
			Labels: map[string]string{
				"app.kubernetes.io/name":       "zelos-otel-env",
				"app.kubernetes.io/managed-by": "zelosai",
				"app.kubernetes.io/part-of":    "zelos",
			},
		},
		Data: data,
	}
}

// BuildCollectorConfigMap renders the OTel collector pipeline config.
func BuildCollectorConfigMap(owner metav1.Object) *corev1.ConfigMap {
	const collectorConfig = `receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318
processors:
  batch: {}
  memory_limiter:
    check_interval: 1s
    limit_percentage: 80
    spike_limit_percentage: 25
exporters:
  debug:
    verbosity: basic
service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [debug]
    metrics:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [debug]
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [debug]
`
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TelemetryCollectorName + "-config",
			Namespace: owner.GetNamespace(),
			Labels: map[string]string{
				"app.kubernetes.io/name":       TelemetryCollectorName,
				"app.kubernetes.io/instance":   owner.GetName(),
				"app.kubernetes.io/managed-by": "zelosai",
				"app.kubernetes.io/part-of":    "zelos",
			},
		},
		Data: map[string]string{"config.yaml": collectorConfig},
	}
}

// BuildCollectorDeployment renders the OTel collector Deployment.
func BuildCollectorDeployment(owner metav1.Object, spec zelosv1alpha1.TelemetrySpec) *appsv1.Deployment {
	labels := map[string]string{
		"app.kubernetes.io/name":       TelemetryCollectorName,
		"app.kubernetes.io/instance":   owner.GetName(),
		"app.kubernetes.io/managed-by": "zelosai",
		"app.kubernetes.io/part-of":    "zelos",
	}
	selector := map[string]string{
		"app.kubernetes.io/name":     TelemetryCollectorName,
		"app.kubernetes.io/instance": owner.GetName(),
	}

	image := "otel/opentelemetry-collector-contrib"
	tag := "0.110.0"
	if spec.Image.Repository != "" {
		image = spec.Image.Repository
	}
	if spec.Image.Tag != "" {
		tag = spec.Image.Tag
	}

	replicas := int32(1)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TelemetryCollectorName,
			Namespace: owner.GetNamespace(),
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: selector},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "otel-collector",
						Image: fmt.Sprintf("%s:%s", image, tag),
						Args:  []string{"--config=/etc/otelcol/config.yaml"},
						Ports: []corev1.ContainerPort{
							{Name: "otlp-grpc", ContainerPort: 4317, Protocol: corev1.ProtocolTCP},
							{Name: "otlp-http", ContainerPort: 4318, Protocol: corev1.ProtocolTCP},
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("50m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("512Mi"),
							},
						},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "config",
							MountPath: "/etc/otelcol",
							ReadOnly:  true,
						}},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromString("otlp-grpc")},
							},
							InitialDelaySeconds: 5,
							PeriodSeconds:       10,
						},
					}},
					Volumes: []corev1.Volume{{
						Name: "config",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: TelemetryCollectorName + "-config"},
							},
						},
					}},
				},
			},
		},
	}
}

// BuildCollectorService renders the ClusterIP Service in front of the collector.
func BuildCollectorService(owner metav1.Object) *corev1.Service {
	selector := map[string]string{
		"app.kubernetes.io/name":     TelemetryCollectorName,
		"app.kubernetes.io/instance": owner.GetName(),
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TelemetryCollectorName,
			Namespace: owner.GetNamespace(),
			Labels: map[string]string{
				"app.kubernetes.io/name":       TelemetryCollectorName,
				"app.kubernetes.io/managed-by": "zelosai",
				"app.kubernetes.io/part-of":    "zelos",
			},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: selector,
			Ports: []corev1.ServicePort{
				{Name: "otlp-grpc", Port: 4317, TargetPort: intstr.FromString("otlp-grpc"), Protocol: corev1.ProtocolTCP},
				{Name: "otlp-http", Port: 4318, TargetPort: intstr.FromString("otlp-http"), Protocol: corev1.ProtocolTCP},
			},
		},
	}
}
