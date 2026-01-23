package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MutexSpec defines the desired state of Mutex
type MutexSpec struct {
	// TTL is the optional time-to-live for automatic unlock
	// +optional
	TTL *metav1.Duration `json:"ttl,omitempty"`
}

// MutexStatus defines the observed state of Mutex
type MutexStatus struct {
	// Holder is the current lock holder
	// +optional
	Holder string `json:"holder,omitempty"`

	// LockedAt is when the mutex was locked
	// +optional
	LockedAt *metav1.Time `json:"lockedAt,omitempty"`

	// ExpiresAt is when the mutex expires (if TTL is set)
	// +optional
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`

	// Phase represents the current state of the mutex
	// +kubebuilder:validation:Enum=Unlocked;Locked
	Phase MutexPhase `json:"phase"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// MutexPhase represents the phase of a Mutex
type MutexPhase string

const (
	MutexPhaseUnlocked MutexPhase = "Unlocked"
	MutexPhaseLocked   MutexPhase = "Locked"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Holder",type=string,JSONPath=`.status.holder`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Locked",type=date,JSONPath=`.status.lockedAt`

// Mutex is the Schema for the mutexes API
type Mutex struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MutexSpec   `json:"spec,omitempty"`
	Status MutexStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MutexList contains a list of Mutex
type MutexList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Mutex `json:"items"`
}
