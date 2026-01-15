package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

func setupGateTestScheme(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))
	return scheme
}

func TestGateWaitCmd_Open(t *testing.T) {
	scheme := setupGateTestScheme(t)

	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "default",
		},
		Spec: syncv1.GateSpec{
			Conditions: []syncv1.GateCondition{},
		},
		Status: syncv1.GateStatus{
			Phase:    syncv1.GatePhaseOpen,
			OpenedAt: &metav1.Time{},
		},
	}

	oldClient := k8sClient
	oldNamespace := namespace
	defer func() {
		k8sClient = oldClient
		namespace = oldNamespace
	}()

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gate).
		Build()
	namespace = "default"

	cmd := newGateWaitCmd()
	cmd.SetArgs([]string{"test-gate"})

	_, err := executeCommandWithOutput(t, cmd)
	require.NoError(t, err)
}

func TestGateWaitCmd_Failed(t *testing.T) {
	scheme := setupGateTestScheme(t)

	gate := &syncv1.Gate{
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
		Status: syncv1.GateStatus{
			Phase: syncv1.GatePhaseFailed,
			ConditionStatuses: []syncv1.GateConditionStatus{
				{
					Type:    "Job",
					Name:    "test-job",
					Met:     false,
					Message: "Job not found",
				},
			},
		},
	}

	oldClient := k8sClient
	oldNamespace := namespace
	defer func() {
		k8sClient = oldClient
		namespace = oldNamespace
	}()

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gate).
		Build()
	namespace = "default"

	cmd := newGateWaitCmd()
	cmd.SetArgs([]string{"test-gate"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gate test-gate failed")
}

func TestGateListCmd(t *testing.T) {
	scheme := setupGateTestScheme(t)

	gates := []runtime.Object{
		&syncv1.Gate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gate1",
				Namespace: "default",
			},
			Spec: syncv1.GateSpec{
				Conditions: []syncv1.GateCondition{
					{Type: "Job", Name: "job1", State: "Complete"},
					{Type: "Semaphore", Name: "sem1", Value: &[]int32{3}[0]},
				},
			},
			Status: syncv1.GateStatus{
				Phase: syncv1.GatePhaseWaiting,
				ConditionStatuses: []syncv1.GateConditionStatus{
					{Type: "Job", Name: "job1", Met: true, Message: "Complete"},
					{Type: "Semaphore", Name: "sem1", Met: false, Message: "Not enough permits"},
				},
			},
		},
		&syncv1.Gate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gate2",
				Namespace: "default",
			},
			Spec: syncv1.GateSpec{
				Conditions: []syncv1.GateCondition{
					{Type: "Barrier", Name: "barrier1", State: "Open"},
				},
			},
			Status: syncv1.GateStatus{
				Phase:    syncv1.GatePhaseOpen,
				OpenedAt: &metav1.Time{},
				ConditionStatuses: []syncv1.GateConditionStatus{
					{Type: "Barrier", Name: "barrier1", Met: true, Message: "Barrier is open"},
				},
			},
		},
	}

	oldClient := k8sClient
	oldNamespace := namespace
	defer func() {
		k8sClient = oldClient
		namespace = oldNamespace
	}()

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gates...).
		Build()
	namespace = "default"

	cmd := newGateListCmd()

	_, err := executeCommandWithOutput(t, cmd)
	require.NoError(t, err)
}

func TestGateCreateCmd(t *testing.T) {
	scheme := setupGateTestScheme(t)

	oldClient := k8sClient
	oldNamespace := namespace
	defer func() {
		k8sClient = oldClient
		namespace = oldNamespace
	}()

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	namespace = "default"

	cmd := newGateCreateCmd()
	cmd.SetArgs([]string{"test-gate"})

	_, err := executeCommandWithOutput(t, cmd)
	require.NoError(t, err)
}

func TestGateDeleteCmd(t *testing.T) {
	scheme := setupGateTestScheme(t)

	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "default",
		},
	}

	oldClient := k8sClient
	oldNamespace := namespace
	defer func() {
		k8sClient = oldClient
		namespace = oldNamespace
	}()

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gate).
		Build()
	namespace = "default"

	cmd := newGateDeleteCmd()
	cmd.SetArgs([]string{"test-gate"})

	_, err := executeCommandWithOutput(t, cmd)
	require.NoError(t, err)
}

func TestGateOpenCmd(t *testing.T) {
	scheme := setupGateTestScheme(t)

	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "default",
		},
		Status: syncv1.GateStatus{
			Phase: syncv1.GatePhaseWaiting,
		},
	}

	oldClient := k8sClient
	oldNamespace := namespace
	defer func() {
		k8sClient = oldClient
		namespace = oldNamespace
	}()

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gate).
		WithStatusSubresource(&syncv1.Gate{}).
		Build()
	namespace = "default"

	cmd := newGateOpenCmd()
	cmd.SetArgs([]string{"test-gate"})

	_, err := executeCommandWithOutput(t, cmd)
	require.NoError(t, err)
}

func TestGateCloseCmd(t *testing.T) {
	scheme := setupGateTestScheme(t)

	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "default",
		},
		Status: syncv1.GateStatus{
			Phase: syncv1.GatePhaseOpen,
		},
	}

	oldClient := k8sClient
	oldNamespace := namespace
	defer func() {
		k8sClient = oldClient
		namespace = oldNamespace
	}()

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gate).
		WithStatusSubresource(&syncv1.Gate{}).
		Build()
	namespace = "default"

	cmd := newGateCloseCmd()
	cmd.SetArgs([]string{"test-gate"})

	_, err := executeCommandWithOutput(t, cmd)
	require.NoError(t, err)
}
