package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LeaseRequestSpec defines the desired state of LeaseRequest
type LeaseRequestSpec struct {
	// Lease is the name of the lease being requested
	Lease string `json:"lease"`

	// Holder is the pod/job requesting the lease
	Holder string `json:"holder"`

	// Priority for lease acquisition (higher wins)
	// +optional
	Priority *int32 `json:"priority,omitempty"`

	// TTL is the time-to-live for the lease request
	// +optional
	TTL *metav1.Duration `json:"ttl,omitempty"`
}

// LeaseRequestStatus defines the observed state of LeaseRequest
type LeaseRequestStatus struct {
	// Phase represents the current state of the request
	Phase LeaseRequestPhase `json:"phase"`

	// RequestedAt is when the request was made
	// +optional
	RequestedAt *metav1.Time `json:"requestedAt,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// LeaseRequestPhase represents the phase of a LeaseRequest
type LeaseRequestPhase string

const (
	LeaseRequestPhasePending LeaseRequestPhase = "Pending"
	LeaseRequestPhaseGranted LeaseRequestPhase = "Granted"
	LeaseRequestPhaseDenied  LeaseRequestPhase = "Denied"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Lease",type=string,JSONPath=`.spec.lease`
//+kubebuilder:printcolumn:name="Holder",type=string,JSONPath=`.spec.holder`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

// LeaseRequest is the Schema for the lease requests API
type LeaseRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LeaseRequestSpec   `json:"spec,omitempty"`
	Status LeaseRequestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LeaseRequestList contains a list of LeaseRequest
type LeaseRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LeaseRequest `json:"items"`
}
