package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ZelosPlatformSpec is the umbrella spec composing every component.
type ZelosPlatformSpec struct {
	// ImagePullSecret is the standard ghcr.io pull-secret name applied to every workload.
	// +kubebuilder:default=ghcr-pull-secret
	// +optional
	ImagePullSecret string `json:"imagePullSecret,omitempty"`

	// +optional
	Defaults PlatformDefaults `json:"defaults,omitempty"`

	// +optional
	Telemetry TelemetrySpec `json:"telemetry,omitempty"`

	// +optional
	Gateway PlatformComponent `json:"gateway,omitempty"`

	// +optional
	Backplane PlatformBackplane `json:"backplane,omitempty"`

	// +optional
	MCP PlatformMCP `json:"mcp,omitempty"`

	// +optional
	Broker PlatformBroker `json:"broker,omitempty"`

	// +optional
	Server PlatformComponent `json:"server,omitempty"`

	// +optional
	Client PlatformClient `json:"client,omitempty"`
}

// PlatformComponent is the per-component knob bag rendered into child CRs.
type PlatformComponent struct {
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// +optional
	Image ImageSpec `json:"image,omitempty"`

	// +optional
	Autoscaling *AutoscalingSpec `json:"autoscaling,omitempty"`

	// +optional
	Persistence *PersistenceSpec `json:"persistence,omitempty"`

	// +optional
	Config map[string]string `json:"config,omitempty"`

	// +optional
	SecretRefs []SecretMount `json:"secretRefs,omitempty"`
}

// PlatformBackplane extends PlatformComponent with substrate selection.
type PlatformBackplane struct {
	PlatformComponent `json:",inline"`

	// Substrate selects the message bus. NATS is operator-installed; redis/kafka are BYO.
	// +kubebuilder:validation:Enum=nats;redis;kafka
	// +kubebuilder:default=nats
	// +optional
	Substrate string `json:"substrate,omitempty"`

	// ExternalURL points workloads at an existing substrate when set.
	// +optional
	ExternalURL string `json:"externalURL,omitempty"`

	// TLSSecretRef references a Secret containing ca.crt/tls.crt/tls.key for substrate TLS.
	// +optional
	TLSSecretRef string `json:"tlsSecretRef,omitempty"`
}

// PlatformMCP extends PlatformComponent with auth-provider wiring.
type PlatformMCP struct {
	PlatformComponent `json:",inline"`

	// +optional
	AuthProviderSecretRef string `json:"authProviderSecretRef,omitempty"`

	// +optional
	MCPServersConfigMapRef string `json:"mcpServersConfigMapRef,omitempty"`
}

// PlatformBroker extends PlatformComponent with tunnel-broker fields.
type PlatformBroker struct {
	PlatformComponent `json:",inline"`

	// +kubebuilder:validation:Enum=tailscale;wireguard;mtls-ws;ssh
	// +optional
	TunnelTransport string `json:"tunnelTransport,omitempty"`

	// +optional
	AllowedLLMHosts []string `json:"allowedLLMHosts,omitempty"`
}

// PlatformClient extends PlatformComponent with runtime selection and DaemonSet hints.
type PlatformClient struct {
	PlatformComponent `json:",inline"`

	// InCluster opts the client into an in-cluster DaemonSet on labeled GPU nodes.
	// Default deployment path is Ansible/host via zelos.dgx.
	// +optional
	InCluster bool `json:"inCluster,omitempty"`

	// +kubebuilder:validation:Enum=vllm;ollama
	// +optional
	Runtime string `json:"runtime,omitempty"`

	// +optional
	RuntimeURL string `json:"runtimeURL,omitempty"`

	// +optional
	Model string `json:"model,omitempty"`

	// +optional
	SubscribeTopics []string `json:"subscribeTopics,omitempty"`
}

// ZelosPlatformStatus captures the suite-level reconcile status.
type ZelosPlatformStatus struct {
	CommonStatus `json:",inline"`

	// Phase is a short, human-readable summary of where the install is in
	// its lifecycle (e.g. Pending, Progressing, Ready, Degraded). It is
	// derived from the conditions and is intended for `kubectl get` display
	// only; automation should read the conditions.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Children lists the per-component CRs the suite currently owns, each
	// with its rolled-up readiness so operators can see component health
	// without describing every child CR.
	// +optional
	Children []ChildRef `json:"children,omitempty"`
}

// ChildRef is a reference to an owned per-component CR plus its rolled-up
// readiness as observed by the umbrella reconciler.
type ChildRef struct {
	// Kind is the child CR Kind (e.g. ZelosGateway).
	Kind string `json:"kind"`

	// Name is the child CR name.
	Name string `json:"name"`

	// Ready is the string form of the child's Ready condition status
	// ("True"/"False"/"Unknown").
	// +optional
	Ready string `json:"ready,omitempty"`

	// Message is a human-readable explanation of the child's current
	// readiness (e.g. "2/2 replicas available").
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=zp,categories=zelos
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ZelosPlatform is the umbrella resource that composes every Zelos component
// into a single deployable unit. Apply one ZelosPlatform per environment
// (dev, staging, prod). The reconciler creates owned per-component CRs.
type ZelosPlatform struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ZelosPlatformSpec   `json:"spec,omitempty"`
	Status ZelosPlatformStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ZelosPlatformList contains a list of ZelosPlatform.
type ZelosPlatformList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ZelosPlatform `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ZelosPlatform{}, &ZelosPlatformList{})
}
