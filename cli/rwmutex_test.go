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

func TestRWMutexListCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	rwmutexes := []runtime.Object{
		&syncv1.RWMutex{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rwmutex1",
				Namespace: "default",
			},
			Spec: syncv1.RWMutexSpec{},
			Status: syncv1.RWMutexStatus{
				Phase: syncv1.RWMutexPhaseUnlocked,
			},
		},
		&syncv1.RWMutex{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rwmutex2",
				Namespace: "default",
			},
			Spec: syncv1.RWMutexSpec{},
			Status: syncv1.RWMutexStatus{
				Phase:       syncv1.RWMutexPhaseReadLocked,
				ReadHolders: []string{"reader-1", "reader-2"},
				LockedAt:    &metav1.Time{},
			},
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(rwmutexes...).
		Build()
	namespace = "default"

	cmd := newRWMutexListCmd()

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestRWMutexCreateCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	namespace = "default"

	cmd := newRWMutexCreateCmd()
	cmd.SetArgs([]string{"test-rwmutex"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestRWMutexCreateCmd_WithTTL(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	namespace = "default"

	cmd := newRWMutexCreateCmd()
	cmd.SetArgs([]string{"test-rwmutex", "--ttl", "5m"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestRWMutexDeleteCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "default",
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(rwmutex).
		Build()
	namespace = "default"

	cmd := newRWMutexDeleteCmd()
	cmd.SetArgs([]string{"test-rwmutex"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestRWMutexRLockCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "default",
		},
		Status: syncv1.RWMutexStatus{
			Phase: syncv1.RWMutexPhaseUnlocked,
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(rwmutex).
		WithStatusSubresource(&syncv1.RWMutex{}).
		Build()
	namespace = "default"

	cmd := newRWMutexRLockCmd()
	cmd.SetArgs([]string{"test-rwmutex", "--holder", "test-reader"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestRWMutexLockCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "default",
		},
		Status: syncv1.RWMutexStatus{
			Phase: syncv1.RWMutexPhaseUnlocked,
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(rwmutex).
		WithStatusSubresource(&syncv1.RWMutex{}).
		Build()
	namespace = "default"

	cmd := newRWMutexLockCmd()
	cmd.SetArgs([]string{"test-rwmutex", "--holder", "test-writer"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestRWMutexUnlockCmd_ReadLock(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "default",
		},
		Status: syncv1.RWMutexStatus{
			Phase:       syncv1.RWMutexPhaseReadLocked,
			ReadHolders: []string{"test-reader"},
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(rwmutex).
		WithStatusSubresource(&syncv1.RWMutex{}).
		Build()
	namespace = "default"

	cmd := newRWMutexUnlockCmd()
	cmd.SetArgs([]string{"test-rwmutex", "--holder", "test-reader"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestRWMutexUnlockCmd_WriteLock(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "default",
		},
		Status: syncv1.RWMutexStatus{
			Phase:       syncv1.RWMutexPhaseWriteLocked,
			WriteHolder: "test-writer",
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(rwmutex).
		WithStatusSubresource(&syncv1.RWMutex{}).
		Build()
	namespace = "default"

	cmd := newRWMutexUnlockCmd()
	cmd.SetArgs([]string{"test-rwmutex", "--holder", "test-writer"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestRWMutexLockCmd_WithTimeout(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "default",
		},
		Status: syncv1.RWMutexStatus{
			Phase: syncv1.RWMutexPhaseUnlocked,
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(rwmutex).
		WithStatusSubresource(&syncv1.RWMutex{}).
		Build()
	namespace = "default"

	cmd := newRWMutexLockCmd()
	cmd.SetArgs([]string{"test-rwmutex", "--holder", "test-writer", "--timeout", "5s"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestRWMutexListCmd_Empty(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	namespace = "default"

	cmd := newRWMutexListCmd()

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestRWMutexUnlockCmd_NotHolder(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "default",
		},
		Status: syncv1.RWMutexStatus{
			Phase:       syncv1.RWMutexPhaseWriteLocked,
			WriteHolder: "other-writer",
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(rwmutex).
		WithStatusSubresource(&syncv1.RWMutex{}).
		Build()
	namespace = "default"

	cmd := newRWMutexUnlockCmd()
	cmd.SetArgs([]string{"test-rwmutex", "--holder", "test-writer"})

	err := cmd.Execute()
	require.Error(t, err)
}

func TestRWMutexLockCmd_AlreadyWriteLocked(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rwmutex",
			Namespace: "default",
		},
		Status: syncv1.RWMutexStatus{
			Phase:       syncv1.RWMutexPhaseWriteLocked,
			WriteHolder: "other-writer",
			LockedAt:    &metav1.Time{},
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(rwmutex).
		WithStatusSubresource(&syncv1.RWMutex{}).
		Build()
	namespace = "default"

	cmd := newRWMutexLockCmd()
	cmd.SetArgs([]string{"test-rwmutex", "--holder", "test-writer", "--timeout", "100ms"})

	err := cmd.Execute()
	require.Error(t, err)
}
