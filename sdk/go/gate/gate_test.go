package gate

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
	konductor "github.com/LogicIQ/konductor/sdk/go/client"
)

func TestWaitGate_Open(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "default",
		},
		Status: syncv1.GateStatus{
			Phase: syncv1.GatePhaseOpen,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gate).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	err := client.WaitGate(context.Background(), "test-gate")
	require.NoError(t, err)
}

func TestWaitGate_Failed(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "default",
		},
		Status: syncv1.GateStatus{
			Phase: syncv1.GatePhaseFailed,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gate).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	err := client.WaitGate(context.Background(), "test-gate")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gate test-gate failed")
}

func TestWaitGate_Timeout(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "default",
		},
		Status: syncv1.GateStatus{
			Phase: syncv1.GatePhaseWaiting,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gate).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	err := client.WaitGate(context.Background(), "test-gate", 
		konductor.WithTimeout(100*time.Millisecond))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestCheckGate(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	tests := []struct {
		name     string
		phase    syncv1.GatePhase
		expected bool
	}{
		{
			name:     "open gate",
			phase:    syncv1.GatePhaseOpen,
			expected: true,
		},
		{
			name:     "waiting gate",
			phase:    syncv1.GatePhaseWaiting,
			expected: false,
		},
		{
			name:     "failed gate",
			phase:    syncv1.GatePhaseFailed,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gate := &syncv1.Gate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-gate",
					Namespace: "default",
				},
				Status: syncv1.GateStatus{
					Phase: tt.phase,
				},
			}

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(gate).
				Build()

			client := konductor.NewFromClient(k8sClient, "default")

			isOpen, err := client.CheckGate(context.Background(), "test-gate")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, isOpen)
		})
	}
}

func TestGetGateConditions(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "default",
		},
		Status: syncv1.GateStatus{
			ConditionStatuses: []syncv1.GateConditionStatus{
				{
					Type:    "Job",
					Name:    "job1",
					Met:     true,
					Message: "Job completed",
				},
				{
					Type:    "Semaphore",
					Name:    "sem1",
					Met:     false,
					Message: "Not enough permits",
				},
			},
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gate).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	conditions, err := client.GetGateConditions(context.Background(), "test-gate")
	require.NoError(t, err)
	assert.Len(t, conditions, 2)
	assert.Equal(t, "Job", conditions[0].Type)
	assert.True(t, conditions[0].Met)
	assert.Equal(t, "Semaphore", conditions[1].Type)
	assert.False(t, conditions[1].Met)
}

func TestWaitForConditions_AllMet(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "default",
		},
		Status: syncv1.GateStatus{
			ConditionStatuses: []syncv1.GateConditionStatus{
				{
					Type:    "Job",
					Name:    "job1",
					Met:     true,
					Message: "Job completed",
				},
				{
					Type:    "Semaphore",
					Name:    "sem1",
					Met:     true,
					Message: "Permits available",
				},
			},
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gate).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	err := client.WaitForConditions(context.Background(), "test-gate", []string{"job1", "sem1"})
	require.NoError(t, err)
}

func TestWaitForConditions_Timeout(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "default",
		},
		Status: syncv1.GateStatus{
			ConditionStatuses: []syncv1.GateConditionStatus{
				{
					Type:    "Job",
					Name:    "job1",
					Met:     false,
					Message: "Job not completed",
				},
			},
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gate).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	err := client.WaitForConditions(context.Background(), "test-gate", []string{"job1"}, 
		konductor.WithTimeout(100*time.Millisecond))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestWithGate(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "default",
		},
		Status: syncv1.GateStatus{
			Phase: syncv1.GatePhaseOpen,
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gate).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	executed := false
	err := client.WithGate(context.Background(), "test-gate", func() error {
		executed = true
		return nil
	})

	require.NoError(t, err)
	assert.True(t, executed)
}

func TestListGates(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	gates := []runtime.Object{
		&syncv1.Gate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gate1",
				Namespace: "default",
			},
		},
		&syncv1.Gate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gate2",
				Namespace: "default",
			},
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gates...).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	result, err := client.ListGates(context.Background())
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestGetGate(t *testing.T) {
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
					Type: "Job",
					Name: "test-job",
				},
			},
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gate).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	result, err := client.GetGate(context.Background(), "test-gate")
	require.NoError(t, err)
	assert.Equal(t, "test-gate", result.Name)
	assert.Len(t, result.Spec.Conditions, 1)
}

func TestGetGateStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gate",
			Namespace: "default",
		},
		Status: syncv1.GateStatus{
			Phase: syncv1.GatePhaseWaiting,
			ConditionStatuses: []syncv1.GateConditionStatus{
				{
					Type: "Job",
					Name: "test-job",
					Met:  false,
				},
			},
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gate).
		Build()

	client := konductor.NewFromClient(k8sClient, "default")

	status, err := client.GetGateStatus(context.Background(), "test-gate")
	require.NoError(t, err)
	assert.Equal(t, syncv1.GatePhaseWaiting, status.Phase)
	assert.Len(t, status.ConditionStatuses, 1)
}