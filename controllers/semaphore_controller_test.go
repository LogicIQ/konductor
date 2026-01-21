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

func TestSemaphoreReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	tests := []struct {
		name          string
		semaphore     *syncv1.Semaphore
		permits       []syncv1.Permit
		expectedPhase syncv1.SemaphorePhase
		expectedInUse int32
		expectedAvail int32
	}{
		{
			name: "empty semaphore should be ready",
			semaphore: &syncv1.Semaphore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-sem",
					Namespace: "default",
				},
				Spec: syncv1.SemaphoreSpec{
					Permits: 5,
				},
			},
			permits:       []syncv1.Permit{},
			expectedPhase: syncv1.SemaphorePhaseReady,
			expectedInUse: 0,
			expectedAvail: 5,
		},
		{
			name: "partial usage should be ready",
			semaphore: &syncv1.Semaphore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-sem",
					Namespace: "default",
				},
				Spec: syncv1.SemaphoreSpec{
					Permits: 5,
				},
			},
			permits: []syncv1.Permit{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "permit-1",
						Namespace: "default",
						Labels:    map[string]string{"semaphore": "test-sem"},
					},
					Spec: syncv1.PermitSpec{
						Semaphore: "test-sem",
						Holder:    "holder-1",
					},
				},
			},
			expectedPhase: syncv1.SemaphorePhaseReady,
			expectedInUse: 1,
			expectedAvail: 4,
		},
		{
			name: "full semaphore should be full",
			semaphore: &syncv1.Semaphore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-sem",
					Namespace: "default",
				},
				Spec: syncv1.SemaphoreSpec{
					Permits: 2,
				},
			},
			permits: []syncv1.Permit{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "permit-1",
						Namespace: "default",
						Labels:    map[string]string{"semaphore": "test-sem"},
					},
					Spec: syncv1.PermitSpec{
						Semaphore: "test-sem",
						Holder:    "holder-1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "permit-2",
						Namespace: "default",
						Labels:    map[string]string{"semaphore": "test-sem"},
					},
					Spec: syncv1.PermitSpec{
						Semaphore: "test-sem",
						Holder:    "holder-2",
					},
				},
			},
			expectedPhase: syncv1.SemaphorePhaseFull,
			expectedInUse: 2,
			expectedAvail: 0,
		},
		{
			name: "expired permits should not count",
			semaphore: &syncv1.Semaphore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-sem",
					Namespace: "default",
				},
				Spec: syncv1.SemaphoreSpec{
					Permits: 3,
				},
			},
			permits: []syncv1.Permit{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "permit-1",
						Namespace: "default",
						Labels:    map[string]string{"semaphore": "test-sem"},
					},
					Spec: syncv1.PermitSpec{
						Semaphore: "test-sem",
						Holder:    "holder-1",
					},
					Status: syncv1.PermitStatus{
						ExpiresAt: &metav1.Time{Time: time.Now().Add(-time.Hour)},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "permit-2",
						Namespace: "default",
						Labels:    map[string]string{"semaphore": "test-sem"},
					},
					Spec: syncv1.PermitSpec{
						Semaphore: "test-sem",
						Holder:    "holder-2",
					},
					Status: syncv1.PermitStatus{
						ExpiresAt: &metav1.Time{Time: time.Now().Add(time.Hour)},
					},
				},
			},
			expectedPhase: syncv1.SemaphorePhaseReady,
			expectedInUse: 1,
			expectedAvail: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs := []runtime.Object{tt.semaphore}
			for i := range tt.permits {
				objs = append(objs, &tt.permits[i])
			}

			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(objs...).
				WithStatusSubresource(&syncv1.Semaphore{}).
				Build()

			reconciler := &SemaphoreReconciler{
				Client: client,
				Scheme: scheme,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.semaphore.Name,
					Namespace: tt.semaphore.Namespace,
				},
			}

			// First reconcile initializes the status
			_, err := reconciler.Reconcile(context.Background(), req)
			require.NoError(t, err)

			// Second reconcile verifies steady-state behavior
			result, err := reconciler.Reconcile(context.Background(), req)
			require.NoError(t, err)
			assert.Equal(t, time.Minute, result.RequeueAfter)

			var updated syncv1.Semaphore
			err = client.Get(context.Background(), req.NamespacedName, &updated)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedPhase, updated.Status.Phase)
			assert.Equal(t, tt.expectedInUse, updated.Status.InUse)
			assert.Equal(t, tt.expectedAvail, updated.Status.Available)
		})
	}
}

func TestSemaphoreReconciler_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := &SemaphoreReconciler{
		Client: client,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "nonexistent",
			Namespace: "default",
		},
	}

	result, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}
