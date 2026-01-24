package main

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

func TestLeaseAcquireCmd(t *testing.T) {
	t.Skip("Skipping test that requires controller to grant lease")
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	lease := &syncv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease",
			Namespace: "default",
		},
		Spec: syncv1.LeaseSpec{
			TTL: &metav1.Duration{Duration: time.Hour},
		},
		Status: syncv1.LeaseStatus{
			Phase: syncv1.LeasePhaseAvailable,
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(lease).
		WithStatusSubresource(&syncv1.LeaseRequest{}).
		Build()
	namespace = "default"

	cmd := newLeaseAcquireCmd()
	cmd.SetArgs([]string{"test-lease", "--holder", "test-holder"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)

	_ = buf.String()
}

func TestLeaseAcquireCmd_NotAvailable(t *testing.T) {
	t.Skip("Skipping test that requires controller")
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	lease := &syncv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease",
			Namespace: "default",
		},
		Spec: syncv1.LeaseSpec{
			TTL: &metav1.Duration{Duration: time.Hour},
		},
		Status: syncv1.LeaseStatus{
			Phase:  syncv1.LeasePhaseHeld,
			Holder: "other-holder",
		},
	}

	request := &syncv1.LeaseRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease-test-holder",
			Namespace: "default",
		},
		Spec: syncv1.LeaseRequestSpec{
			Lease:  "test-lease",
			Holder: "test-holder",
		},
		Status: syncv1.LeaseRequestStatus{
			Phase: syncv1.LeaseRequestPhasePending,
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(lease, request).
		WithStatusSubresource(&syncv1.LeaseRequest{}).
		Build()
	namespace = "default"

	cmd := newLeaseAcquireCmd()
	cmd.SetArgs([]string{"test-lease", "--holder", "test-holder"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lease 'test-lease' is not available")
}

func TestLeaseReleaseCmd(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	request := &syncv1.LeaseRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease-test-holder",
			Namespace: "default",
		},
		Spec: syncv1.LeaseRequestSpec{
			Lease:  "test-lease",
			Holder: "test-holder",
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(request).
		Build()
	namespace = "default"

	cmd := newLeaseReleaseCmd()
	cmd.SetArgs([]string{"test-lease", "--holder", "test-holder"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)

	_ = buf.String()
}

func TestLeaseListCmd(t *testing.T) {
	logger = initTestLogger(t)
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	leases := []runtime.Object{
		&syncv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lease1",
				Namespace: "default",
			},
			Spec: syncv1.LeaseSpec{
				TTL: &metav1.Duration{Duration: time.Hour},
			},
			Status: syncv1.LeaseStatus{
				Phase:  syncv1.LeasePhaseAvailable,
				Holder: "",
			},
		},
		&syncv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lease2",
				Namespace: "default",
			},
			Spec: syncv1.LeaseSpec{
				TTL: &metav1.Duration{Duration: time.Hour},
			},
			Status: syncv1.LeaseStatus{
				Phase:      syncv1.LeasePhaseHeld,
				Holder:     "holder1",
				AcquiredAt: &metav1.Time{},
				RenewCount: 5,
			},
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(leases...).
		Build()
	namespace = "default"

	cmd := newLeaseListCmd()

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)

	_ = buf.String()
}

func TestLeaseCmd_DefaultHolder(t *testing.T) {
	t.Skip("Skipping test that requires controller to grant lease")
	originalHostname := os.Getenv("HOSTNAME")
	defer os.Setenv("HOSTNAME", originalHostname)

	os.Setenv("HOSTNAME", "test-pod")

	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	lease := &syncv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease",
			Namespace: "default",
		},
		Spec: syncv1.LeaseSpec{
			TTL: &metav1.Duration{Duration: time.Hour},
		},
		Status: syncv1.LeaseStatus{
			Phase: syncv1.LeasePhaseAvailable,
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(lease).
		WithStatusSubresource(&syncv1.LeaseRequest{}).
		Build()
	namespace = "default"

	cmd := newLeaseAcquireCmd()
	cmd.SetArgs([]string{"test-lease"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)

	_ = buf.String()
}

func TestLeaseCreateCmd(t *testing.T) {
	logger = initTestLogger(t)
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	namespace = "default"

	cmd := newLeaseCreateCmd()
	cmd.SetArgs([]string{"test-lease", "--ttl", "1h"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)

	_ = buf.String()
}

func TestLeaseDeleteCmd(t *testing.T) {
	logger = initTestLogger(t)
	scheme := runtime.NewScheme()
	require.NoError(t, syncv1.AddToScheme(scheme))

	lease := &syncv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease",
			Namespace: "default",
		},
	}

	k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(lease).
		Build()
	namespace = "default"

	cmd := newLeaseDeleteCmd()
	cmd.SetArgs([]string{"test-lease"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.Execute()
	require.NoError(t, err)

	_ = buf.String()
}
