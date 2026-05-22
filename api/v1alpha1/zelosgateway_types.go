package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ZelosGatewaySpec defines the HTTP gateway that routes IDE traffic into the suite.
type ZelosGatewaySpec struct {
	WorkloadSpec `json:",inline"`

	// BackplaneRef names a ZelosBackplane in the same namespace.
	// +optional
	BackplaneRef string `json:"backplaneRef,omitempty"`

	// MCPRef names a ZelosMCP in the same namespace.
	// +optional
	MCPRef string `json:"mcpRef,omitempty"`

	// AuthProviderSecretRef names a Secret with OIDC client credentials.
	// +optional
	AuthProviderSecretRef string `json:"authProviderSecretRef,omitempty"`
}

// ZelosGatewayStatus reports the gateway reconcile state.
type ZelosGatewayStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=zgw,categories=zelos
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ZelosGateway is the HTTP entry point for IDE traffic. Stateless; scales horizontally.
type ZelosGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ZelosGatewaySpec   `json:"spec,omitempty"`
	Status ZelosGatewayStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ZelosGatewayList contains a list of ZelosGateway.
type ZelosGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ZelosGateway `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ZelosGateway{}, &ZelosGatewayList{})
}
