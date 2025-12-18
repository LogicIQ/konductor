package semaphore

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

func TestListSemaphores(t *testing.T) {
	semaphore1 := &syncv1.Semaphore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sem1",
			Namespace: "test-ns",
		},
		Spec: syncv1.SemaphoreSpec{
			Permits: 5,
		},
		Status: syncv1.SemaphoreStatus{
			InUse:     2,
			Available: 3,
			Phase:     syncv1.SemaphorePhaseReady,
		},
	}

	semaphore2 := &syncv1.Semaphore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sem2",
			Namespace: "test-ns",
		},
		Spec: syncv1.SemaphoreSpec{
			Permits: 3,
		},
		Status: syncv1.SemaphoreStatus{
			InUse:     3,
			Available: 0,
			Phase:     syncv1.SemaphorePhaseFull,
		},
	}

	client := setupTestClient(t, semaphore1, semaphore2)

	semaphores, err := ListSemaphores(client, context.Background())
	require.NoError(t, err)
	assert.Len(t, semaphores, 2)

	// Check that we got both semaphores
	names := make([]string, len(semaphores))
	for i, sem := range semaphores {
		names[i] = sem.Name
	}
	assert.Contains(t, names, "sem1")
	assert.Contains(t, names, "sem2")
}

func TestGetSemaphore(t *testing.T) {
	semaphore := &syncv1.Semaphore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sem",
			Namespace: "test-ns",
		},
		Spec: syncv1.SemaphoreSpec{
			Permits: 5,
		},
		Status: syncv1.SemaphoreStatus{
			InUse:     2,
			Available: 3,
			Phase:     syncv1.SemaphorePhaseReady,
		},
	}

	client := setupTestClient(t, semaphore)

	result, err := GetSemaphore(client, context.Background(), "test-sem")
	require.NoError(t, err)
	assert.Equal(t, "test-sem", result.Name)
	assert.Equal(t, int32(5), result.Spec.Permits)
	assert.Equal(t, int32(2), result.Status.InUse)
	assert.Equal(t, int32(3), result.Status.Available)
}

func TestGetSemaphore_NotFound(t *testing.T) {
	client := setupTestClient(t)

	_, err := GetSemaphore(client, context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get semaphore")
}

func TestAcquireSemaphore_NotImplemented(t *testing.T) {
	client := setupTestClient(t)

	_, err := AcquireSemaphore(client, context.Background(), "test-sem")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestWithSemaphore_NotImplemented(t *testing.T) {
	client := setupTestClient(t)

	err := WithSemaphore(client, context.Background(), "test-sem", func() error {
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}