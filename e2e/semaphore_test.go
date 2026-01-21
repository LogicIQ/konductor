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

func TestE2ESemaphore(t *testing.T) {
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
	semaphoreName := fmt.Sprintf("e2e-test-semaphore-%d", time.Now().Unix())

	// Create semaphore using CLI
	cmd := exec.Command("../bin/koncli", "semaphore", "create", semaphoreName, "--permits", "3", "--ttl", "5m", "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create semaphore: %v, output: %s", err, string(output))
	}

	// Wait for semaphore to be ready
	err = wait.PollImmediate(2*time.Second, 10*time.Second, func() (bool, error) {
		semaphore := &syncv1.Semaphore{}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: semaphoreName, Namespace: namespace}, semaphore)
		if err != nil {
			return false, nil
		}
		return semaphore.Status.Available == 3, nil
	})
	if err != nil {
		t.Fatalf("Semaphore was not ready: %v", err)
	}

	// Acquire permits using CLI
	cmd = exec.Command("../bin/koncli", "semaphore", "acquire", semaphoreName, "--holder", "worker-1", "--ttl", "2m", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to acquire permit: %v, output: %s", err, string(output))
	}

	cmd = exec.Command("../bin/koncli", "semaphore", "acquire", semaphoreName, "--holder", "worker-2", "--ttl", "2m", "--wait-for-update", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to acquire permit: %v, output: %s", err, string(output))
	}

	// Wait for controller to process permit acquisitions
	err = wait.PollImmediate(2*time.Second, 10*time.Second, func() (bool, error) {
		semaphore := &syncv1.Semaphore{}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: semaphoreName, Namespace: namespace}, semaphore)
		if err != nil {
			return false, nil
		}
		return semaphore.Status.Available == 1, nil
	})
	if err != nil {
		// Get final state for debugging
		semaphore := &syncv1.Semaphore{}
		if getErr := k8sClient.Get(context.TODO(), client.ObjectKey{Name: semaphoreName, Namespace: namespace}, semaphore); getErr != nil {
			t.Errorf("Failed to get semaphore for debugging: %v", getErr)
		} else {
			t.Errorf("Expected 1 available permit, got %d (controller may not be running)", semaphore.Status.Available)
		}
	}

	// Release permit using CLI
	cmd = exec.Command("../bin/koncli", "semaphore", "release", semaphoreName, "--holder", "worker-1", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to release permit: %v, output: %s", err, string(output))
	}

	// Cleanup
	cmd = exec.Command("../bin/koncli", "semaphore", "delete", semaphoreName, "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to delete semaphore: %v, output: %s", err, string(output))
	}
}