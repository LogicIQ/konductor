package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LeaseSpec defines the desired state of Lease
type LeaseSpec struct {
	// TTL is the time-to-live for the lease
	// +kubebuilder:validation:Required
	TTL metav1.Duration `json:"ttl"`

	// Priority for lease acquisition (higher wins)
	// +optional
	Priority *int32 `json:"priority,omitempty"`
}

// LeaseStatus defines the observed state of Lease
type LeaseStatus struct {
	// Holder is the current lease holder
	// +optional
	Holder string `json:"holder,omitempty"`

	// AcquiredAt is when the lease was acquired
	// +optional
	AcquiredAt *metav1.Time `json:"acquiredAt,omitempty"`

	// ExpiresAt is when the lease expires
	// +optional
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`

	// Phase represents the current state of the lease
	Phase LeasePhase `json:"phase"`

	// RenewCount tracks the number of renewals
	RenewCount int32 `json:"renewCount,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// LeasePhase represents the phase of a Lease
type LeasePhase string

const (
	LeasePhaseAvailable LeasePhase = "Available"
	LeasePhaseHeld      LeasePhase = "Held"
	LeasePhaseExpired   LeasePhase = "Expired"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Holder",type=string,JSONPath=`.status.holder`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Acquired",type=date,JSONPath=`.status.acquiredAt`

// Lease is the Schema for the leases API
type Lease struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LeaseSpec   `json:"spec,omitempty"`
	Status LeaseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LeaseList contains a list of Lease
type LeaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Lease `json:"items"`
}
