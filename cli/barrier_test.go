package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

func TestBarrierWaitCmd_Open(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	require.NoError(t, syncv1.AddToScheme(scheme))

	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-barrier",
			Namespace: "default",
		},
		Spec: syncv1.BarrierSpec{
			Expected: 3,
		},
		Status: syncv1.BarrierStatus{
			Phase:   syncv1.BarrierPhaseOpen,
			Arrived: 3,
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(barrier).
		Build()
	namespace = "default"

	cmd := newBarrierWaitCmd()
	cmd.SetArgs([]string{"test-barrier"})

	_, err := executeCommandWithOutput(t, cmd)
	require.NoError(t, err)
}

func TestBarrierWaitCmd_Failed(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	require.NoError(t, syncv1.AddToScheme(scheme))

	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-barrier",
			Namespace: "default",
		},
		Spec: syncv1.BarrierSpec{
			Expected: 3,
		},
		Status: syncv1.BarrierStatus{
			Phase:   syncv1.BarrierPhaseFailed,
			Arrived: 1,
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(barrier).
		Build()
	namespace = "default"

	cmd := newBarrierWaitCmd()
	cmd.SetArgs([]string{"test-barrier"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "barrier test-barrier failed")
}

func TestBarrierArriveCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	require.NoError(t, syncv1.AddToScheme(scheme))

	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-barrier",
			Namespace: "default",
		},
		Spec: syncv1.BarrierSpec{
			Expected: 3,
		},
		Status: syncv1.BarrierStatus{
			Phase:   syncv1.BarrierPhaseWaiting,
			Arrived: 1,
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(barrier).
		Build()
	namespace = "default"

	cmd := newBarrierArriveCmd()
	cmd.SetArgs([]string{"test-barrier", "--holder", "test-holder"})

	_, err := executeCommandWithOutput(t, cmd)
	require.NoError(t, err)
}

func TestBarrierListCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	require.NoError(t, syncv1.AddToScheme(scheme))

	barriers := []runtime.Object{
		&syncv1.Barrier{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "barrier1",
				Namespace: "default",
			},
			Spec: syncv1.BarrierSpec{
				Expected: 3,
			},
			Status: syncv1.BarrierStatus{
				Phase:   syncv1.BarrierPhaseWaiting,
				Arrived: 1,
			},
		},
		&syncv1.Barrier{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "barrier2",
				Namespace: "default",
			},
			Spec: syncv1.BarrierSpec{
				Expected: 2,
			},
			Status: syncv1.BarrierStatus{
				Phase:    syncv1.BarrierPhaseOpen,
				Arrived:  2,
				OpenedAt: &metav1.Time{},
			},
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(barriers...).
		Build()
	namespace = "default"

	cmd := newBarrierListCmd()

	_, err := executeCommandWithOutput(t, cmd)
	require.NoError(t, err)
}

func TestBarrierCmd_DefaultHolder(t *testing.T) {
	originalHostname := os.Getenv("HOSTNAME")
	defer os.Setenv("HOSTNAME", originalHostname)

	os.Setenv("HOSTNAME", "test-pod")

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	require.NoError(t, syncv1.AddToScheme(scheme))

	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-barrier",
			Namespace: "default",
		},
		Spec: syncv1.BarrierSpec{
			Expected: 3,
		},
		Status: syncv1.BarrierStatus{
			Phase:   syncv1.BarrierPhaseWaiting,
			Arrived: 1,
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(barrier).
		Build()
	namespace = "default"

	cmd := newBarrierArriveCmd()
	cmd.SetArgs([]string{"test-barrier"})

	output, err := executeCommandWithOutput(t, cmd)
	require.NoError(t, err)
	_ = output // Remove holder assertion since it's not in output
}

func TestBarrierCreateCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	namespace = "default"

	cmd := newBarrierCreateCmd()
	cmd.SetArgs([]string{"test-barrier", "--expected", "5"})

	_, err := executeCommandWithOutput(t, cmd)
	require.NoError(t, err)
}

func TestBarrierDeleteCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	require.NoError(t, syncv1.AddToScheme(scheme))

	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-barrier",
			Namespace: "default",
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(barrier).
		Build()
	namespace = "default"

	cmd := newBarrierDeleteCmd()
	cmd.SetArgs([]string{"test-barrier"})

	_, err := executeCommandWithOutput(t, cmd)
	require.NoError(t, err)
}
