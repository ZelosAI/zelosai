package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ZelosClientSpec defines the inference worker that subscribes to backplane topics.
// Default deployment path is Ansible/host (zelos.dgx). InCluster=true opts into a
// DaemonSet on labeled GPU nodes.
type ZelosClientSpec struct {
	WorkloadSpec `json:",inline"`

	// InCluster opts into an in-cluster DaemonSet on nodes matching nodeSelector.
	// +optional
	InCluster bool `json:"inCluster,omitempty"`

	// Runtime selects the local inference runtime.
	// +kubebuilder:validation:Enum=vllm;ollama
	// +optional
	Runtime string `json:"runtime,omitempty"`

	// RuntimeURL is the local inference endpoint (e.g. http://localhost:8000).
	// +optional
	RuntimeURL string `json:"runtimeURL,omitempty"`

	// Model is the model identifier to load.
	// +optional
	Model string `json:"model,omitempty"`

	// SubscribeTopics lists backplane topics this client claims work from.
	// +optional
	SubscribeTopics []string `json:"subscribeTopics,omitempty"`

	// BackplaneRef names a ZelosBackplane in the same namespace.
	// +optional
	BackplaneRef string `json:"backplaneRef,omitempty"`
}

type ZelosClientStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=zcl,categories=zelos
// +kubebuilder:printcolumn:name="Runtime",type=string,JSONPath=`.spec.runtime`
// +kubebuilder:printcolumn:name="Model",type=string,JSONPath=`.spec.model`
// +kubebuilder:printcolumn:name="InCluster",type=boolean,JSONPath=`.spec.inCluster`

// ZelosClient is the worker bridging backplane topics to local vLLM/Ollama.
type ZelosClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ZelosClientSpec   `json:"spec,omitempty"`
	Status ZelosClientStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type ZelosClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ZelosClient `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ZelosClient{}, &ZelosClientList{})
}
