package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

func int32Ptr(v int32) *int32 {
	return &v
}

func TestGateReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	tests := []struct {
		name          string
		gate          *syncv1.Gate
		objects       []runtime.Object
		expectedPhase syncv1.GatePhase
		expectedMet   int
	}{
		{
			name: "gate with no conditions should open",
			gate: &syncv1.Gate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-gate",
					Namespace: "default",
				},
				Spec: syncv1.GateSpec{
					Conditions: []syncv1.GateCondition{},
				},
			},
			objects:       []runtime.Object{},
			expectedPhase: syncv1.GatePhaseOpen,
			expectedMet:   0,
		},
		{
			name: "gate waiting for job completion",
			gate: &syncv1.Gate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-gate",
					Namespace: "default",
				},
				Spec: syncv1.GateSpec{
					Conditions: []syncv1.GateCondition{
						{
							Type:  "Job",
							Name:  "test-job",
							State: "Complete",
						},
					},
				},
			},
			objects: []runtime.Object{
				&batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-job",
						Namespace: "default",
					},
					Status: batchv1.JobStatus{
						Succeeded: 0,
					},
				},
			},
			expectedPhase: syncv1.GatePhaseWaiting,
			expectedMet:   0,
		},
		{
			name: "gate should open when job completes",
			gate: &syncv1.Gate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-gate",
					Namespace: "default",
				},
				Spec: syncv1.GateSpec{
					Conditions: []syncv1.GateCondition{
						{
							Type:  "Job",
							Name:  "test-job",
							State: "Complete",
						},
					},
				},
			},
			objects: []runtime.Object{
				&batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-job",
						Namespace: "default",
					},
					Status: batchv1.JobStatus{
						Succeeded: 1,
					},
				},
			},
			expectedPhase: syncv1.GatePhaseOpen,
			expectedMet:   1,
		},
		{
			name: "gate waiting for semaphore permits",
			gate: &syncv1.Gate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-gate",
					Namespace: "default",
				},
				Spec: syncv1.GateSpec{
					Conditions: []syncv1.GateCondition{
						{
							Type:  "Semaphore",
							Name:  "test-sem",
							Value: int32Ptr(3),
						},
					},
				},
			},
			objects: []runtime.Object{
				&syncv1.Semaphore{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-sem",
						Namespace: "default",
					},
					Status: syncv1.SemaphoreStatus{
						Available: 2,
					},
				},
			},
			expectedPhase: syncv1.GatePhaseWaiting,
			expectedMet:   0,
		},
		{
			name: "gate should open when semaphore has enough permits",
			gate: &syncv1.Gate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-gate",
					Namespace: "default",
				},
				Spec: syncv1.GateSpec{
					Conditions: []syncv1.GateCondition{
						{
							Type:  "Semaphore",
							Name:  "test-sem",
							Value: int32Ptr(3),
						},
					},
				},
			},
			objects: []runtime.Object{
				&syncv1.Semaphore{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-sem",
						Namespace: "default",
					},
					Status: syncv1.SemaphoreStatus{
						Available: 5,
					},
				},
			},
			expectedPhase: syncv1.GatePhaseOpen,
			expectedMet:   1,
		},
		{
			name: "gate waiting for barrier to open",
			gate: &syncv1.Gate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-gate",
					Namespace: "default",
				},
				Spec: syncv1.GateSpec{
					Conditions: []syncv1.GateCondition{
						{
							Type:  "Barrier",
							Name:  "test-barrier",
							State: "Open",
						},
					},
				},
			},
			objects: []runtime.Object{
				&syncv1.Barrier{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-barrier",
						Namespace: "default",
					},
					Status: syncv1.BarrierStatus{
						Phase: syncv1.BarrierPhaseWaiting,
					},
				},
			},
			expectedPhase: syncv1.GatePhaseWaiting,
			expectedMet:   0,
		},
		{
			name: "gate should open when barrier opens",
			gate: &syncv1.Gate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-gate",
					Namespace: "default",
				},
				Spec: syncv1.GateSpec{
					Conditions: []syncv1.GateCondition{
						{
							Type:  "Barrier",
							Name:  "test-barrier",
							State: "Open",
						},
					},
				},
			},
			objects: []runtime.Object{
				&syncv1.Barrier{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-barrier",
						Namespace: "default",
					},
					Status: syncv1.BarrierStatus{
						Phase: syncv1.BarrierPhaseOpen,
					},
				},
			},
			expectedPhase: syncv1.GatePhaseOpen,
			expectedMet:   1,
		},
		{
			name: "gate waiting for lease availability",
			gate: &syncv1.Gate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-gate",
					Namespace: "default",
				},
				Spec: syncv1.GateSpec{
					Conditions: []syncv1.GateCondition{
						{
							Type:  "Lease",
							Name:  "test-lease",
							State: "Available",
						},
					},
				},
			},
			objects: []runtime.Object{
				&syncv1.Lease{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-lease",
						Namespace: "default",
					},
					Status: syncv1.LeaseStatus{
						Phase: syncv1.LeasePhaseHeld,
					},
				},
			},
			expectedPhase: syncv1.GatePhaseWaiting,
			expectedMet:   0,
		},
		{
			name: "gate should open when lease becomes available",
			gate: &syncv1.Gate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-gate",
					Namespace: "default",
				},
				Spec: syncv1.GateSpec{
					Conditions: []syncv1.GateCondition{
						{
							Type:  "Lease",
							Name:  "test-lease",
							State: "Available",
						},
					},
				},
			},
			objects: []runtime.Object{
				&syncv1.Lease{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-lease",
						Namespace: "default",
					},
					Status: syncv1.LeaseStatus{
						Phase: syncv1.LeasePhaseAvailable,
					},
				},
			},
			expectedPhase: syncv1.GatePhaseOpen,
			expectedMet:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs := []runtime.Object{tt.gate}
			objs = append(objs, tt.objects...)

			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(objs...).
				WithStatusSubresource(&syncv1.Gate{}).
				Build()

			reconciler := &GateReconciler{
				Client: client,
				Scheme: scheme,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.gate.Name,
					Namespace: tt.gate.Namespace,
				},
			}

			result, err := reconciler.Reconcile(context.Background(), req)
			require.NoError(t, err)

			var updated syncv1.Gate
			err = client.Get(context.Background(), req.NamespacedName, &updated)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedPhase, updated.Status.Phase)
			assert.Len(t, updated.Status.ConditionStatuses, len(tt.gate.Spec.Conditions))

			metCount := 0
			for _, status := range updated.Status.ConditionStatuses {
				if status.Met {
					metCount++
				}
			}
			assert.Equal(t, tt.expectedMet, metCount)

			if tt.expectedPhase == syncv1.GatePhaseOpen {
				assert.NotNil(t, updated.Status.OpenedAt)
			}

			if tt.expectedPhase == syncv1.GatePhaseWaiting {
				assert.Equal(t, 10*time.Second, result.RequeueAfter)
			}
		})
	}
}

func TestGateReconciler_Timeout(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-gate",
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(time.Now().Add(-2 * time.Hour)),
		},
		Spec: syncv1.GateSpec{
			Timeout: &metav1.Duration{Duration: time.Hour},
			Conditions: []syncv1.GateCondition{
				{
					Type:  "Job",
					Name:  "nonexistent-job",
					State: "Complete",
				},
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gate).
		WithStatusSubresource(&syncv1.Gate{}).
		Build()

	reconciler := &GateReconciler{
		Client: client,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      gate.Name,
			Namespace: gate.Namespace,
		},
	}

	_, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)

	var updated syncv1.Gate
	err = client.Get(context.Background(), req.NamespacedName, &updated)
	require.NoError(t, err)

	assert.Equal(t, syncv1.GatePhaseFailed, updated.Status.Phase)
}
