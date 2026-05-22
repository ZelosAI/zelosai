package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ZelosServerSpec is the minimal spec for the (placeholder) zelosserver component.
type ZelosServerSpec struct {
	WorkloadSpec `json:",inline"`
}

type ZelosServerStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=zsv,categories=zelos
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`

// ZelosServer is a placeholder component (scope TBD).
type ZelosServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ZelosServerSpec   `json:"spec,omitempty"`
	Status ZelosServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type ZelosServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ZelosServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ZelosServer{}, &ZelosServerList{})
}
