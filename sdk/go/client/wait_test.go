package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

func TestRetryWithBackoff_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	client := NewFromClient(k8sClient, "default")

	callCount := 0
	err := client.RetryWithBackoff(context.Background(), func() error {
		callCount++
		if callCount < 2 {
			return apierrors.NewConflict(schema.GroupResource{}, "test", errors.New("conflict"))
		}
		return nil
	}, &WaitConfig{
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Factor:       1.5,
		Timeout:      1 * time.Second,
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestRetryWithBackoff_NonConflictError(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	client := NewFromClient(k8sClient, "default")

	callCount := 0
	err := client.RetryWithBackoff(context.Background(), func() error {
		callCount++
		return errors.New("non-conflict error")
	}, &WaitConfig{
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Factor:       1.5,
		Timeout:      1 * time.Second,
	})

	assert.Error(t, err)
	assert.Equal(t, 1, callCount) // Should not retry non-conflict errors
}

func TestWaitForCondition_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	wg := &syncv1.WaitGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wg",
			Namespace: "default",
		},
		Status: syncv1.WaitGroupStatus{
			Counter: 0, // Start with condition already met
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(wg).
		WithStatusSubresource(&syncv1.WaitGroup{}).
		Build()
	client := NewFromClient(k8sClient, "default")

	err := client.WaitForCondition(context.Background(), wg, func(obj interface{}) bool {
		waitGroup := obj.(*syncv1.WaitGroup)
		return waitGroup.Status.Counter == 0 // Condition is already met
	}, &WaitConfig{
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		Factor:        1.5,
		Timeout:       1 * time.Second,
		OperatorDelay: 10 * time.Millisecond, // Short delay for test
	})

	assert.NoError(t, err)
}

func TestWaitForCondition_Timeout(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	wg := &syncv1.WaitGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wg",
			Namespace: "default",
		},
		Status: syncv1.WaitGroupStatus{
			Counter: 1,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(wg).
		Build()
	client := NewFromClient(k8sClient, "default")

	err := client.WaitForCondition(context.Background(), wg, func(obj interface{}) bool {
		return false // Never satisfied
	}, &WaitConfig{
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      20 * time.Millisecond,
		Factor:        1.5,
		Timeout:       100 * time.Millisecond, // Short timeout
		OperatorDelay: 10 * time.Millisecond,
	})

	assert.Error(t, err)
}

func TestDefaultWaitConfig(t *testing.T) {
	config := DefaultWaitConfig()

	assert.Equal(t, 500*time.Millisecond, config.InitialDelay)
	assert.Equal(t, 5*time.Second, config.MaxDelay)
	assert.Equal(t, 1.5, config.Factor)
	assert.Equal(t, 0.1, config.Jitter)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 2*time.Second, config.OperatorDelay)
}
