package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ImageSpec describes a container image to pull. Repository defaults to
// ghcr.io/zelosai/<component> when empty. See docs/architecture/10-image-registry.md.
type ImageSpec struct {
	// Repository is the full image repository, e.g. ghcr.io/zelosai/zelosgateway.
	// +optional
	Repository string `json:"repository,omitempty"`

	// Tag is the image tag. Defaults to the operator's component default.
	// +optional
	Tag string `json:"tag,omitempty"`

	// PullPolicy is the imagePullPolicy applied to the rendered container.
	// +kubebuilder:validation:Enum=Always;IfNotPresent;Never
	// +optional
	PullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`
}

// SecretMount projects a single key from an existing Secret into the pod
// at /etc/zelos/secrets/<key> by default, and exposes the file path via an
// env var (default ZELOS<COMPONENT>_<KEY>_FILE).
type SecretMount struct {
	// Name is the Secret name in the same namespace as the CR.
	Name string `json:"name"`

	// Key is the data key inside the Secret.
	Key string `json:"key"`

	// Path overrides the default file mount path (/etc/zelos/secrets/<key>).
	// +optional
	Path string `json:"path,omitempty"`

	// Env overrides the env var name used to expose the file path.
	// +optional
	Env string `json:"env,omitempty"`
}

// PersistenceSpec describes a PVC the operator should create for the workload.
type PersistenceSpec struct {
	// Enabled toggles PVC creation. Defaults to false when omitted.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Size is the requested storage capacity (e.g. "1Gi").
	// +kubebuilder:default="1Gi"
	// +optional
	Size string `json:"size,omitempty"`

	// StorageClassName overrides the cluster default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// AccessModes defaults to [ReadWriteOnce].
	// +optional
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`
}

// AutoscalingSpec is rendered to a HorizontalPodAutoscaler.
type AutoscalingSpec struct {
	// +kubebuilder:validation:Minimum=1
	MinReplicas int32 `json:"minReplicas"`
	// +kubebuilder:validation:Minimum=1
	MaxReplicas int32 `json:"maxReplicas"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +optional
	TargetCPUUtilization *int32 `json:"targetCPUUtilization,omitempty"`
}

// ServiceSpec controls the Service the operator creates for the workload.
type ServiceSpec struct {
	// +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer
	// +kubebuilder:default=ClusterIP
	// +optional
	Type corev1.ServiceType `json:"type,omitempty"`

	// +optional
	Port int32 `json:"port,omitempty"`
}

// WorkloadSpec is the common envelope embedded in every component CRD.
type WorkloadSpec struct {
	// +optional
	Image ImageSpec `json:"image,omitempty"`

	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Config keys are rendered to a ConfigMap mounted as envFrom.
	// +optional
	Config map[string]string `json:"config,omitempty"`

	// SecretRefs project Secret keys as files under /etc/zelos/secrets/.
	// +optional
	SecretRefs []SecretMount `json:"secretRefs,omitempty"`

	// Persistence, when enabled, provisions a PVC mounted at /var/lib/zelos/<component>/.
	// +optional
	Persistence *PersistenceSpec `json:"persistence,omitempty"`

	// Autoscaling provisions an HPA targeting the rendered Deployment.
	// +optional
	Autoscaling *AutoscalingSpec `json:"autoscaling,omitempty"`

	// +optional
	Service *ServiceSpec `json:"service,omitempty"`

	// ImagePullSecrets are passed through to every Pod the operator renders.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// LogLevel overrides the OTEL_LOG_LEVEL applied to the pod.
	// +kubebuilder:validation:Enum=trace;debug;info;warn;error
	// +optional
	LogLevel string `json:"logLevel,omitempty"`
}

// TelemetrySpec controls the OpenTelemetry contract from docs/architecture/11-telemetry.md.
type TelemetrySpec struct {
	// Enabled deploys an OTel collector and injects OTEL_* env vars into every workload.
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// ExternalEndpoint, when set, suppresses the operator-installed collector
	// and points workloads at the given OTLP/gRPC endpoint instead.
	// +optional
	ExternalEndpoint string `json:"externalEndpoint,omitempty"`

	// LogLevel is the default OTEL_LOG_LEVEL applied to all components.
	// +kubebuilder:validation:Enum=trace;debug;info;warn;error
	// +kubebuilder:default=info
	// +optional
	LogLevel string `json:"logLevel,omitempty"`

	// Image overrides the OpenTelemetry collector image.
	// +optional
	Image ImageSpec `json:"image,omitempty"`
}

// PlatformDefaults supplies cluster-wide defaults applied to every component.
type PlatformDefaults struct {
	// +optional
	Image ImageSpec `json:"image,omitempty"`

	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// CommonStatus is embedded into every component CRD status block.
type CommonStatus struct {
	// ObservedGeneration is the .metadata.generation last reconciled.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions captures the standard Kubernetes-convention conditions
	// (Available, Ready, Progressing) for the resource. Managed via
	// meta.SetStatusCondition so transition times and observedGeneration
	// follow the upstream apimachinery semantics.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}
