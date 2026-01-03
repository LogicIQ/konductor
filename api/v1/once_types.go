package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OnceSpec defines the desired state of Once
type OnceSpec struct {
	// TTL is the optional time-to-live for cleanup
	// +optional
	TTL *metav1.Duration `json:"ttl,omitempty"`
}

// OnceStatus defines the observed state of Once
type OnceStatus struct {
	// Executed indicates if the action has been executed
	Executed bool `json:"executed"`

	// Executor is who executed the action
	// +optional
	Executor string `json:"executor,omitempty"`

	// ExecutedAt is when the action was executed
	// +optional
	ExecutedAt *metav1.Time `json:"executedAt,omitempty"`

	// Phase represents the current state
	Phase OncePhase `json:"phase"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// OncePhase represents the phase of a Once
type OncePhase string

const (
	OncePhasePending  OncePhase = "Pending"
	OncePhaseExecuted OncePhase = "Executed"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Executed",type=boolean,JSONPath=`.status.executed`
//+kubebuilder:printcolumn:name="Executor",type=string,JSONPath=`.status.executor`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="ExecutedAt",type=date,JSONPath=`.status.executedAt`

// Once is the Schema for the onces API
type Once struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OnceSpec   `json:"spec,omitempty"`
	Status OnceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OnceList contains a list of Once
type OnceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Once `json:"items"`
}
