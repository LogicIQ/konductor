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

func TestE2EBarrier(t *testing.T) {
	k8sClient, err := setupClient()
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}

	namespace := "default"
	barrierName := fmt.Sprintf("e2e-test-barrier-%d", time.Now().Unix())

	// Create barrier using CLI
	cmd := exec.Command("../bin/koncli", "barrier", "create", barrierName, "--expected", "2", "--timeout", "2m", "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create barrier: %v, output: %s", err, string(output))
	}

	// Wait for barrier to be ready
	err = wait.PollImmediate(2*time.Second, 60*time.Second, func() (bool, error) {
		barrier := &syncv1.Barrier{}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: barrierName, Namespace: namespace}, barrier)
		return err == nil, nil
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
		t.Fatalf("Failed to delete barrier: %v, output: %s", err, string(output))
	}
}