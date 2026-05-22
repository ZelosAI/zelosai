package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ZelosBrokerSpec defines the sync-path broker / tunnel terminator.
type ZelosBrokerSpec struct {
	WorkloadSpec `json:",inline"`

	// TunnelTransport selects the secure-tunnel transport.
	// +kubebuilder:validation:Enum=tailscale;wireguard;mtls-ws;ssh
	// +optional
	TunnelTransport string `json:"tunnelTransport,omitempty"`

	// AllowedLLMHosts restricts which downstream LLM hosts the broker may reach.
	// +optional
	AllowedLLMHosts []string `json:"allowedLLMHosts,omitempty"`
}

type ZelosBrokerStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=zbk,categories=zelos
// +kubebuilder:printcolumn:name="Transport",type=string,JSONPath=`.spec.tunnelTransport`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`

// ZelosBroker terminates customer-side sync tunnels and pulls assets from /workspace.
type ZelosBroker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ZelosBrokerSpec   `json:"spec,omitempty"`
	Status ZelosBrokerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type ZelosBrokerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ZelosBroker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ZelosBroker{}, &ZelosBrokerList{})
}
