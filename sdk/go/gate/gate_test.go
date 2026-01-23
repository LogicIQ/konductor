package gate

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

	builder := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...)
	if len(objects) > 0 {
		builder = builder.WithStatusSubresource(&syncv1.Gate{})
	}
	k8sClient := builder.Build()

	return konductor.NewFromClient(k8sClient, "test-ns")
}

func TestList(t *testing.T) {
	gate1 := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gate1",
			Namespace: "test-ns",
		},
		Spec: syncv1.GateSpec{
			Conditions: []syncv1.GateCondition{},
		},
	}

	client := setupTestClient(t, gate1)

	gates, err := List(client, context.Background())
	require.NoError(t, err)
	assert.Len(t, gates, 1)
	assert.Equal(t, "gate1", gates[0].Name)
}

func TestGet(t *testing.T) {
	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "test-ns",
		},
		Spec: syncv1.GateSpec{
			Conditions: []syncv1.GateCondition{},
		},
	}

	client := setupTestClient(t, gate)

	result, err := Get(client, context.Background(), "test-gate")
	require.NoError(t, err)
	assert.Equal(t, "test-gate", result.Name)
}

func TestCreate(t *testing.T) {
	client := setupTestClient(t)

	err := Create(client, context.Background(), "test-gate")
	assert.NoError(t, err)
}

func TestDelete(t *testing.T) {
	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "test-ns",
		},
	}
	client := setupTestClient(t, gate)

	err := Delete(client, context.Background(), "test-gate")
	assert.NoError(t, err)
}

func TestGetStatus(t *testing.T) {
	now := metav1.Now()
	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "test-ns",
		},
		Status: syncv1.GateStatus{
			Phase:    syncv1.GatePhaseOpen,
			OpenedAt: &now,
			ConditionStatuses: []syncv1.GateConditionStatus{
				{Type: "Job", Name: "job1", Met: true, Message: "Complete"},
			},
		},
	}

	client := setupTestClient(t, gate)

	status, err := GetStatus(client, context.Background(), "test-gate")
	require.NoError(t, err)
	assert.Equal(t, syncv1.GatePhaseOpen, status.Phase)
	assert.NotNil(t, status.OpenedAt)
	assert.Len(t, status.ConditionStatuses, 1)
}

func TestCheck_Open(t *testing.T) {
	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "test-ns",
		},
		Status: syncv1.GateStatus{
			Phase: syncv1.GatePhaseOpen,
		},
	}

	client := setupTestClient(t, gate)

	isOpen, err := Check(client, context.Background(), "test-gate")
	require.NoError(t, err)
	assert.True(t, isOpen)
}

func TestCheck_Waiting(t *testing.T) {
	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "test-ns",
		},
		Status: syncv1.GateStatus{
			Phase: syncv1.GatePhaseWaiting,
		},
	}

	client := setupTestClient(t, gate)

	isOpen, err := Check(client, context.Background(), "test-gate")
	require.NoError(t, err)
	assert.False(t, isOpen)
}

func TestGetConditions(t *testing.T) {
	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "test-ns",
		},
		Status: syncv1.GateStatus{
			ConditionStatuses: []syncv1.GateConditionStatus{
				{Type: "Job", Name: "job1", Met: true, Message: "Complete"},
				{Type: "Semaphore", Name: "sem1", Met: false, Message: "Not enough permits"},
			},
		},
	}

	client := setupTestClient(t, gate)

	conditions, err := GetConditions(client, context.Background(), "test-gate")
	require.NoError(t, err)
	assert.Len(t, conditions, 2)
	assert.Equal(t, "Job", conditions[0].Type)
	assert.True(t, conditions[0].Met)
	assert.Equal(t, "Semaphore", conditions[1].Type)
	assert.False(t, conditions[1].Met)
}

func TestWait_AlreadyOpen(t *testing.T) {
	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "test-ns",
		},
		Status: syncv1.GateStatus{
			Phase: syncv1.GatePhaseOpen,
		},
	}

	client := setupTestClient(t, gate)

	err := Wait(client, context.Background(), "test-gate")
	assert.NoError(t, err)
}

func TestWait_Failed(t *testing.T) {
	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "test-ns",
		},
		Status: syncv1.GateStatus{
			Phase: syncv1.GatePhaseFailed,
		},
	}

	client := setupTestClient(t, gate)

	err := Wait(client, context.Background(), "test-gate")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gate test-gate failed")
}

func TestUpdate(t *testing.T) {
	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "test-ns",
		},
		Spec: syncv1.GateSpec{
			Conditions: []syncv1.GateCondition{},
		},
	}
	client := setupTestClient(t, gate)

	// Add conditions
	gate.Spec.Conditions = append(gate.Spec.Conditions, syncv1.GateCondition{
		Type: "Job",
		Name: "test-job",
	})
	err := Update(client, context.Background(), gate)
	assert.NoError(t, err)
}
