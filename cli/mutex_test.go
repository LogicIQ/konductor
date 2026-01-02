package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

func TestMutexListCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	mutexes := []runtime.Object{
		&syncv1.Mutex{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mutex1",
				Namespace: "default",
			},
			Spec: syncv1.MutexSpec{},
			Status: syncv1.MutexStatus{
				Phase:  syncv1.MutexPhaseUnlocked,
				Holder: "",
			},
		},
		&syncv1.Mutex{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mutex2",
				Namespace: "default",
			},
			Spec: syncv1.MutexSpec{},
			Status: syncv1.MutexStatus{
				Phase:    syncv1.MutexPhaseLocked,
				Holder:   "holder1",
				LockedAt: &metav1.Time{},
			},
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(mutexes...).
		Build()
	namespace = "default"

	cmd := newMutexListCmd()

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestMutexCreateCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	namespace = "default"

	cmd := newMutexCreateCmd()
	cmd.SetArgs([]string{"test-mutex"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestMutexCreateCmd_WithTTL(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	namespace = "default"

	cmd := newMutexCreateCmd()
	cmd.SetArgs([]string{"test-mutex", "--ttl", "5m"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestMutexDeleteCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mutex",
			Namespace: "default",
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(mutex).
		Build()
	namespace = "default"

	cmd := newMutexDeleteCmd()
	cmd.SetArgs([]string{"test-mutex"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestMutexLockCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mutex",
			Namespace: "default",
		},
		Status: syncv1.MutexStatus{
			Phase: syncv1.MutexPhaseUnlocked,
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(mutex).
		WithStatusSubresource(&syncv1.Mutex{}).
		Build()
	namespace = "default"

	cmd := newMutexLockCmd()
	cmd.SetArgs([]string{"test-mutex", "--holder", "test-holder"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestMutexUnlockCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mutex",
			Namespace: "default",
		},
		Status: syncv1.MutexStatus{
			Phase:  syncv1.MutexPhaseLocked,
			Holder: "test-holder",
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(mutex).
		WithStatusSubresource(&syncv1.Mutex{}).
		Build()
	namespace = "default"

	cmd := newMutexUnlockCmd()
	cmd.SetArgs([]string{"test-mutex", "--holder", "test-holder"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestMutexLockCmd_WithTimeout(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mutex",
			Namespace: "default",
		},
		Status: syncv1.MutexStatus{
			Phase: syncv1.MutexPhaseUnlocked,
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(mutex).
		WithStatusSubresource(&syncv1.Mutex{}).
		Build()
	namespace = "default"

	cmd := newMutexLockCmd()
	cmd.SetArgs([]string{"test-mutex", "--holder", "test-holder", "--timeout", "5s"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestMutexListCmd_Empty(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	namespace = "default"

	cmd := newMutexListCmd()

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestMutexUnlockCmd_NotHolder(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mutex",
			Namespace: "default",
		},
		Status: syncv1.MutexStatus{
			Phase:  syncv1.MutexPhaseLocked,
			Holder: "other-holder",
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(mutex).
		WithStatusSubresource(&syncv1.Mutex{}).
		Build()
	namespace = "default"

	cmd := newMutexUnlockCmd()
	cmd.SetArgs([]string{"test-mutex", "--holder", "test-holder"})

	err := cmd.Execute()
	require.Error(t, err)
}

func TestMutexLockCmd_AlreadyLocked(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mutex",
			Namespace: "default",
		},
		Status: syncv1.MutexStatus{
			Phase:    syncv1.MutexPhaseLocked,
			Holder:   "other-holder",
			LockedAt: &metav1.Time{Time: time.Now()},
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(mutex).
		WithStatusSubresource(&syncv1.Mutex{}).
		Build()
	namespace = "default"

	cmd := newMutexLockCmd()
	cmd.SetArgs([]string{"test-mutex", "--holder", "test-holder", "--timeout", "100ms"})

	err := cmd.Execute()
	require.Error(t, err)
}
