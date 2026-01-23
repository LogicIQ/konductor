//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

func TestE2EGate(t *testing.T) {
	k8sClient, err := setupClient()
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}

	// Check operator status first
	statusCmd := exec.Command("../bin/koncli", "operator", "--operator-namespace", "konductor-system")
	statusOutput, statusErr := statusCmd.CombinedOutput()
	if statusErr != nil {
		t.Logf("Operator status check failed: %v, output: %s", statusErr, string(statusOutput))
	} else {
		t.Logf("Operator status: %s", string(statusOutput))
	}

	namespace := "default"
	gateName := fmt.Sprintf("e2e-test-gate-%d", time.Now().Unix())

	// Create gate using CLI (closed by default)
	cmd := exec.Command("../bin/koncli", "gate", "create", gateName, "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create gate: %v, output: %s", err, string(output))
	}

	t.Logf("Created gate: %s", string(output))

	// Wait for gate to be ready
	err = wait.PollImmediate(2*time.Second, 10*time.Second, func() (bool, error) {
		gate := &syncv1.Gate{}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: gateName, Namespace: namespace}, gate)
		if err != nil {
			t.Logf("Waiting for gate %s: %v", gateName, err)
			return false, nil
		}
		t.Logf("Gate %s found with status: %+v", gateName, gate.Status)
		return true, nil
	})
	if err != nil {
		t.Fatalf("Gate was not ready: %v", err)
	}

	// Verify gate is open initially (no conditions = open by default)
	gate := &syncv1.Gate{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: gateName, Namespace: namespace}, gate)
	if err != nil {
		t.Fatalf("Failed to get gate: %v", err)
	}

	if gate.Status.Phase != syncv1.GatePhaseOpen {
		t.Error("Expected gate with no conditions to be open initially")
	}

	// Close gate using CLI
	cmd = exec.Command("../bin/koncli", "gate", "close", gateName, "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to close gate: %v, output: %s", err, string(output))
	}

	// Verify gate is closed
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: gateName, Namespace: namespace}, gate)
	if err != nil {
		t.Fatalf("Failed to get gate: %v", err)
	}

	if gate.Status.Phase != syncv1.GatePhaseClosed {
		t.Error("Expected gate to be closed")
	}

	// Cleanup
	cmd = exec.Command("../bin/koncli", "gate", "delete", gateName, "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to delete gate: %v, output: %s", err, string(output))
	}
}