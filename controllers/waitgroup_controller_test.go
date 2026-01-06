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

func TestWaitGroupReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	tests := []struct {
		name          string
		wg            *syncv1.WaitGroup
		expectedPhase syncv1.WaitGroupPhase
	}{
		{
			name: "waiting waitgroup",
			wg: &syncv1.WaitGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-wg",
					Namespace: "default",
				},
				Spec: syncv1.WaitGroupSpec{},
				Status: syncv1.WaitGroupStatus{
					Counter: 3,
				},
			},
			expectedPhase: syncv1.WaitGroupPhaseWaiting,
		},
		{
			name: "done waitgroup",
			wg: &syncv1.WaitGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-wg",
					Namespace: "default",
				},
				Spec: syncv1.WaitGroupSpec{},
				Status: syncv1.WaitGroupStatus{
					Counter: 0,
				},
			},
			expectedPhase: syncv1.WaitGroupPhaseDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(tt.wg).
				WithStatusSubresource(&syncv1.WaitGroup{}).
				Build()

			reconciler := &WaitGroupReconciler{
				Client: client,
				Scheme: scheme,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.wg.Name,
					Namespace: tt.wg.Namespace,
				},
			}

			_, err := reconciler.Reconcile(context.Background(), req)
			require.NoError(t, err)

			var updated syncv1.WaitGroup
			err = client.Get(context.Background(), req.NamespacedName, &updated)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedPhase, updated.Status.Phase)
		})
	}
}
