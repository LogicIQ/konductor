package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PermitSpec defines the desired state of Permit
type PermitSpec struct {
	// Semaphore is the name of the semaphore this permit belongs to
	Semaphore string `json:"semaphore"`

	// Holder is the pod/job that owns this permit
	Holder string `json:"holder"`

	// TTL is the time-to-live for this permit
	// +optional
	TTL *metav1.Duration `json:"ttl,omitempty"`
}

// PermitStatus defines the observed state of Permit
type PermitStatus struct {
	// Phase represents the current state of the permit
	Phase PermitPhase `json:"phase"`

	// AcquiredAt is when the permit was acquired
	// +optional
	AcquiredAt *metav1.Time `json:"acquiredAt,omitempty"`

	// ExpiresAt is when the permit expires
	// +optional
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// PermitPhase represents the phase of a Permit
type PermitPhase string

const (
	PermitPhaseGranted PermitPhase = "Granted"
	PermitPhaseDenied  PermitPhase = "Denied"
	PermitPhaseExpired PermitPhase = "Expired"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Semaphore",type=string,JSONPath=`.spec.semaphore`
//+kubebuilder:printcolumn:name="Holder",type=string,JSONPath=`.spec.holder`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

// Permit is the Schema for the permits API
type Permit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PermitSpec   `json:"spec,omitempty"`
	Status PermitStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PermitList contains a list of Permit
type PermitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Permit `json:"items"`
}
