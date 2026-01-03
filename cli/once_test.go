package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

func TestOnceListCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	onces := []runtime.Object{
		&syncv1.Once{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "once1",
				Namespace: "default",
			},
			Spec: syncv1.OnceSpec{},
			Status: syncv1.OnceStatus{
				Executed: false,
				Phase:    syncv1.OncePhasePending,
			},
		},
		&syncv1.Once{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "once2",
				Namespace: "default",
			},
			Spec: syncv1.OnceSpec{},
			Status: syncv1.OnceStatus{
				Executed: true,
				Executor: "pod-1",
				Phase:    syncv1.OncePhaseExecuted,
			},
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(onces...).
		Build()
	namespace = "default"

	cmd := newOnceListCmd()

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestOnceCreateCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	namespace = "default"

	cmd := newOnceCreateCmd()
	cmd.SetArgs([]string{"test-once"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestOnceCreateCmd_WithTTL(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	namespace = "default"

	cmd := newOnceCreateCmd()
	cmd.SetArgs([]string{"test-once", "--ttl", "5m"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestOnceDeleteCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	once := &syncv1.Once{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-once",
			Namespace: "default",
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(once).
		Build()
	namespace = "default"

	cmd := newOnceDeleteCmd()
	cmd.SetArgs([]string{"test-once"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestOnceCheckCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	once := &syncv1.Once{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-once",
			Namespace: "default",
		},
		Status: syncv1.OnceStatus{
			Executed: true,
			Executor: "pod-1",
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(once).
		Build()
	namespace = "default"

	cmd := newOnceCheckCmd()
	cmd.SetArgs([]string{"test-once"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestOnceListCmd_Empty(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	namespace = "default"

	cmd := newOnceListCmd()

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}
