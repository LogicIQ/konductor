package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

func setupMutexScheme(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))
	return scheme
}

func TestMutexReconciler_Reconcile(t *testing.T) {
	scheme := setupMutexScheme(t)

	tests := []struct {
		name          string
		mutex         *syncv1.Mutex
		expectedPhase syncv1.MutexPhase
	}{
		{
			name: "unlocked mutex",
			mutex: &syncv1.Mutex{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-mutex",
					Namespace: "default",
				},
				Spec: syncv1.MutexSpec{},
			},
			expectedPhase: syncv1.MutexPhaseUnlocked,
		},
		{
			name: "locked mutex",
			mutex: &syncv1.Mutex{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-mutex",
					Namespace: "default",
				},
				Spec: syncv1.MutexSpec{},
				Status: syncv1.MutexStatus{
					Phase:  syncv1.MutexPhaseLocked,
					Holder: "holder-1",
				},
			},
			expectedPhase: syncv1.MutexPhaseLocked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(tt.mutex).
				WithStatusSubresource(&syncv1.Mutex{}).
				Build()

			reconciler := &MutexReconciler{
				Client: client,
				Scheme: scheme,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.mutex.Name,
					Namespace: tt.mutex.Namespace,
				},
			}

			_, err := reconciler.Reconcile(context.Background(), req)
			require.NoError(t, err)

			var updated syncv1.Mutex
			err = client.Get(context.Background(), req.NamespacedName, &updated)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedPhase, updated.Status.Phase)
		})
	}
}

func TestMutexReconciler_Expiration(t *testing.T) {
	scheme := setupMutexScheme(t)

	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mutex",
			Namespace: "default",
		},
		Spec: syncv1.MutexSpec{
			TTL: &metav1.Duration{Duration: time.Hour},
		},
		Status: syncv1.MutexStatus{
			Phase:     syncv1.MutexPhaseLocked,
			Holder:    "holder-1",
			ExpiresAt: &metav1.Time{Time: time.Now().Add(-time.Hour)},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(mutex).
		WithStatusSubresource(&syncv1.Mutex{}).
		Build()

	reconciler := &MutexReconciler{
		Client: client,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      mutex.Name,
			Namespace: mutex.Namespace,
		},
	}

	_, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)

	var updated syncv1.Mutex
	err = client.Get(context.Background(), req.NamespacedName, &updated)
	require.NoError(t, err)

	assert.Equal(t, syncv1.MutexPhaseUnlocked, updated.Status.Phase)
	assert.Equal(t, "", updated.Status.Holder)
	assert.Nil(t, updated.Status.ExpiresAt)
}

func TestMutexReconciler_RequeueWithTTL(t *testing.T) {
	scheme := setupMutexScheme(t)

	expiresAt := metav1.NewTime(time.Now().Add(time.Hour))
	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mutex",
			Namespace: "default",
		},
		Spec: syncv1.MutexSpec{
			TTL: &metav1.Duration{Duration: time.Hour},
		},
		Status: syncv1.MutexStatus{
			Phase:     syncv1.MutexPhaseLocked,
			Holder:    "holder-1",
			ExpiresAt: &expiresAt,
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(mutex).
		WithStatusSubresource(&syncv1.Mutex{}).
		Build()

	reconciler := &MutexReconciler{
		Client: client,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      mutex.Name,
			Namespace: mutex.Namespace,
		},
	}

	result, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, result.RequeueAfter > 0)
}
