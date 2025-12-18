package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

func TestNewFromClient(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	tests := []struct {
		name              string
		namespace         string
		expectedNamespace string
	}{
		{
			name:              "with namespace",
			namespace:         "test-ns",
			expectedNamespace: "test-ns",
		},
		{
			name:              "empty namespace defaults to default",
			namespace:         "",
			expectedNamespace: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewFromClient(k8sClient, tt.namespace)
			assert.Equal(t, tt.expectedNamespace, client.Namespace())
			assert.Equal(t, k8sClient, client.K8sClient())
		})
	}
}

func TestClient_WithNamespace(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	client := NewFromClient(k8sClient, "original")

	newClient := client.WithNamespace("new-ns")

	assert.Equal(t, "original", client.Namespace())
	assert.Equal(t, "new-ns", newClient.Namespace())
	assert.Equal(t, k8sClient, newClient.K8sClient())
}

func TestOptions(t *testing.T) {
	opts := &Options{}

	WithTTL(300)(opts)
	WithTimeout(60)(opts)
	WithPriority(5)(opts)
	WithHolder("test-holder")(opts)

	assert.Equal(t, int64(300), opts.TTL.Nanoseconds())
	assert.Equal(t, int64(60), opts.Timeout.Nanoseconds())
	assert.Equal(t, int32(5), opts.Priority)
	assert.Equal(t, "test-holder", opts.Holder)
}

func TestClient_ReleaseSemaphorePermit(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	// Create a permit to delete
	permit := &syncv1.Permit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-semaphore-test-holder",
			Namespace: "test-ns",
		},
		Spec: syncv1.PermitSpec{
			Semaphore: "test-semaphore",
			Holder:    "test-holder",
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(permit).
		Build()

	client := NewFromClient(k8sClient, "test-ns")

	err := client.ReleaseSemaphorePermit(context.Background(), "test-semaphore", "test-holder")
	assert.NoError(t, err)

	// Verify permit was deleted
	var retrievedPermit syncv1.Permit
	err = k8sClient.Get(context.Background(), types.NamespacedName{
		Name:      "test-semaphore-test-holder",
		Namespace: "test-ns",
	}, &retrievedPermit)
	assert.True(t, errors.IsNotFound(err))
}

func TestClient_ReleaseLease(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	// Create a lease request to delete
	request := &syncv1.LeaseRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease-test-holder",
			Namespace: "test-ns",
		},
		Spec: syncv1.LeaseRequestSpec{
			Lease:  "test-lease",
			Holder: "test-holder",
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(request).
		Build()

	client := NewFromClient(k8sClient, "test-ns")

	err := client.ReleaseLease(context.Background(), "test-lease", "test-holder")
	assert.NoError(t, err)

	// Verify lease request was deleted
	var retrievedRequest syncv1.LeaseRequest
	err = k8sClient.Get(context.Background(), types.NamespacedName{
		Name:      "test-lease-test-holder",
		Namespace: "test-ns",
	}, &retrievedRequest)
	assert.True(t, errors.IsNotFound(err))
}

func TestClient_ListPermits(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	permit1 := &syncv1.Permit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sem-holder1",
			Namespace: "test-ns",
			Labels:    map[string]string{"semaphore": "test-sem"},
		},
		Spec: syncv1.PermitSpec{
			Semaphore: "test-sem",
			Holder:    "holder1",
		},
	}

	permit2 := &syncv1.Permit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sem-holder2",
			Namespace: "test-ns",
			Labels:    map[string]string{"semaphore": "test-sem"},
		},
		Spec: syncv1.PermitSpec{
			Semaphore: "test-sem",
			Holder:    "holder2",
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(permit1, permit2).
		Build()

	client := NewFromClient(k8sClient, "test-ns")

	permits, err := client.ListPermits(context.Background(), "test-sem")
	require.NoError(t, err)
	assert.Len(t, permits, 2)

	// Check permit holders
	holders := make([]string, len(permits))
	for i, permit := range permits {
		holders[i] = permit.Spec.Holder
	}
	assert.Contains(t, holders, "holder1")
	assert.Contains(t, holders, "holder2")
}

func TestClient_ListLeaseRequests(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	request1 := &syncv1.LeaseRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease-holder1",
			Namespace: "test-ns",
			Labels:    map[string]string{"lease": "test-lease"},
		},
		Spec: syncv1.LeaseRequestSpec{
			Lease:  "test-lease",
			Holder: "holder1",
		},
		Status: syncv1.LeaseRequestStatus{
			Phase: syncv1.LeaseRequestPhasePending,
		},
	}

	request2 := &syncv1.LeaseRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease-holder2",
			Namespace: "test-ns",
			Labels:    map[string]string{"lease": "test-lease"},
		},
		Spec: syncv1.LeaseRequestSpec{
			Lease:  "test-lease",
			Holder: "holder2",
		},
		Status: syncv1.LeaseRequestStatus{
			Phase: syncv1.LeaseRequestPhaseGranted,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(request1, request2).
		Build()

	client := NewFromClient(k8sClient, "test-ns")

	requests, err := client.ListLeaseRequests(context.Background(), "test-lease")
	require.NoError(t, err)
	assert.Len(t, requests, 2)

	// Check request holders
	holders := make([]string, len(requests))
	for i, req := range requests {
		holders[i] = req.Spec.Holder
	}
	assert.Contains(t, holders, "holder1")
	assert.Contains(t, holders, "holder2")

	// Check phases
	phases := make([]syncv1.LeaseRequestPhase, len(requests))
	for i, req := range requests {
		phases[i] = req.Status.Phase
	}
	assert.Contains(t, phases, syncv1.LeaseRequestPhasePending)
	assert.Contains(t, phases, syncv1.LeaseRequestPhaseGranted)
}