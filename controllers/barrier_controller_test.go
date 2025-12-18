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

func TestBarrierReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	tests := []struct {
		name          string
		barrier       *syncv1.Barrier
		arrivals      []syncv1.Arrival
		expectedPhase syncv1.BarrierPhase
		expectedCount int32
	}{
		{
			name: "waiting barrier with no arrivals",
			barrier: &syncv1.Barrier{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-barrier",
					Namespace: "default",
				},
				Spec: syncv1.BarrierSpec{
					Expected: 3,
				},
			},
			arrivals:      []syncv1.Arrival{},
			expectedPhase: syncv1.BarrierPhaseWaiting,
			expectedCount: 0,
		},
		{
			name: "barrier with partial arrivals",
			barrier: &syncv1.Barrier{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-barrier",
					Namespace: "default",
				},
				Spec: syncv1.BarrierSpec{
					Expected: 3,
				},
			},
			arrivals: []syncv1.Arrival{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "arrival-1",
						Namespace: "default",
						Labels:    map[string]string{"barrier": "test-barrier"},
					},
					Spec: syncv1.ArrivalSpec{
						Barrier: "test-barrier",
						Holder:  "holder-1",
					},
				},
			},
			expectedPhase: syncv1.BarrierPhaseWaiting,
			expectedCount: 1,
		},
		{
			name: "barrier should open when expected arrivals reached",
			barrier: &syncv1.Barrier{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-barrier",
					Namespace: "default",
				},
				Spec: syncv1.BarrierSpec{
					Expected: 2,
				},
			},
			arrivals: []syncv1.Arrival{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "arrival-1",
						Namespace: "default",
						Labels:    map[string]string{"barrier": "test-barrier"},
					},
					Spec: syncv1.ArrivalSpec{
						Barrier: "test-barrier",
						Holder:  "holder-1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "arrival-2",
						Namespace: "default",
						Labels:    map[string]string{"barrier": "test-barrier"},
					},
					Spec: syncv1.ArrivalSpec{
						Barrier: "test-barrier",
						Holder:  "holder-2",
					},
				},
			},
			expectedPhase: syncv1.BarrierPhaseOpen,
			expectedCount: 2,
		},
		{
			name: "barrier with quorum should open early",
			barrier: &syncv1.Barrier{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-barrier",
					Namespace: "default",
				},
				Spec: syncv1.BarrierSpec{
					Expected: 5,
					Quorum:   &[]int32{2}[0],
				},
			},
			arrivals: []syncv1.Arrival{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "arrival-1",
						Namespace: "default",
						Labels:    map[string]string{"barrier": "test-barrier"},
					},
					Spec: syncv1.ArrivalSpec{
						Barrier: "test-barrier",
						Holder:  "holder-1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "arrival-2",
						Namespace: "default",
						Labels:    map[string]string{"barrier": "test-barrier"},
					},
					Spec: syncv1.ArrivalSpec{
						Barrier: "test-barrier",
						Holder:  "holder-2",
					},
				},
			},
			expectedPhase: syncv1.BarrierPhaseOpen,
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs := []runtime.Object{tt.barrier}
			for i := range tt.arrivals {
				objs = append(objs, &tt.arrivals[i])
			}

			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(objs...).
				WithStatusSubresource(&syncv1.Barrier{}).
				Build()

			reconciler := &BarrierReconciler{
				Client: client,
				Scheme: scheme,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.barrier.Name,
					Namespace: tt.barrier.Namespace,
				},
			}

			result, err := reconciler.Reconcile(context.Background(), req)
			require.NoError(t, err)

			var updated syncv1.Barrier
			err = client.Get(context.Background(), req.NamespacedName, &updated)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedPhase, updated.Status.Phase)
			assert.Equal(t, tt.expectedCount, updated.Status.Arrived)
			assert.Len(t, updated.Status.Arrivals, int(tt.expectedCount))

			if tt.expectedPhase == syncv1.BarrierPhaseOpen {
				assert.NotNil(t, updated.Status.OpenedAt)
			}

			if tt.barrier.Spec.Timeout != nil && tt.expectedPhase == syncv1.BarrierPhaseWaiting {
				assert.Equal(t, time.Minute, result.RequeueAfter)
			}
		})
	}
}

func TestBarrierReconciler_Timeout(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-barrier",
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(time.Now().Add(-2 * time.Hour)),
		},
		Spec: syncv1.BarrierSpec{
			Expected: 3,
			Timeout:  &metav1.Duration{Duration: time.Hour},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(barrier).
		WithStatusSubresource(&syncv1.Barrier{}).
		Build()

	reconciler := &BarrierReconciler{
		Client: client,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      barrier.Name,
			Namespace: barrier.Namespace,
		},
	}

	_, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)

	var updated syncv1.Barrier
	err = client.Get(context.Background(), req.NamespacedName, &updated)
	require.NoError(t, err)

	assert.Equal(t, syncv1.BarrierPhaseFailed, updated.Status.Phase)
}