package barrier

import (
	"context"
	"testing"

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
		Build()

	return konductor.NewFromClient(k8sClient, "test-ns")
}

func TestList(t *testing.T) {
	barrier1 := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "barrier1",
			Namespace: "test-ns",
		},
		Spec: syncv1.BarrierSpec{
			Expected: 3,
		},
	}

	client := setupTestClient(t, barrier1)

	barriers, err := List(client, context.Background())
	require.NoError(t, err)
	assert.Len(t, barriers, 1)
	assert.Equal(t, "barrier1", barriers[0].Name)
}

func TestGet(t *testing.T) {
	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-barrier",
			Namespace: "test-ns",
		},
		Spec: syncv1.BarrierSpec{
			Expected: 3,
		},
	}

	client := setupTestClient(t, barrier)

	result, err := Get(client, context.Background(), "test-barrier")
	require.NoError(t, err)
	assert.Equal(t, "test-barrier", result.Name)
	assert.Equal(t, int32(3), result.Spec.Expected)
}

func TestCreate(t *testing.T) {
	client := setupTestClient(t)

	err := Create(client, context.Background(), "test-barrier", 3)
	assert.NoError(t, err)
}

func TestDelete(t *testing.T) {
	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-barrier",
			Namespace: "test-ns",
		},
	}
	client := setupTestClient(t, barrier)

	err := Delete(client, context.Background(), "test-barrier")
	assert.NoError(t, err)
}

func TestGetStatus(t *testing.T) {
	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-barrier",
			Namespace: "test-ns",
		},
		Spec: syncv1.BarrierSpec{
			Expected: 3,
		},
		Status: syncv1.BarrierStatus{
			Arrived:  2,
			Phase:    syncv1.BarrierPhaseWaiting,
			Arrivals: []string{"holder1", "holder2"},
		},
	}

	client := setupTestClient(t, barrier)

	status, err := GetStatus(client, context.Background(), "test-barrier")
	require.NoError(t, err)
	assert.Equal(t, int32(2), status.Arrived)
	assert.Equal(t, syncv1.BarrierPhaseWaiting, status.Phase)
	assert.Len(t, status.Arrivals, 2)
}

func TestArriveBarrier(t *testing.T) {
	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-barrier",
			Namespace: "test-ns",
		},
		Spec: syncv1.BarrierSpec{
			Expected: 3,
		},
		Status: syncv1.BarrierStatus{
			Arrived: 1,
			Phase:   syncv1.BarrierPhaseWaiting,
		},
	}

	client := setupTestClient(t, barrier)

	err := Arrive(client, context.Background(), "test-barrier", konductor.WithHolder("test-holder"))
	require.NoError(t, err)

	// Verify arrival was created
	var arrivals syncv1.ArrivalList
	err = client.K8sClient().List(context.Background(), &arrivals)
	require.NoError(t, err)
	assert.Len(t, arrivals.Items, 1)
	assert.Equal(t, "test-barrier", arrivals.Items[0].Spec.Barrier)
	assert.Equal(t, "test-holder", arrivals.Items[0].Spec.Holder)
}

func TestWaitBarrier_AlreadyOpen(t *testing.T) {
	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-barrier",
			Namespace: "test-ns",
		},
		Spec: syncv1.BarrierSpec{
			Expected: 2,
		},
		Status: syncv1.BarrierStatus{
			Arrived:  2,
			Phase:    syncv1.BarrierPhaseOpen,
			OpenedAt: &metav1.Time{},
		},
	}

	client := setupTestClient(t, barrier)

	err := Wait(client, context.Background(), "test-barrier")
	assert.NoError(t, err)
}

func TestWaitBarrier_Failed(t *testing.T) {
	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-barrier",
			Namespace: "test-ns",
		},
		Spec: syncv1.BarrierSpec{
			Expected: 3,
		},
		Status: syncv1.BarrierStatus{
			Arrived: 1,
			Phase:   syncv1.BarrierPhaseFailed,
		},
	}

	client := setupTestClient(t, barrier)

	err := Wait(client, context.Background(), "test-barrier")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "barrier test-barrier failed")
}

func TestUpdate(t *testing.T) {
	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-barrier",
			Namespace: "test-ns",
		},
		Spec: syncv1.BarrierSpec{
			Expected: 3,
		},
	}
	client := setupTestClient(t, barrier)

	// Update expected count
	barrier.Spec.Expected = 5
	err := Update(client, context.Background(), barrier)
	assert.NoError(t, err)
}
