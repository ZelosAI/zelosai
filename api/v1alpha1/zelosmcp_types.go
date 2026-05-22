package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ZelosMCPSpec defines the MCP aggregator (FastMCP server) deployment.
type ZelosMCPSpec struct {
	WorkloadSpec `json:",inline"`

	// AuthProviderSecretRef names a Secret with auth.key + provider configs.
	// Standard keys: "auth.key", "providers.json".
	// +optional
	AuthProviderSecretRef string `json:"authProviderSecretRef,omitempty"`

	// MCPServersConfigMapRef names a ConfigMap with the mcpServers catalog.
	// +optional
	MCPServersConfigMapRef string `json:"mcpServersConfigMapRef,omitempty"`
}

type ZelosMCPStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=zmcp,categories=zelos
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ZelosMCP is the MCP aggregator. Stateful (SQLite under /var/lib/zelos/zelosmcp/).
type ZelosMCP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ZelosMCPSpec   `json:"spec,omitempty"`
	Status ZelosMCPStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type ZelosMCPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ZelosMCP `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ZelosMCP{}, &ZelosMCPList{})
}
