package once

import (
	"context"
	"errors"
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
		WithStatusSubresource(&syncv1.Once{}).
		Build()

	return konductor.NewFromClient(k8sClient, "test-ns")
}

func TestList(t *testing.T) {
	once1 := &syncv1.Once{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "once1",
			Namespace: "test-ns",
		},
	}

	client := setupTestClient(t, once1)

	onces, err := List(client, context.Background())
	require.NoError(t, err)
	assert.Len(t, onces, 1)
	assert.Equal(t, "once1", onces[0].Name)
}

func TestGet(t *testing.T) {
	once := &syncv1.Once{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-once",
			Namespace: "test-ns",
		},
	}

	client := setupTestClient(t, once)

	result, err := Get(client, context.Background(), "test-once")
	require.NoError(t, err)
	assert.Equal(t, "test-once", result.Name)
}

func TestCreate(t *testing.T) {
	client := setupTestClient(t)

	err := Create(client, context.Background(), "test-once")
	assert.NoError(t, err)
}

func TestCreateWithTTL(t *testing.T) {
	client := setupTestClient(t)

	err := Create(client, context.Background(), "test-once", konductor.WithTTL(5*time.Minute))
	assert.NoError(t, err)

	once, err := Get(client, context.Background(), "test-once")
	require.NoError(t, err)
	assert.NotNil(t, once.Spec.TTL)
	assert.Equal(t, 5*time.Minute, once.Spec.TTL.Duration)
}

func TestDelete(t *testing.T) {
	once := &syncv1.Once{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-once",
			Namespace: "test-ns",
		},
	}
	client := setupTestClient(t, once)

	err := Delete(client, context.Background(), "test-once")
	assert.NoError(t, err)
}

func TestIsExecuted_NotExecuted(t *testing.T) {
	once := &syncv1.Once{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-once",
			Namespace: "test-ns",
		},
		Status: syncv1.OnceStatus{
			Executed: false,
		},
	}

	client := setupTestClient(t, once)

	executed, err := IsExecuted(client, context.Background(), "test-once")
	require.NoError(t, err)
	assert.False(t, executed)
}

func TestIsExecuted_Executed(t *testing.T) {
	once := &syncv1.Once{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-once",
			Namespace: "test-ns",
		},
		Status: syncv1.OnceStatus{
			Executed: true,
		},
	}

	client := setupTestClient(t, once)

	executed, err := IsExecuted(client, context.Background(), "test-once")
	require.NoError(t, err)
	assert.True(t, executed)
}

func TestDo_FirstExecution(t *testing.T) {
	once := &syncv1.Once{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-once",
			Namespace: "test-ns",
		},
		Status: syncv1.OnceStatus{
			Executed: false,
		},
	}

	client := setupTestClient(t, once)

	executed := false
	didExecute, err := Do(client, context.Background(), "test-once", func() error {
		executed = true
		return nil
	}, konductor.WithHolder("test-executor"))

	require.NoError(t, err)
	assert.True(t, didExecute)
	assert.True(t, executed)

	updated, err := Get(client, context.Background(), "test-once")
	require.NoError(t, err)
	assert.True(t, updated.Status.Executed)
	assert.Equal(t, "test-executor", updated.Status.Executor)
	assert.Equal(t, syncv1.OncePhaseExecuted, updated.Status.Phase)
}

func TestDo_AlreadyExecuted(t *testing.T) {
	once := &syncv1.Once{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-once",
			Namespace: "test-ns",
		},
		Status: syncv1.OnceStatus{
			Executed: true,
			Executor: "previous-executor",
		},
	}

	client := setupTestClient(t, once)

	executed := false
	didExecute, err := Do(client, context.Background(), "test-once", func() error {
		executed = true
		return nil
	}, konductor.WithHolder("test-executor"))

	require.NoError(t, err)
	assert.False(t, didExecute)
	assert.False(t, executed)
}

func TestDo_FunctionError(t *testing.T) {
	once := &syncv1.Once{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-once",
			Namespace: "test-ns",
		},
		Status: syncv1.OnceStatus{
			Executed: false,
		},
	}

	client := setupTestClient(t, once)

	expectedErr := errors.New("function failed")
	didExecute, err := Do(client, context.Background(), "test-once", func() error {
		return expectedErr
	}, konductor.WithHolder("test-executor"))

	require.Error(t, err)
	assert.True(t, didExecute)
	assert.Contains(t, err.Error(), "execution failed")
}
