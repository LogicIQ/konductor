package lease

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
	konductor "github.com/LogicIQ/konductor/sdk/go/client"
)

func setupTestClient(t *testing.T, objects ...runtime.Object) *konductor.Client {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(objects...).
		Build()

	return konductor.NewFromClient(k8sClient, "test-ns")
}

func TestListLeases(t *testing.T) {
	lease1 := &syncv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lease1",
			Namespace: "test-ns",
		},
		Spec: syncv1.LeaseSpec{
			TTL: metav1.Duration{Duration: 300},
		},
		Status: syncv1.LeaseStatus{
			Phase:      syncv1.LeasePhaseAvailable,
			RenewCount: 0,
		},
	}

	lease2 := &syncv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lease2",
			Namespace: "test-ns",
		},
		Spec: syncv1.LeaseSpec{
			TTL: metav1.Duration{Duration: 600},
		},
		Status: syncv1.LeaseStatus{
			Phase:      syncv1.LeasePhaseHeld,
			Holder:     "test-holder",
			RenewCount: 5,
		},
	}

	client := setupTestClient(t, lease1, lease2)

	leases, err := ListLeases(client, context.Background())
	require.NoError(t, err)
	assert.Len(t, leases, 2)

	// Check that we got both leases
	names := make([]string, len(leases))
	for i, lease := range leases {
		names[i] = lease.Name
	}
	assert.Contains(t, names, "lease1")
	assert.Contains(t, names, "lease2")
}

func TestGetLease(t *testing.T) {
	lease := &syncv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease",
			Namespace: "test-ns",
		},
		Spec: syncv1.LeaseSpec{
			TTL: metav1.Duration{Duration: 300},
		},
		Status: syncv1.LeaseStatus{
			Phase:      syncv1.LeasePhaseHeld,
			Holder:     "test-holder",
			RenewCount: 3,
		},
	}

	client := setupTestClient(t, lease)

	result, err := GetLease(client, context.Background(), "test-lease")
	require.NoError(t, err)
	assert.Equal(t, "test-lease", result.Name)
	assert.Equal(t, syncv1.LeasePhaseHeld, result.Status.Phase)
	assert.Equal(t, "test-holder", result.Status.Holder)
	assert.Equal(t, int32(3), result.Status.RenewCount)
}

func TestGetLease_NotFound(t *testing.T) {
	client := setupTestClient(t)

	_, err := GetLease(client, context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get lease")
}

func TestIsLeaseAvailable_Available(t *testing.T) {
	lease := &syncv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease",
			Namespace: "test-ns",
		},
		Status: syncv1.LeaseStatus{
			Phase: syncv1.LeasePhaseAvailable,
		},
	}

	client := setupTestClient(t, lease)

	available, err := IsLeaseAvailable(client, context.Background(), "test-lease")
	require.NoError(t, err)
	assert.True(t, available)
}

func TestIsLeaseAvailable_Held(t *testing.T) {
	lease := &syncv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease",
			Namespace: "test-ns",
		},
		Status: syncv1.LeaseStatus{
			Phase:  syncv1.LeasePhaseHeld,
			Holder: "someone-else",
		},
	}

	client := setupTestClient(t, lease)

	available, err := IsLeaseAvailable(client, context.Background(), "test-lease")
	require.NoError(t, err)
	assert.False(t, available)
}

func TestAcquireLease_NotImplemented(t *testing.T) {
	client := setupTestClient(t)

	_, err := AcquireLease(client, context.Background(), "test-lease")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestWithLease_NotImplemented(t *testing.T) {
	client := setupTestClient(t)

	err := WithLease(client, context.Background(), "test-lease", func() error {
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestTryAcquireLease_NotImplemented(t *testing.T) {
	client := setupTestClient(t)

	_, err := TryAcquireLease(client, context.Background(), "test-lease")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}