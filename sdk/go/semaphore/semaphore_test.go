package semaphore

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

func TestAcquireSemaphore(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	semaphore := &syncv1.Semaphore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sem",
			Namespace: "default",
		},
		Spec: syncv1.SemaphoreSpec{
			Permits: 5,
		},
		Status: syncv1.SemaphoreStatus{
			Available: 3,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(semaphore).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	permit, err := client.AcquireSemaphore(context.Background(), "test-sem", 
		konductor.WithHolder("test-holder"),
		konductor.WithTTL(time.Hour))

	require.NoError(t, err)
	assert.NotNil(t, permit)
	assert.Equal(t, "test-sem", permit.Name())
	assert.Equal(t, "test-holder", permit.Holder())
}

func TestAcquireSemaphore_NoPermits(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	semaphore := &syncv1.Semaphore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sem",
			Namespace: "default",
		},
		Spec: syncv1.SemaphoreSpec{
			Permits: 5,
		},
		Status: syncv1.SemaphoreStatus{
			Available: 0,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(semaphore).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.AcquireSemaphore(ctx, "test-sem", 
		konductor.WithHolder("test-holder"),
		konductor.WithTimeout(50*time.Millisecond))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestPermit_Release(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	permit := &syncv1.Permit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sem-test-holder",
			Namespace: "default",
		},
		Spec: syncv1.PermitSpec{
			Semaphore: "test-sem",
			Holder:    "test-holder",
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(permit).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	p := &Permit{
		client:   client,
		name:     "test-sem",
		permitID: "test-sem-test-holder",
		holder:   "test-holder",
	}

	err := p.Release()
	require.NoError(t, err)
}

func TestWithSemaphore(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	semaphore := &syncv1.Semaphore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sem",
			Namespace: "default",
		},
		Spec: syncv1.SemaphoreSpec{
			Permits: 5,
		},
		Status: syncv1.SemaphoreStatus{
			Available: 3,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(semaphore).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	executed := false
	err := client.WithSemaphore(context.Background(), "test-sem", func() error {
		executed = true
		return nil
	}, konductor.WithHolder("test-holder"))

	require.NoError(t, err)
	assert.True(t, executed)
}

func TestListSemaphores(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	semaphores := []runtime.Object{
		&syncv1.Semaphore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sem1",
				Namespace: "default",
			},
		},
		&syncv1.Semaphore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sem2",
				Namespace: "default",
			},
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(semaphores...).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	result, err := client.ListSemaphores(context.Background())
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestGetSemaphore(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	semaphore := &syncv1.Semaphore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sem",
			Namespace: "default",
		},
		Spec: syncv1.SemaphoreSpec{
			Permits: 5,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(semaphore).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	result, err := client.GetSemaphore(context.Background(), "test-sem")
	require.NoError(t, err)
	assert.Equal(t, "test-sem", result.Name)
	assert.Equal(t, int32(5), result.Spec.Permits)
}

func TestAcquireSemaphore_DefaultHolder(t *testing.T) {
	originalHostname := os.Getenv("HOSTNAME")
	defer os.Setenv("HOSTNAME", originalHostname)

	os.Setenv("HOSTNAME", "test-pod")

	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	semaphore := &syncv1.Semaphore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sem",
			Namespace: "default",
		},
		Spec: syncv1.SemaphoreSpec{
			Permits: 5,
		},
		Status: syncv1.SemaphoreStatus{
			Available: 3,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(semaphore).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	permit, err := client.AcquireSemaphore(context.Background(), "test-sem")

	require.NoError(t, err)
	assert.Equal(t, "test-pod", permit.Holder())
}