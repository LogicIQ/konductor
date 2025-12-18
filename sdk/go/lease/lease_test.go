package lease

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
	konductor "github.com/LogicIQ/konductor/sdk/go/client"
)

func TestAcquireLease_Granted(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	request := &syncv1.LeaseRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease-test-holder",
			Namespace: "default",
		},
		Status: syncv1.LeaseRequestStatus{
			Phase: syncv1.LeaseRequestPhaseGranted,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(request).
		WithStatusSubresource(&syncv1.LeaseRequest{}).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	lease, err := client.AcquireLease(context.Background(), "test-lease", 
		konductor.WithHolder("test-holder"))

	require.NoError(t, err)
	assert.NotNil(t, lease)
	assert.Equal(t, "test-lease", lease.Name())
	assert.Equal(t, "test-holder", lease.Holder())
}

func TestAcquireLease_Denied(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	request := &syncv1.LeaseRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease-test-holder",
			Namespace: "default",
		},
		Status: syncv1.LeaseRequestStatus{
			Phase: syncv1.LeaseRequestPhaseDenied,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(request).
		WithStatusSubresource(&syncv1.LeaseRequest{}).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	_, err := client.AcquireLease(context.Background(), "test-lease", 
		konductor.WithHolder("test-holder"))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lease request denied")
}

func TestAcquireLease_Timeout(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	request := &syncv1.LeaseRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease-test-holder",
			Namespace: "default",
		},
		Status: syncv1.LeaseRequestStatus{
			Phase: syncv1.LeaseRequestPhasePending,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(request).
		WithStatusSubresource(&syncv1.LeaseRequest{}).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	_, err := client.AcquireLease(context.Background(), "test-lease", 
		konductor.WithHolder("test-holder"),
		konductor.WithTimeout(100*time.Millisecond))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestLease_Release(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	request := &syncv1.LeaseRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease-test-holder",
			Namespace: "default",
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(request).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	lease := &Lease{
		client:    client,
		name:      "test-lease",
		requestID: "test-lease-test-holder",
		holder:    "test-holder",
	}

	err := lease.Release()
	require.NoError(t, err)
}

func TestWithLease(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	request := &syncv1.LeaseRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease-test-holder",
			Namespace: "default",
		},
		Status: syncv1.LeaseRequestStatus{
			Phase: syncv1.LeaseRequestPhaseGranted,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(request).
		WithStatusSubresource(&syncv1.LeaseRequest{}).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	executed := false
	err := client.WithLease(context.Background(), "test-lease", func() error {
		executed = true
		return nil
	}, konductor.WithHolder("test-holder"))

	require.NoError(t, err)
	assert.True(t, executed)
}

func TestTryAcquireLease(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	request := &syncv1.LeaseRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease-test-holder",
			Namespace: "default",
		},
		Status: syncv1.LeaseRequestStatus{
			Phase: syncv1.LeaseRequestPhaseGranted,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(request).
		WithStatusSubresource(&syncv1.LeaseRequest{}).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	lease, err := client.TryAcquireLease(context.Background(), "test-lease", 
		konductor.WithHolder("test-holder"))

	require.NoError(t, err)
	assert.NotNil(t, lease)
}

func TestListLeases(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	leases := []runtime.Object{
		&syncv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lease1",
				Namespace: "default",
			},
		},
		&syncv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lease2",
				Namespace: "default",
			},
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(leases...).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	result, err := client.ListLeases(context.Background())
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestGetLease(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	lease := &syncv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease",
			Namespace: "default",
		},
		Spec: syncv1.LeaseSpec{
			TTL: metav1.Duration{Duration: time.Hour},
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(lease).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	result, err := client.GetLease(context.Background(), "test-lease")
	require.NoError(t, err)
	assert.Equal(t, "test-lease", result.Name)
	assert.Equal(t, time.Hour, result.Spec.TTL.Duration)
}

func TestIsLeaseAvailable(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	lease := &syncv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease",
			Namespace: "default",
		},
		Status: syncv1.LeaseStatus{
			Phase: syncv1.LeasePhaseAvailable,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(lease).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	available, err := client.IsLeaseAvailable(context.Background(), "test-lease")
	require.NoError(t, err)
	assert.True(t, available)
}

func TestAcquireLease_DefaultHolder(t *testing.T) {
	originalHostname := os.Getenv("HOSTNAME")
	defer os.Setenv("HOSTNAME", originalHostname)

	os.Setenv("HOSTNAME", "test-pod")

	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	request := &syncv1.LeaseRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease-test-pod",
			Namespace: "default",
		},
		Status: syncv1.LeaseRequestStatus{
			Phase: syncv1.LeaseRequestPhaseGranted,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(request).
		WithStatusSubresource(&syncv1.LeaseRequest{}).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	lease, err := client.AcquireLease(context.Background(), "test-lease")

	require.NoError(t, err)
	assert.Equal(t, "test-pod", lease.Holder())
}