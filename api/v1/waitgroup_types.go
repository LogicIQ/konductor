package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WaitGroupSpec defines the desired state of WaitGroup
type WaitGroupSpec struct {
	// TTL is the optional time-to-live for cleanup
	// +optional
	TTL *metav1.Duration `json:"ttl,omitempty"`
}

// WaitGroupStatus defines the observed state of WaitGroup
type WaitGroupStatus struct {
	// Counter is the current count
	Counter int32 `json:"counter"`

	// Phase represents the current state
	Phase WaitGroupPhase `json:"phase"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// WaitGroupPhase represents the phase of a WaitGroup
type WaitGroupPhase string

const (
	WaitGroupPhaseWaiting WaitGroupPhase = "Waiting"
	WaitGroupPhaseDone    WaitGroupPhase = "Done"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Counter",type=integer,JSONPath=`.status.counter`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

// WaitGroup is the Schema for the waitgroups API
type WaitGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WaitGroupSpec   `json:"spec,omitempty"`
	Status WaitGroupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WaitGroupList contains a list of WaitGroup
type WaitGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WaitGroup `json:"items"`
}
