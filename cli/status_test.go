package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

func _TestStatusCommands(t *testing.T) {
	scheme := runtime.NewScheme()
	err := syncv1.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}

	// Create test resources
	now := metav1.Now()
	testResources := []client.Object{
		&syncv1.Semaphore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-semaphore",
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
		&syncv1.Barrier{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-barrier",
				Namespace: "default",
			},
			Spec: syncv1.BarrierSpec{
				Expected: 5,
			},
			Status: syncv1.BarrierStatus{
				Arrived:  5,
				Phase:    syncv1.BarrierPhaseOpen,
				OpenedAt: &now,
			},
		},
		&syncv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-lease",
				Namespace: "default",
			},
			Spec: syncv1.LeaseSpec{
				TTL: metav1.Duration{Duration: 5 * time.Minute},
			},
			Status: syncv1.LeaseStatus{
				Holder:     "test-holder",
				Phase:      syncv1.LeasePhaseHeld,
				AcquiredAt: &now,
			},
		},
		&syncv1.Gate{
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
				Phase: syncv1.GatePhaseOpen,
				ConditionStatuses: []syncv1.GateConditionStatus{
					{
						Type: "Job",
						Name: "test-job",
						Met:  true,
					},
				},
				OpenedAt: &now,
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(testResources...).
		Build()

	k8sClient = fakeClient
	namespace = "default"

	tests := []struct {
		name        string
		args        []string
		expectError bool
		checkFunc   func(t *testing.T, output string)
	}{
		{
			name:        "status all",
			args:        []string{"status", "all"},
			expectError: false,
			checkFunc: func(t *testing.T, output string) {
				expectedStrings := []string{
					"Semaphores",
					"Barriers",
					"Leases",
					"Gates",
					"test-semaphore",
					"test-barrier",
					"test-lease",
					"test-gate",
				}
				for _, expected := range expectedStrings {
					if !bytes.Contains([]byte(output), []byte(expected)) {
						t.Errorf("Expected output to contain '%s', got: %s", expected, output)
					}
				}
			},
		},
		{
			name:        "status semaphore",
			args:        []string{"status", "semaphore", "test-semaphore"},
			expectError: false,
			checkFunc: func(t *testing.T, output string) {
				expectedStrings := []string{
					"test-semaphore",
					"permits_total",
					"permits_in_use",
					"Ready",
				}
				for _, expected := range expectedStrings {
					if !bytes.Contains([]byte(output), []byte(expected)) {
						t.Errorf("Expected output to contain '%s', got: %s", expected, output)
					}
				}
			},
		},
		{
			name:        "status barrier",
			args:        []string{"status", "barrier", "test-barrier"},
			expectError: false,
			checkFunc: func(t *testing.T, output string) {
				expectedStrings := []string{
					"test-barrier",
					"expected",
					"arrived",
					"Open",
				}
				for _, expected := range expectedStrings {
					if !bytes.Contains([]byte(output), []byte(expected)) {
						t.Errorf("Expected output to contain '%s', got: %s", expected, output)
					}
				}
			},
		},
		{
			name:        "status lease",
			args:        []string{"status", "lease", "test-lease"},
			expectError: false,
			checkFunc: func(t *testing.T, output string) {
				expectedStrings := []string{
					"test-lease",
					"holder",
					"test-holder",
					"Held",
				}
				for _, expected := range expectedStrings {
					if !bytes.Contains([]byte(output), []byte(expected)) {
						t.Errorf("Expected output to contain '%s', got: %s", expected, output)
					}
				}
			},
		},
		{
			name:        "status gate",
			args:        []string{"status", "gate", "test-gate"},
			expectError: false,
			checkFunc: func(t *testing.T, output string) {
				expectedStrings := []string{
					"test-gate",
					"Open",
					"met",
					"test-job",
				}
				for _, expected := range expectedStrings {
					if !bytes.Contains([]byte(output), []byte(expected)) {
						t.Errorf("Expected output to contain '%s', got: %s", expected, output)
					}
				}
			},
		},
		{
			name:        "status semaphore - missing name",
			args:        []string{"status", "semaphore"},
			expectError: true,
			checkFunc:   nil,
		},
		{
			name:        "status nonexistent resource",
			args:        []string{"status", "semaphore", "nonexistent"},
			expectError: true,
			checkFunc:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := &cobra.Command{Use: "koncli"}
			rootCmd.AddCommand(newStatusCmd())

			var output bytes.Buffer
			rootCmd.SetOut(&output)
			rootCmd.SetErr(&output)

			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, output.String())
			}
		})
	}
}

func TestStatusAllEmpty(t *testing.T) {
	scheme := runtime.NewScheme()
	err := syncv1.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}

	// Create client with no resources
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	k8sClient = fakeClient
	namespace = "default"

	// Initialize logger to capture output
	var logBuf bytes.Buffer
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.TimeKey = ""
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(&logBuf),
		zapcore.DebugLevel,
	)
	logger = zap.New(core)

	rootCmd := &cobra.Command{Use: "koncli"}
	rootCmd.AddCommand(newStatusCmd())

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetArgs([]string{"status", "all"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	outputStr := output.String() + logBuf.String()

	// Check that structured logs contain expected data
	expectedStrings := []string{
		"Semaphores",
		"Barriers",
		"Leases",
		"Gates",
		"count",
	}

	for _, expected := range expectedStrings {
		if !bytes.Contains([]byte(outputStr), []byte(expected)) {
			t.Errorf("Expected output to contain '%s', got: %s", expected, outputStr)
		}
	}
}
