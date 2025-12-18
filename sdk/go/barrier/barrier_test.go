package barrier

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

func TestWaitBarrier_Open(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-barrier",
			Namespace: "default",
		},
		Status: syncv1.BarrierStatus{
			Phase: syncv1.BarrierPhaseOpen,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(barrier).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	err := client.WaitBarrier(context.Background(), "test-barrier")
	require.NoError(t, err)
}

func TestWaitBarrier_Failed(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-barrier",
			Namespace: "default",
		},
		Status: syncv1.BarrierStatus{
			Phase: syncv1.BarrierPhaseFailed,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(barrier).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	err := client.WaitBarrier(context.Background(), "test-barrier")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "barrier test-barrier failed")
}

func TestWaitBarrier_Timeout(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-barrier",
			Namespace: "default",
		},
		Status: syncv1.BarrierStatus{
			Phase: syncv1.BarrierPhaseWaiting,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(barrier).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	err := client.WaitBarrier(context.Background(), "test-barrier", 
		konductor.WithTimeout(100*time.Millisecond))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestArriveBarrier(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	err := client.ArriveBarrier(context.Background(), "test-barrier", 
		konductor.WithHolder("test-holder"))
	require.NoError(t, err)
}

func TestArriveBarrier_DefaultHolder(t *testing.T) {
	originalHostname := os.Getenv("HOSTNAME")
	defer os.Setenv("HOSTNAME", originalHostname)

	os.Setenv("HOSTNAME", "test-pod")

	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	err := client.ArriveBarrier(context.Background(), "test-barrier")
	require.NoError(t, err)
}

func TestWithBarrier(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	executed := false
	err := client.WithBarrier(context.Background(), "test-barrier", func() error {
		executed = true
		return nil
	}, konductor.WithHolder("test-holder"))

	require.NoError(t, err)
	assert.True(t, executed)
}

func TestListBarriers(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	barriers := []runtime.Object{
		&syncv1.Barrier{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "barrier1",
				Namespace: "default",
			},
		},
		&syncv1.Barrier{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "barrier2",
				Namespace: "default",
			},
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(barriers...).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	result, err := client.ListBarriers(context.Background())
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestGetBarrier(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-barrier",
			Namespace: "default",
		},
		Spec: syncv1.BarrierSpec{
			Expected: 3,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(barrier).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	result, err := client.GetBarrier(context.Background(), "test-barrier")
	require.NoError(t, err)
	assert.Equal(t, "test-barrier", result.Name)
	assert.Equal(t, int32(3), result.Spec.Expected)
}

func TestGetBarrierStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-barrier",
			Namespace: "default",
		},
		Status: syncv1.BarrierStatus{
			Phase:   syncv1.BarrierPhaseWaiting,
			Arrived: 2,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(barrier).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	status, err := client.GetBarrierStatus(context.Background(), "test-barrier")
	require.NoError(t, err)
	assert.Equal(t, syncv1.BarrierPhaseWaiting, status.Phase)
	assert.Equal(t, int32(2), status.Arrived)
}