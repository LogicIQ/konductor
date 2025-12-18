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

	namespace := "default"
	semaphoreName := fmt.Sprintf("e2e-test-semaphore-%d", time.Now().Unix())

	// Create semaphore using CLI
	cmd := exec.Command("../bin/koncli", "semaphore", "create", semaphoreName, "--permits", "3", "--ttl", "5m", "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create semaphore: %v, output: %s", err, string(output))
	}

	// Wait for semaphore to be ready
	err = wait.PollImmediate(2*time.Second, 60*time.Second, func() (bool, error) {
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

	// Verify semaphore state
	semaphore := &syncv1.Semaphore{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: semaphoreName, Namespace: namespace}, semaphore)
	if err != nil {
		t.Fatalf("Failed to get semaphore: %v", err)
	}

	if semaphore.Status.Available != 1 {
		t.Errorf("Expected 1 available permit, got %d", semaphore.Status.Available)
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