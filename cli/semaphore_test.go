package main

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

func TestSemaphoreAcquireCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	semaphore := &syncv1.Semaphore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sem",
			Namespace: "default",
		},
		Spec: syncv1.SemaphoreSpec{
			Permits: 5,
		},
		Status: syncv1.SemaphoreStatus{
			Available: 3,
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(semaphore).
		Build()
	namespace = "default"

	cmd := newSemaphoreAcquireCmd()
	cmd.SetArgs([]string{"test-sem", "--holder", "test-holder"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Acquired permit for semaphore 'test-sem'")
	assert.Contains(t, output, "holder: test-holder")
}

func TestSemaphoreAcquireCmd_NoPermits(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	semaphore := &syncv1.Semaphore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sem",
			Namespace: "default",
		},
		Spec: syncv1.SemaphoreSpec{
			Permits: 5,
		},
		Status: syncv1.SemaphoreStatus{
			Available: 0,
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(semaphore).
		Build()
	namespace = "default"

	cmd := newSemaphoreAcquireCmd()
	cmd.SetArgs([]string{"test-sem", "--holder", "test-holder"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no permits available")
}

func TestSemaphoreReleaseCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	permit := &syncv1.Permit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sem-test-holder",
			Namespace: "default",
		},
		Spec: syncv1.PermitSpec{
			Semaphore: "test-sem",
			Holder:    "test-holder",
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(permit).
		Build()
	namespace = "default"

	cmd := newSemaphoreReleaseCmd()
	cmd.SetArgs([]string{"test-sem", "--holder", "test-holder"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Released permit for semaphore 'test-sem'")
}

func TestSemaphoreListCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	semaphores := []runtime.Object{
		&syncv1.Semaphore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sem1",
				Namespace: "default",
			},
			Spec: syncv1.SemaphoreSpec{
				Permits: 5,
			},
			Status: syncv1.SemaphoreStatus{
				InUse:     2,
				Available: 3,
				Phase:     syncv1.SemaphorePhaseReady,
			},
		},
		&syncv1.Semaphore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sem2",
				Namespace: "default",
			},
			Spec: syncv1.SemaphoreSpec{
				Permits: 3,
			},
			Status: syncv1.SemaphoreStatus{
				InUse:     3,
				Available: 0,
				Phase:     syncv1.SemaphorePhaseFull,
			},
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(semaphores...).
		Build()
	namespace = "default"

	cmd := newSemaphoreListCmd()

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "sem1")
	assert.Contains(t, output, "sem2")
	assert.Contains(t, output, "Ready")
	assert.Contains(t, output, "Full")
}

func TestSemaphoreCmd_DefaultHolder(t *testing.T) {
	originalHostname := os.Getenv("HOSTNAME")
	defer os.Setenv("HOSTNAME", originalHostname)

	os.Setenv("HOSTNAME", "test-pod")

	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	semaphore := &syncv1.Semaphore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sem",
			Namespace: "default",
		},
		Spec: syncv1.SemaphoreSpec{
			Permits: 5,
		},
		Status: syncv1.SemaphoreStatus{
			Available: 3,
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(semaphore).
		Build()
	namespace = "default"

	cmd := newSemaphoreAcquireCmd()
	cmd.SetArgs([]string{"test-sem"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "holder: test-pod")
}