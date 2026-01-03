package rwmutex

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
		WithStatusSubresource(&syncv1.RWMutex{}).
		Build()

	return konductor.NewFromClient(k8sClient, "test-ns")
}

func TestList(t *testing.T) {
	rwmutex1 := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rwmutex1",
			Namespace: "test-ns",
		},
	}

	client := setupTestClient(t, rwmutex1)

	rwmutexes, err := List(client, context.Background())
	require.NoError(t, err)
	assert.Len(t, rwmutexes, 1)
	assert.Equal(t, "rwmutex1", rwmutexes[0].Name)
}

func TestGet(t *testing.T) {
	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "test-ns",
		},
	}

	client := setupTestClient(t, rwmutex)

	result, err := Get(client, context.Background(), "test-rwmutex")
	require.NoError(t, err)
	assert.Equal(t, "test-rwmutex", result.Name)
}

func TestCreate(t *testing.T) {
	client := setupTestClient(t)

	err := Create(client, context.Background(), "test-rwmutex")
	assert.NoError(t, err)
}

func TestCreateWithTTL(t *testing.T) {
	client := setupTestClient(t)

	err := Create(client, context.Background(), "test-rwmutex", konductor.WithTTL(5*time.Minute))
	assert.NoError(t, err)

	rwmutex, err := Get(client, context.Background(), "test-rwmutex")
	require.NoError(t, err)
	assert.NotNil(t, rwmutex.Spec.TTL)
	assert.Equal(t, 5*time.Minute, rwmutex.Spec.TTL.Duration)
}

func TestDelete(t *testing.T) {
	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "test-ns",
		},
	}
	client := setupTestClient(t, rwmutex)

	err := Delete(client, context.Background(), "test-rwmutex")
	assert.NoError(t, err)
}

func TestRLock(t *testing.T) {
	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "test-ns",
		},
		Status: syncv1.RWMutexStatus{
			Phase: syncv1.RWMutexPhaseUnlocked,
		},
	}

	client := setupTestClient(t, rwmutex)

	m, err := RLock(client, context.Background(), "test-rwmutex", konductor.WithHolder("reader-1"))
	require.NoError(t, err)
	assert.Equal(t, "test-rwmutex", m.Name())
	assert.Equal(t, "reader-1", m.Holder())
	assert.True(t, m.isRead)
}

func TestRLock_MultipleReaders(t *testing.T) {
	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "test-ns",
		},
		Status: syncv1.RWMutexStatus{
			Phase:       syncv1.RWMutexPhaseReadLocked,
			ReadHolders: []string{"reader-1"},
		},
	}

	client := setupTestClient(t, rwmutex)

	m, err := RLock(client, context.Background(), "test-rwmutex", konductor.WithHolder("reader-2"))
	require.NoError(t, err)
	assert.Equal(t, "reader-2", m.Holder())

	updated, err := Get(client, context.Background(), "test-rwmutex")
	require.NoError(t, err)
	assert.Contains(t, updated.Status.ReadHolders, "reader-2")
}

func TestLock(t *testing.T) {
	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "test-ns",
		},
		Status: syncv1.RWMutexStatus{
			Phase: syncv1.RWMutexPhaseUnlocked,
		},
	}

	client := setupTestClient(t, rwmutex)

	m, err := Lock(client, context.Background(), "test-rwmutex", konductor.WithHolder("writer-1"))
	require.NoError(t, err)
	assert.Equal(t, "test-rwmutex", m.Name())
	assert.Equal(t, "writer-1", m.Holder())
	assert.False(t, m.isRead)
}

func TestRUnlock(t *testing.T) {
	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "test-ns",
		},
		Status: syncv1.RWMutexStatus{
			Phase:       syncv1.RWMutexPhaseReadLocked,
			ReadHolders: []string{"reader-1"},
		},
	}

	client := setupTestClient(t, rwmutex)

	m := &RWMutex{
		client: client,
		name:   "test-rwmutex",
		holder: "reader-1",
		isRead: true,
	}

	err := m.Unlock()
	require.NoError(t, err)

	updated, err := Get(client, context.Background(), "test-rwmutex")
	require.NoError(t, err)
	assert.Equal(t, syncv1.RWMutexPhaseUnlocked, updated.Status.Phase)
	assert.Empty(t, updated.Status.ReadHolders)
}

func TestRUnlock_MultipleReaders(t *testing.T) {
	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "test-ns",
		},
		Status: syncv1.RWMutexStatus{
			Phase:       syncv1.RWMutexPhaseReadLocked,
			ReadHolders: []string{"reader-1", "reader-2"},
		},
	}

	client := setupTestClient(t, rwmutex)

	m := &RWMutex{
		client: client,
		name:   "test-rwmutex",
		holder: "reader-1",
		isRead: true,
	}

	err := m.Unlock()
	require.NoError(t, err)

	updated, err := Get(client, context.Background(), "test-rwmutex")
	require.NoError(t, err)
	assert.Equal(t, syncv1.RWMutexPhaseReadLocked, updated.Status.Phase)
	assert.Contains(t, updated.Status.ReadHolders, "reader-2")
	assert.NotContains(t, updated.Status.ReadHolders, "reader-1")
}

func TestWUnlock(t *testing.T) {
	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "test-ns",
		},
		Status: syncv1.RWMutexStatus{
			Phase:       syncv1.RWMutexPhaseWriteLocked,
			WriteHolder: "writer-1",
		},
	}

	client := setupTestClient(t, rwmutex)

	m := &RWMutex{
		client: client,
		name:   "test-rwmutex",
		holder: "writer-1",
		isRead: false,
	}

	err := m.Unlock()
	require.NoError(t, err)

	updated, err := Get(client, context.Background(), "test-rwmutex")
	require.NoError(t, err)
	assert.Equal(t, syncv1.RWMutexPhaseUnlocked, updated.Status.Phase)
	assert.Empty(t, updated.Status.WriteHolder)
}

func TestWUnlock_NotHolder(t *testing.T) {
	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "test-ns",
		},
		Status: syncv1.RWMutexStatus{
			Phase:       syncv1.RWMutexPhaseWriteLocked,
			WriteHolder: "other-writer",
		},
	}

	client := setupTestClient(t, rwmutex)

	m := &RWMutex{
		client: client,
		name:   "test-rwmutex",
		holder: "writer-1",
		isRead: false,
	}

	err := m.Unlock()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not the holder")
}

func TestRLock_Timeout(t *testing.T) {
	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "test-ns",
		},
		Status: syncv1.RWMutexStatus{
			Phase:       syncv1.RWMutexPhaseWriteLocked,
			WriteHolder: "writer-1",
		},
	}

	client := setupTestClient(t, rwmutex)

	_, err := RLock(client, context.Background(), "test-rwmutex",
		konductor.WithHolder("reader-1"),
		konductor.WithTimeout(100*time.Millisecond))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestLock_Timeout(t *testing.T) {
	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "test-ns",
		},
		Status: syncv1.RWMutexStatus{
			Phase:       syncv1.RWMutexPhaseWriteLocked,
			WriteHolder: "other-writer",
		},
	}

	client := setupTestClient(t, rwmutex)

	_, err := Lock(client, context.Background(), "test-rwmutex",
		konductor.WithHolder("writer-1"),
		konductor.WithTimeout(100*time.Millisecond))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}
