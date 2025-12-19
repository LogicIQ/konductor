//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestE2EBarrier(t *testing.T) {
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
	barrierName := fmt.Sprintf("e2e-test-barrier-%d", time.Now().Unix())

	// Create barrier using CLI
	cmd := exec.Command("../bin/koncli", "barrier", "create", barrierName, "--expected", "2", "--timeout", "2m", "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create barrier: %v, output: %s", err, string(output))
	}

	t.Logf("Created barrier: %s", string(output))

	// Wait for barrier to be ready
	err = wait.PollImmediate(2*time.Second, 10*time.Second, func() (bool, error) {
		barrier := &syncv1.Barrier{}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: barrierName, Namespace: namespace}, barrier)
		if err != nil {
			t.Logf("Waiting for barrier %s: %v", barrierName, err)
			return false, nil
		}
		t.Logf("Barrier %s found with status: %+v", barrierName, barrier.Status)
		return true, nil
	})
	if err != nil {
		t.Fatalf("Barrier was not ready: %v", err)
	}

	// Arrive at barrier using CLI
	cmd = exec.Command("../bin/koncli", "barrier", "arrive", barrierName, "worker-1", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to arrive at barrier: %v, output: %s", err, string(output))
	}

	cmd = exec.Command("../bin/koncli", "barrier", "arrive", barrierName, "worker-2", "--wait-for-update", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to arrive at barrier: %v, output: %s", err, string(output))
	}

	// Verify barrier state
	barrier := &syncv1.Barrier{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: barrierName, Namespace: namespace}, barrier)
	if err != nil {
		t.Fatalf("Failed to get barrier: %v", err)
	}

	if barrier.Status.Arrived != 2 {
		t.Errorf("Expected 2 arrivals, got %d", barrier.Status.Arrived)
	}

	// Cleanup
	cmd = exec.Command("../bin/koncli", "barrier", "delete", barrierName, "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		// Don't fail if barrier doesn't exist (might have been auto-deleted)
		if !contains(string(output), "not found") {
			t.Fatalf("Failed to delete barrier: %v, output: %s", err, string(output))
		}
		t.Logf("Barrier already deleted or not found: %s", string(output))
	}
}