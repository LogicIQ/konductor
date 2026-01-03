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

func TestRWMutexReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	tests := []struct {
		name          string
		rwmutex       *syncv1.RWMutex
		expectedPhase syncv1.RWMutexPhase
	}{
		{
			name: "unlocked rwmutex",
			rwmutex: &syncv1.RWMutex{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-rwmutex",
					Namespace: "default",
				},
				Spec: syncv1.RWMutexSpec{},
			},
			expectedPhase: syncv1.RWMutexPhaseUnlocked,
		},
		{
			name: "read locked rwmutex",
			rwmutex: &syncv1.RWMutex{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-rwmutex",
					Namespace: "default",
				},
				Spec: syncv1.RWMutexSpec{},
				Status: syncv1.RWMutexStatus{
					Phase:       syncv1.RWMutexPhaseReadLocked,
					ReadHolders: []string{"reader-1", "reader-2"},
				},
			},
			expectedPhase: syncv1.RWMutexPhaseReadLocked,
		},
		{
			name: "write locked rwmutex",
			rwmutex: &syncv1.RWMutex{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-rwmutex",
					Namespace: "default",
				},
				Spec: syncv1.RWMutexSpec{},
				Status: syncv1.RWMutexStatus{
					Phase:       syncv1.RWMutexPhaseWriteLocked,
					WriteHolder: "writer-1",
				},
			},
			expectedPhase: syncv1.RWMutexPhaseWriteLocked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(tt.rwmutex).
				WithStatusSubresource(&syncv1.RWMutex{}).
				Build()

			reconciler := &RWMutexReconciler{
				Client: client,
				Scheme: scheme,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.rwmutex.Name,
					Namespace: tt.rwmutex.Namespace,
				},
			}

			_, err := reconciler.Reconcile(context.Background(), req)
			require.NoError(t, err)

			var updated syncv1.RWMutex
			err = client.Get(context.Background(), req.NamespacedName, &updated)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedPhase, updated.Status.Phase)
		})
	}
}

func TestRWMutexReconciler_Expiration(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "default",
		},
		Spec: syncv1.RWMutexSpec{
			TTL: &metav1.Duration{Duration: time.Hour},
		},
		Status: syncv1.RWMutexStatus{
			Phase:       syncv1.RWMutexPhaseWriteLocked,
			WriteHolder: "writer-1",
			ExpiresAt:   &metav1.Time{Time: time.Now().Add(-time.Hour)},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(rwmutex).
		WithStatusSubresource(&syncv1.RWMutex{}).
		Build()

	reconciler := &RWMutexReconciler{
		Client: client,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      rwmutex.Name,
			Namespace: rwmutex.Namespace,
		},
	}

	_, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)

	var updated syncv1.RWMutex
	err = client.Get(context.Background(), req.NamespacedName, &updated)
	require.NoError(t, err)

	assert.Equal(t, syncv1.RWMutexPhaseUnlocked, updated.Status.Phase)
	assert.Equal(t, "", updated.Status.WriteHolder)
	assert.Nil(t, updated.Status.ReadHolders)
	assert.Nil(t, updated.Status.ExpiresAt)
}

func TestRWMutexReconciler_RequeueWithTTL(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	expiresAt := metav1.NewTime(time.Now().Add(time.Hour))
	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "default",
		},
		Spec: syncv1.RWMutexSpec{
			TTL: &metav1.Duration{Duration: time.Hour},
		},
		Status: syncv1.RWMutexStatus{
			Phase:       syncv1.RWMutexPhaseReadLocked,
			ReadHolders: []string{"reader-1"},
			ExpiresAt:   &expiresAt,
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(rwmutex).
		WithStatusSubresource(&syncv1.RWMutex{}).
		Build()

	reconciler := &RWMutexReconciler{
		Client: client,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      rwmutex.Name,
			Namespace: rwmutex.Namespace,
		},
	}

	result, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, result.RequeueAfter > 0)
}
