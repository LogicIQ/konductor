package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ArrivalSpec defines the desired state of Arrival
type ArrivalSpec struct {
	// Barrier is the name of the barrier this arrival belongs to
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	Barrier string `json:"barrier"`

	// Holder is the pod/job that has arrived
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Holder string `json:"holder"`
}

// ArrivalStatus defines the observed state of Arrival
type ArrivalStatus struct {
	// Phase represents the current state of the arrival
	// +kubebuilder:default=Recorded
	// +kubebuilder:validation:Enum=Recorded
	Phase ArrivalPhase `json:"phase"`

	// ArrivedAt is when the arrival was recorded
	// +optional
	ArrivedAt *metav1.Time `json:"arrivedAt,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ArrivalPhase represents the phase of an Arrival
type ArrivalPhase string

const (
	ArrivalPhaseRecorded ArrivalPhase = "Recorded"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Barrier",type=string,JSONPath=`.spec.barrier`
//+kubebuilder:printcolumn:name="Holder",type=string,JSONPath=`.spec.holder`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

// Arrival is the Schema for the arrivals API
type Arrival struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArrivalSpec   `json:"spec,omitempty"`
	Status ArrivalStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ArrivalList contains a list of Arrival
type ArrivalList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Arrival `json:"items"`
}
