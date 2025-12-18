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

	namespace := "default"
	gateName := fmt.Sprintf("e2e-test-gate-%d", time.Now().Unix())

	// Create gate using CLI (closed by default)
	cmd := exec.Command("../bin/koncli", "gate", "create", gateName, "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create gate: %v, output: %s", err, string(output))
	}

	// Wait for gate to be ready
	err = wait.PollImmediate(2*time.Second, 60*time.Second, func() (bool, error) {
		gate := &syncv1.Gate{}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: gateName, Namespace: namespace}, gate)
		return err == nil, nil
	})
	if err != nil {
		t.Fatalf("Gate was not ready: %v", err)
	}

	// Verify gate is closed
	gate := &syncv1.Gate{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: gateName, Namespace: namespace}, gate)
	if err != nil {
		t.Fatalf("Failed to get gate: %v", err)
	}

	if gate.Status.Phase != syncv1.GatePhaseOpen {
		t.Error("Expected gate with no conditions to be open initially")
	}

	// Open gate using CLI
	cmd = exec.Command("../bin/koncli", "gate", "open", gateName, "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to open gate: %v, output: %s", err, string(output))
	}

	// Verify gate is open
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: gateName, Namespace: namespace}, gate)
	if err != nil {
		t.Fatalf("Failed to get gate: %v", err)
	}

	if gate.Status.Phase != syncv1.GatePhaseOpen {
		t.Error("Expected gate to be open")
	}

	// Close gate using CLI
	cmd = exec.Command("../bin/koncli", "gate", "close", gateName, "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to close gate: %v, output: %s", err, string(output))
	}

	// Cleanup
	cmd = exec.Command("../bin/koncli", "gate", "delete", gateName, "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to delete gate: %v, output: %s", err, string(output))
	}
}