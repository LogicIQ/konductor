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

func TestWaitGroupListCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	wgs := []runtime.Object{
		&syncv1.WaitGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wg1",
				Namespace: "default",
			},
			Status: syncv1.WaitGroupStatus{
				Counter: 3,
				Phase:   syncv1.WaitGroupPhaseWaiting,
			},
		},
		&syncv1.WaitGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wg2",
				Namespace: "default",
			},
			Status: syncv1.WaitGroupStatus{
				Counter: 0,
				Phase:   syncv1.WaitGroupPhaseDone,
			},
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(wgs...).
		Build()
	namespace = "default"

	cmd := newWaitGroupListCmd()

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestWaitGroupCreateCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	namespace = "default"

	cmd := newWaitGroupCreateCmd()
	cmd.SetArgs([]string{"test-wg"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestWaitGroupDeleteCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	wg := &syncv1.WaitGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wg",
			Namespace: "default",
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(wg).
		Build()
	namespace = "default"

	cmd := newWaitGroupDeleteCmd()
	cmd.SetArgs([]string{"test-wg"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestWaitGroupAddCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	wg := &syncv1.WaitGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wg",
			Namespace: "default",
		},
		Status: syncv1.WaitGroupStatus{
			Counter: 0,
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(wg).
		WithStatusSubresource(&syncv1.WaitGroup{}).
		Build()
	namespace = "default"

	cmd := newWaitGroupAddCmd()
	cmd.SetArgs([]string{"test-wg", "--delta", "3"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestWaitGroupDoneCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	wg := &syncv1.WaitGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wg",
			Namespace: "default",
		},
		Status: syncv1.WaitGroupStatus{
			Counter: 1,
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(wg).
		WithStatusSubresource(&syncv1.WaitGroup{}).
		Build()
	namespace = "default"

	cmd := newWaitGroupDoneCmd()
	cmd.SetArgs([]string{"test-wg"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}
