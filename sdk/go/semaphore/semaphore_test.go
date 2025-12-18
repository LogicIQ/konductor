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

func TestList(t *testing.T) {
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

	client := setupTestClient(t, semaphore1)

	semaphores, err := List(client, context.Background())
	require.NoError(t, err)
	assert.Len(t, semaphores, 1)
	assert.Equal(t, "sem1", semaphores[0].Name)
}

func TestGet(t *testing.T) {
	semaphore := &syncv1.Semaphore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sem",
			Namespace: "test-ns",
		},
		Spec: syncv1.SemaphoreSpec{
			Permits: 5,
		},
	}

	client := setupTestClient(t, semaphore)

	result, err := Get(client, context.Background(), "test-sem")
	require.NoError(t, err)
	assert.Equal(t, "test-sem", result.Name)
	assert.Equal(t, int32(5), result.Spec.Permits)
}

func TestCreate(t *testing.T) {
	client := setupTestClient(t)

	err := Create(client, context.Background(), "test-sem", 5)
	assert.NoError(t, err)
}

func TestDelete(t *testing.T) {
	semaphore := &syncv1.Semaphore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sem",
			Namespace: "test-ns",
		},
	}
	client := setupTestClient(t, semaphore)

	err := Delete(client, context.Background(), "test-sem")
	assert.NoError(t, err)
}

func TestUpdate(t *testing.T) {
	semaphore := &syncv1.Semaphore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sem",
			Namespace: "test-ns",
		},
		Spec: syncv1.SemaphoreSpec{
			Permits: 5,
		},
	}
	client := setupTestClient(t, semaphore)

	// Update permits
	semaphore.Spec.Permits = 10
	err := Update(client, context.Background(), semaphore)
	assert.NoError(t, err)
}

