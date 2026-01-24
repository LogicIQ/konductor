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

func waitForWaitGroupReady(k8sClient client.Client, wgName, namespace string) error {
	return wait.PollImmediate(2*time.Second, 10*time.Second, func() (bool, error) {
		wg := &syncv1.WaitGroup{}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: wgName, Namespace: namespace}, wg)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
}

func TestE2EWaitGroup(t *testing.T) {
	k8sClient, err := setupClient()
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}

	namespace := "default"
	wgName := fmt.Sprintf("e2e-test-wg-%d", time.Now().Unix())

	// Create waitgroup
	cmd := exec.Command("../bin/koncli", "waitgroup", "create", wgName, "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create waitgroup: %v, output: %s", err, string(output))
	}

	// Cleanup
	defer func() {
		cmd := exec.Command("../bin/koncli", "waitgroup", "delete", wgName, "-n", namespace)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Logf("Cleanup failed: %v, output: %s", err, string(output))
		}
	}()

	t.Logf("Created waitgroup: %s", string(output))

	// Wait for waitgroup to be ready
	if err := waitForWaitGroupReady(k8sClient, wgName, namespace); err != nil {
		t.Fatalf("WaitGroup was not ready: %v", err)
	}

	// Add to counter
	cmd = exec.Command("../bin/koncli", "waitgroup", "add", wgName, "--delta", "3", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to add: %v, output: %s", err, string(output))
	}

	// Verify counter
	wg := &syncv1.WaitGroup{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: wgName, Namespace: namespace}, wg)
	if err != nil {
		t.Fatalf("Failed to get waitgroup: %v", err)
	}

	if wg.Status.Counter != 3 {
		t.Errorf("Expected counter 3, got %d", wg.Status.Counter)
	}

	// Call done
	cmd = exec.Command("../bin/koncli", "waitgroup", "done", wgName, "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to done: %v, output: %s", err, string(output))
	}
}

func TestE2EWaitGroupList(t *testing.T) {
	k8sClient, err := setupClient()
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}

	namespace := "default"
	wgName := fmt.Sprintf("e2e-test-wg-list-%d", time.Now().Unix())

	// Create waitgroup
	cmd := exec.Command("../bin/koncli", "waitgroup", "create", wgName, "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create waitgroup: %v, output: %s", err, string(output))
	}

	// Cleanup
	defer func() {
		cmd := exec.Command("../bin/koncli", "waitgroup", "delete", wgName, "-n", namespace)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Logf("Cleanup failed: %v, output: %s", err, string(output))
		}
	}()

	// Wait for ready
	if err := waitForWaitGroupReady(k8sClient, wgName, namespace); err != nil {
		t.Fatalf("WaitGroup was not ready: %v", err)
	}

	// List waitgroups
	cmd = exec.Command("../bin/koncli", "waitgroup", "list", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to list: %v, output: %s", err, string(output))
	}

	t.Logf("WaitGroup list: %s", string(output))
}
