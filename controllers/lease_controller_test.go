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

func TestLeaseReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	tests := []struct {
		name           string
		lease          *syncv1.Lease
		requests       []syncv1.LeaseRequest
		expectedPhase  syncv1.LeasePhase
		expectedHolder string
	}{
		{
			name: "available lease with no requests",
			lease: &syncv1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-lease",
					Namespace: "default",
				},
				Spec: syncv1.LeaseSpec{
					TTL: &metav1.Duration{Duration: time.Hour},
				},
			},
			requests:       []syncv1.LeaseRequest{},
			expectedPhase:  syncv1.LeasePhaseAvailable,
			expectedHolder: "",
		},
		{
			name: "available lease should grant to requester",
			lease: &syncv1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-lease",
					Namespace: "default",
				},
				Spec: syncv1.LeaseSpec{
					TTL: &metav1.Duration{Duration: time.Hour},
				},
			},
			requests: []syncv1.LeaseRequest{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "request-1",
						Namespace: "default",
						Labels:    map[string]string{"lease": "test-lease"},
					},
					Spec: syncv1.LeaseRequestSpec{
						Lease:  "test-lease",
						Holder: "holder-1",
					},
				},
			},
			expectedPhase:  syncv1.LeasePhaseHeld,
			expectedHolder: "holder-1",
		},
		{
			name: "lease should grant to highest priority",
			lease: &syncv1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-lease",
					Namespace: "default",
				},
				Spec: syncv1.LeaseSpec{
					TTL: &metav1.Duration{Duration: time.Hour},
				},
			},
			requests: []syncv1.LeaseRequest{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "request-1",
						Namespace: "default",
						Labels:    map[string]string{"lease": "test-lease"},
					},
					Spec: syncv1.LeaseRequestSpec{
						Lease:    "test-lease",
						Holder:   "holder-1",
						Priority: int32Ptr(1),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "request-2",
						Namespace: "default",
						Labels:    map[string]string{"lease": "test-lease"},
					},
					Spec: syncv1.LeaseRequestSpec{
						Lease:    "test-lease",
						Holder:   "holder-2",
						Priority: int32Ptr(5),
					},
				},
			},
			expectedPhase:  syncv1.LeasePhaseHeld,
			expectedHolder: "holder-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs := []runtime.Object{tt.lease}
			for i := range tt.requests {
				objs = append(objs, &tt.requests[i])
			}

			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(objs...).
				WithStatusSubresource(&syncv1.Lease{}, &syncv1.LeaseRequest{}).
				Build()

			reconciler := &LeaseReconciler{
				Client: client,
				Scheme: scheme,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.lease.Name,
					Namespace: tt.lease.Namespace,
				},
			}

			result, err := reconciler.Reconcile(context.Background(), req)
			require.NoError(t, err)

			var updated syncv1.Lease
			err = client.Get(context.Background(), req.NamespacedName, &updated)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedPhase, updated.Status.Phase)
			assert.Equal(t, tt.expectedHolder, updated.Status.Holder)

			if tt.expectedPhase == syncv1.LeasePhaseHeld {
				assert.NotNil(t, updated.Status.AcquiredAt)
				assert.NotNil(t, updated.Status.ExpiresAt)
				assert.True(t, result.RequeueAfter > 0)
			}
		})
	}
}

func TestLeaseReconciler_Expiration(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	lease := &syncv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease",
			Namespace: "default",
		},
		Spec: syncv1.LeaseSpec{
			TTL: &metav1.Duration{Duration: time.Hour},
		},
		Status: syncv1.LeaseStatus{
			Phase:     syncv1.LeasePhaseHeld,
			Holder:    "holder-1",
			ExpiresAt: &metav1.Time{Time: time.Now().Add(-time.Hour)},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(lease).
		WithStatusSubresource(&syncv1.Lease{}).
		Build()

	reconciler := &LeaseReconciler{
		Client: client,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      lease.Name,
			Namespace: lease.Namespace,
		},
	}

	_, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)

	var updated syncv1.Lease
	err = client.Get(context.Background(), req.NamespacedName, &updated)
	require.NoError(t, err)

	assert.Equal(t, syncv1.LeasePhaseAvailable, updated.Status.Phase)
	assert.Equal(t, "", updated.Status.Holder)
	assert.Nil(t, updated.Status.ExpiresAt)
}
