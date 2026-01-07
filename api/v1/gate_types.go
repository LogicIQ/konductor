package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GateCondition defines a condition that must be met
type GateCondition struct {
	// Type of condition (Job, Semaphore, Barrier, Lease)
	Type string `json:"type"`

	// Name of the resource to check
	Name string `json:"name"`

	// Namespace of the resource (optional, defaults to gate's namespace)
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// State required for the condition to be met
	State string `json:"state"`

	// Value for numeric conditions (e.g., semaphore permits)
	// +optional
	Value *int32 `json:"value,omitempty"`
}

// GateSpec defines the desired state of Gate
type GateSpec struct {
	// Conditions that must be met for the gate to open
	Conditions []GateCondition `json:"conditions"`

	// Timeout for waiting for conditions
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`
}

// GateStatus defines the observed state of Gate
type GateStatus struct {
	// Phase represents the current state of the gate
	Phase GatePhase `json:"phase"`

	// ConditionStatuses tracks the status of each condition
	ConditionStatuses []GateConditionStatus `json:"conditionStatuses,omitempty"`

	// OpenedAt is when the gate opened
	// +optional
	OpenedAt *metav1.Time `json:"openedAt,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// GateConditionStatus tracks the status of a gate condition
type GateConditionStatus struct {
	// Type of condition
	Type string `json:"type"`

	// Name of the resource
	Name string `json:"name"`

	// Met indicates if the condition is satisfied
	Met bool `json:"met"`

	// Message provides details about the condition status
	Message string `json:"message,omitempty"`
}

// GatePhase represents the phase of a Gate
type GatePhase string

const (
	GatePhaseWaiting GatePhase = "Waiting"
	GatePhaseOpen    GatePhase = "Open"
	GatePhaseFailed  GatePhase = "Failed"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Conditions",type=string,JSONPath=`.spec.conditions[*].name`

// Gate is the Schema for the gates API
type Gate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GateSpec   `json:"spec,omitempty"`
	Status GateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GateList contains a list of Gate
type GateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Gate `json:"items"`
}
