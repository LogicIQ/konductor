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

func TestE2ELease(t *testing.T) {
	k8sClient, err := setupClient()
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}

	// Check operator status first
	cmd := exec.Command("../bin/koncli", "operator", "--operator-namespace", "konductor-system")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Operator status check failed: %v, output: %s", err, string(output))
	} else {
		t.Logf("Operator status: %s", string(output))
	}

	namespace := "default"
	leaseName := fmt.Sprintf("e2e-test-lease-%d", time.Now().Unix())

	// Create lease using CLI
	cmd := exec.Command("../bin/koncli", "lease", "create", leaseName, "--ttl", "1m", "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create lease: %v, output: %s", err, string(output))
	}

	t.Logf("Created lease: %s", string(output))

	// Wait for lease to be ready
	err = wait.PollImmediate(2*time.Second, 10*time.Second, func() (bool, error) {
		lease := &syncv1.Lease{}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: leaseName, Namespace: namespace}, lease)
		if err != nil {
			t.Logf("Waiting for lease %s: %v", leaseName, err)
			return false, nil
		}
		t.Logf("Lease %s found with status: %+v", leaseName, lease.Status)
		return true, nil
	})
	if err != nil {
		t.Fatalf("Lease was not ready: %v", err)
	}

	// Acquire lease using CLI
	cmd = exec.Command("../bin/koncli", "lease", "acquire", leaseName, "--holder", "worker-1", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to acquire lease: %v, output: %s", err, string(output))
	}

	// Verify lease state
	lease := &syncv1.Lease{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: leaseName, Namespace: namespace}, lease)
	if err != nil {
		t.Fatalf("Failed to get lease: %v", err)
	}

	if lease.Status.Holder != "worker-1" {
		t.Errorf("Expected holder 'worker-1', got '%s'", lease.Status.Holder)
	}

	// Release lease using CLI
	cmd = exec.Command("../bin/koncli", "lease", "release", leaseName, "--holder", "worker-1", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to release lease: %v, output: %s", err, string(output))
	}

	// Cleanup
	cmd = exec.Command("../bin/koncli", "lease", "delete", leaseName, "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to delete lease: %v, output: %s", err, string(output))
	}
}