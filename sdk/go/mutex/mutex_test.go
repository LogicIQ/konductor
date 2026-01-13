package mutex

import (
	"context"
	"testing"
	"time"

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
		WithStatusSubresource(&syncv1.Mutex{}).
		Build()

	return konductor.NewFromClient(k8sClient, "test-ns")
}

func TestList(t *testing.T) {
	mutex1 := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mutex1",
			Namespace: "test-ns",
		},
		Spec: syncv1.MutexSpec{},
	}

	client := setupTestClient(t, mutex1)

	mutexes, err := List(client, context.Background())
	require.NoError(t, err)
	assert.Len(t, mutexes, 1)
	assert.Equal(t, "mutex1", mutexes[0].Name)
}

func TestGet(t *testing.T) {
	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mutex",
			Namespace: "test-ns",
		},
		Spec: syncv1.MutexSpec{},
	}

	client := setupTestClient(t, mutex)

	result, err := Get(client, context.Background(), "test-mutex")
	require.NoError(t, err)
	assert.Equal(t, "test-mutex", result.Name)
}

func TestCreate(t *testing.T) {
	client := setupTestClient(t)

	err := Create(client, context.Background(), "test-mutex")
	assert.NoError(t, err)
}

func TestCreateWithTTL(t *testing.T) {
	client := setupTestClient(t)

	err := Create(client, context.Background(), "test-mutex", konductor.WithTTL(5*time.Minute))
	assert.NoError(t, err)

	mutex, err := Get(client, context.Background(), "test-mutex")
	require.NoError(t, err)
	assert.NotNil(t, mutex.Spec.TTL)
	assert.Equal(t, 5*time.Minute, mutex.Spec.TTL.Duration)
}

func TestDelete(t *testing.T) {
	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mutex",
			Namespace: "test-ns",
		},
	}
	client := setupTestClient(t, mutex)

	err := Delete(client, context.Background(), "test-mutex")
	assert.NoError(t, err)
}

func TestIsLocked_Locked(t *testing.T) {
	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mutex",
			Namespace: "test-ns",
		},
		Status: syncv1.MutexStatus{
			Phase:  syncv1.MutexPhaseLocked,
			Holder: "holder-1",
		},
	}

	client := setupTestClient(t, mutex)

	locked, err := IsLocked(client, context.Background(), "test-mutex")
	require.NoError(t, err)
	assert.True(t, locked)
}

func TestIsLocked_Unlocked(t *testing.T) {
	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mutex",
			Namespace: "test-ns",
		},
		Status: syncv1.MutexStatus{
			Phase: syncv1.MutexPhaseUnlocked,
		},
	}

	client := setupTestClient(t, mutex)

	locked, err := IsLocked(client, context.Background(), "test-mutex")
	require.NoError(t, err)
	assert.False(t, locked)
}

func TestLock(t *testing.T) {
	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mutex",
			Namespace: "test-ns",
		},
		Status: syncv1.MutexStatus{
			Phase: syncv1.MutexPhaseUnlocked,
		},
	}

	client := setupTestClient(t, mutex)

	m, err := Lock(client, context.Background(), "test-mutex", konductor.WithHolder("test-holder"))
	require.NoError(t, err)
	assert.Equal(t, "test-mutex", m.Name())
	assert.Equal(t, "test-holder", m.Holder())
}

func TestUnlock(t *testing.T) {
	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mutex",
			Namespace: "test-ns",
		},
		Status: syncv1.MutexStatus{
			Phase:  syncv1.MutexPhaseLocked,
			Holder: "test-holder",
		},
	}

	client := setupTestClient(t, mutex)

	m := &Mutex{
		client: client,
		name:   "test-mutex",
		holder: "test-holder",
	}

	err := m.Unlock(context.Background())
	require.NoError(t, err)

	updated, err := Get(client, context.Background(), "test-mutex")
	require.NoError(t, err)
	assert.Equal(t, syncv1.MutexPhaseUnlocked, updated.Status.Phase)
	assert.Equal(t, "", updated.Status.Holder)
}

func TestUnlock_NotHolder(t *testing.T) {
	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mutex",
			Namespace: "test-ns",
		},
		Status: syncv1.MutexStatus{
			Phase:  syncv1.MutexPhaseLocked,
			Holder: "other-holder",
		},
	}

	client := setupTestClient(t, mutex)

	m := &Mutex{
		client: client,
		name:   "test-mutex",
		holder: "test-holder",
	}

	err := m.Unlock(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not the holder")
}

func TestTryLock_Available(t *testing.T) {
	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mutex",
			Namespace: "test-ns",
		},
		Status: syncv1.MutexStatus{
			Phase: syncv1.MutexPhaseUnlocked,
		},
	}

	client := setupTestClient(t, mutex)

	m, err := TryLock(client, context.Background(), "test-mutex", konductor.WithHolder("test-holder"))
	require.NoError(t, err)
	assert.NotNil(t, m)
}

func TestTryLock_Locked(t *testing.T) {
	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mutex",
			Namespace: "test-ns",
		},
		Status: syncv1.MutexStatus{
			Phase:  syncv1.MutexPhaseLocked,
			Holder: "other-holder",
		},
	}

	client := setupTestClient(t, mutex)

	_, err := TryLock(client, context.Background(), "test-mutex", konductor.WithHolder("test-holder"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestWith(t *testing.T) {
	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mutex",
			Namespace: "test-ns",
		},
		Status: syncv1.MutexStatus{
			Phase: syncv1.MutexPhaseUnlocked,
		},
	}

	client := setupTestClient(t, mutex)

	executed := false
	err := With(client, context.Background(), "test-mutex", func() error {
		executed = true
		return nil
	}, konductor.WithHolder("test-holder"))

	require.NoError(t, err)
	assert.True(t, executed)

	// Verify mutex is unlocked after
	updated, err := Get(client, context.Background(), "test-mutex")
	require.NoError(t, err)
	assert.Equal(t, syncv1.MutexPhaseUnlocked, updated.Status.Phase)
}
