package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ZelosBackplaneSpec defines the message-bus substrate fronting the suite.
type ZelosBackplaneSpec struct {
	WorkloadSpec `json:",inline"`

	// Substrate selects nats (operator-installed) or redis/kafka (BYO).
	// +kubebuilder:validation:Enum=nats;redis;kafka
	// +kubebuilder:default=nats
	// +optional
	Substrate string `json:"substrate,omitempty"`

	// ExternalURL points workloads at an existing substrate; when set, no
	// in-cluster substrate StatefulSet is created.
	// +optional
	ExternalURL string `json:"externalURL,omitempty"`

	// TLSSecretRef names a Secret with ca.crt/tls.crt/tls.key for substrate TLS.
	// +optional
	TLSSecretRef string `json:"tlsSecretRef,omitempty"`
}

type ZelosBackplaneStatus struct {
	CommonStatus `json:",inline"`

	// URL is the connection URL (cluster-internal for NATS, externalURL for BYO substrates).
	// +optional
	URL string `json:"url,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=zbp,categories=zelos
// +kubebuilder:printcolumn:name="Substrate",type=string,JSONPath=`.spec.substrate`
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.status.url`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`

// ZelosBackplane is the message bus between gateway, broker, mcp, and clients.
type ZelosBackplane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ZelosBackplaneSpec   `json:"spec,omitempty"`
	Status ZelosBackplaneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type ZelosBackplaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ZelosBackplane `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ZelosBackplane{}, &ZelosBackplaneList{})
}
