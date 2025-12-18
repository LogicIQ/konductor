package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

func TestGateWaitCmd_Open(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

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

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gate).
		Build()
	namespace = "default"

	cmd := newGateWaitCmd()
	cmd.SetArgs([]string{"test-gate"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Gate 'test-gate' is open!")
}

func TestGateWaitCmd_Failed(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

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

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gate).
		Build()
	namespace = "default"

	cmd := newGateWaitCmd()
	cmd.SetArgs([]string{"test-gate"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gate 'test-gate' failed")
}

func TestGateListCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

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

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gates...).
		Build()
	namespace = "default"

	cmd := newGateListCmd()

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "gate1")
	assert.Contains(t, output, "gate2")
	assert.Contains(t, output, "1/2")
	assert.Contains(t, output, "1/1")
	assert.Contains(t, output, "Waiting")
	assert.Contains(t, output, "Open")
}