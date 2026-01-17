package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RWMutexSpec defines the desired state of RWMutex
type RWMutexSpec struct {
	// TTL is the optional time-to-live for automatic unlock
	// +optional
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Format=duration
	TTL *metav1.Duration `json:"ttl,omitempty"`
}

// RWMutexStatus defines the observed state of RWMutex
type RWMutexStatus struct {
	// WriteHolder is the current write lock holder
	// +optional
	WriteHolder string `json:"writeHolder,omitempty"`

	// ReadHolders is the list of current read lock holders
	// +optional
	ReadHolders []string `json:"readHolders,omitempty"`

	// LockedAt is when the lock was acquired
	// +optional
	LockedAt *metav1.Time `json:"lockedAt,omitempty"`

	// ExpiresAt is when the lock expires (if TTL is set)
	// +optional
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`

	// Phase represents the current state of the rwmutex
	Phase RWMutexPhase `json:"phase"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// RWMutexPhase represents the phase of a RWMutex
type RWMutexPhase string

const (
	RWMutexPhaseUnlocked    RWMutexPhase = "Unlocked"
	RWMutexPhaseReadLocked  RWMutexPhase = "ReadLocked"
	RWMutexPhaseWriteLocked RWMutexPhase = "WriteLocked"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="WriteHolder",type=string,JSONPath=`.status.writeHolder`
//+kubebuilder:printcolumn:name="ReadHolders",type=string,JSONPath=`.status.readHolders`,priority=1
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Locked",type=date,JSONPath=`.status.lockedAt`

// RWMutex is the Schema for the rwmutexes API
type RWMutex struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RWMutexSpec   `json:"spec,omitempty"`
	Status RWMutexStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RWMutexList contains a list of RWMutex
type RWMutexList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RWMutex `json:"items"`
}
