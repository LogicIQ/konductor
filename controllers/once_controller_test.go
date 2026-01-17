package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

func TestOnceReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	tests := []struct {
		name          string
		once          *syncv1.Once
		expectedPhase syncv1.OncePhase
	}{
		{
			name: "pending once",
			once: &syncv1.Once{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-once",
					Namespace: "default",
				},
				Spec: syncv1.OnceSpec{},
			},
			expectedPhase: syncv1.OncePhasePending,
		},
		{
			name: "executed once",
			once: &syncv1.Once{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-once",
					Namespace: "default",
				},
				Spec: syncv1.OnceSpec{},
				Status: syncv1.OnceStatus{
					Phase:    syncv1.OncePhasePending,
					Executed: true,
					Executor: "pod-1",
				},
			},
			expectedPhase: syncv1.OncePhaseExecuted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(tt.once).
				WithStatusSubresource(&syncv1.Once{}).
				Build()

			reconciler := &OnceReconciler{
				Client: client,
				Scheme: scheme,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.once.Name,
					Namespace: tt.once.Namespace,
				},
			}

			_, err := reconciler.Reconcile(context.Background(), req)
			require.NoError(t, err)

			var updated syncv1.Once
			err = client.Get(context.Background(), req.NamespacedName, &updated)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedPhase, updated.Status.Phase)
		})
	}
}
