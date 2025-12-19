package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SemaphoreSpec defines the desired state of Semaphore
type SemaphoreSpec struct {
	// Permits is the maximum number of concurrent permits allowed
	Permits int32 `json:"permits"`

	// TTL is the default time-to-live for permits
	// +optional
	TTL *metav1.Duration `json:"ttl,omitempty"`
}

// SemaphoreStatus defines the observed state of Semaphore
type SemaphoreStatus struct {
	// InUse is the current number of permits in use
	InUse int32 `json:"inUse"`

	// Available is the number of available permits
	Available int32 `json:"available"`

	// Phase represents the current state of the semaphore
	Phase SemaphorePhase `json:"phase"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// SemaphorePhase represents the phase of a Semaphore
type SemaphorePhase string

const (
	SemaphorePhaseReady       SemaphorePhase = "Ready"
	SemaphorePhaseFull        SemaphorePhase = "Full"
	SemaphorePhaseUnavailable SemaphorePhase = "Unavailable"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Permits",type=integer,JSONPath=`.spec.permits`
//+kubebuilder:printcolumn:name="InUse",type=integer,JSONPath=`.status.inUse`
//+kubebuilder:printcolumn:name="Available",type=integer,JSONPath=`.status.available`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

// Semaphore is the Schema for the semaphores API
type Semaphore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SemaphoreSpec   `json:"spec,omitempty"`
	Status SemaphoreStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SemaphoreList contains a list of Semaphore
type SemaphoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Semaphore `json:"items"`
}
