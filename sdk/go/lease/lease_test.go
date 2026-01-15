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

func TestList(t *testing.T) {
	lease1 := &syncv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lease1",
			Namespace: "test-ns",
		},
		Spec: syncv1.LeaseSpec{
			TTL: &metav1.Duration{Duration: 300},
		},
	}

	client := setupTestClient(t, lease1)

	leases, err := List(client, context.Background())
	require.NoError(t, err)
	assert.Len(t, leases, 1)
	assert.Equal(t, "lease1", leases[0].Name)
}

func TestGet(t *testing.T) {
	lease := &syncv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease",
			Namespace: "test-ns",
		},
		Spec: syncv1.LeaseSpec{
			TTL: &metav1.Duration{Duration: 300},
		},
	}

	client := setupTestClient(t, lease)

	result, err := Get(client, context.Background(), "test-lease")
	require.NoError(t, err)
	assert.Equal(t, "test-lease", result.Name)
}

func TestCreate(t *testing.T) {
	client := setupTestClient(t)

	err := Create(client, context.Background(), "test-lease")
	assert.NoError(t, err)
}

func TestDelete(t *testing.T) {
	lease := &syncv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease",
			Namespace: "test-ns",
		},
	}
	client := setupTestClient(t, lease)

	err := Delete(client, context.Background(), "test-lease")
	assert.NoError(t, err)
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

	available, err := IsAvailable(client, context.Background(), "test-lease")
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

	available, err := IsAvailable(client, context.Background(), "test-lease")
	require.NoError(t, err)
	assert.False(t, available)
}

func TestUpdate(t *testing.T) {
	lease := &syncv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease",
			Namespace: "test-ns",
		},
		Spec: syncv1.LeaseSpec{
			TTL: &metav1.Duration{Duration: 300},
		},
	}
	client := setupTestClient(t, lease)

	// Update TTL
	lease.Spec.TTL = &metav1.Duration{Duration: 600}
	err := Update(client, context.Background(), lease)
	assert.NoError(t, err)
}
