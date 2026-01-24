package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BarrierSpec defines the desired state of Barrier
// +kubebuilder:validation:XValidation:rule="!has(self.quorum) || self.quorum <= self.expected",message="quorum must not exceed expected"
// +kubebuilder:validation:XValidation:rule="!has(self.timeout) || self.timeout.matches(r'^([0-9]+(\\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$')",message="timeout must be a valid duration (e.g., 30s, 5m, 1h)"
type BarrierSpec struct {
	// Expected is the number of arrivals required to open the barrier
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	Expected int32 `json:"expected"`

	// Timeout is the maximum time to wait for all arrivals
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Quorum is the minimum number of arrivals to open (optional)
	// +optional
	// +kubebuilder:validation:Minimum=1
	Quorum *int32 `json:"quorum,omitempty"`
}

// BarrierStatus defines the observed state of Barrier
type BarrierStatus struct {
	// Arrived is the current number of arrivals
	Arrived int32 `json:"arrived"`

	// Phase represents the current state of the barrier
	Phase BarrierPhase `json:"phase"`

	// Arrivals tracks which pods have arrived
	Arrivals []string `json:"arrivals,omitempty"`

	// OpenedAt is when the barrier opened
	// +optional
	OpenedAt *metav1.Time `json:"openedAt,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// BarrierPhase represents the phase of a Barrier
type BarrierPhase string

const (
	BarrierPhaseWaiting BarrierPhase = "Waiting"
	BarrierPhaseOpen    BarrierPhase = "Open"
	BarrierPhaseFailed  BarrierPhase = "Failed"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Expected",type=integer,JSONPath=`.spec.expected`
//+kubebuilder:printcolumn:name="Arrived",type=integer,JSONPath=`.status.arrived`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

// Barrier is the Schema for the barriers API
type Barrier struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BarrierSpec   `json:"spec,omitempty"`
	Status BarrierStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BarrierList contains a list of Barrier
type BarrierList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Barrier `json:"items"`
}
