//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

const testTimeout = 30 * time.Second

func TestE2ELease(t *testing.T) {
	k8sClient, err := setupClient()
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}

	koncliPath := getKoncliPath()
	if _, err := os.Stat(koncliPath); err != nil {
		t.Fatalf("koncli binary not found at %s: %v", koncliPath, err)
	}

	// Check operator status first
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	statusCmd := exec.CommandContext(ctx, koncliPath, "operator", "--operator-namespace", "konductor-system")
	statusOutput, statusErr := statusCmd.CombinedOutput()
	if statusErr != nil {
		t.Logf("Operator status check failed: %v, output: %s", statusErr, string(statusOutput))
	}
	t.Logf("Operator status: %s", string(statusOutput))

	namespace := "default"
	leaseName := fmt.Sprintf("e2e-test-lease-%d", time.Now().Unix())

	// Create lease using CLI
	ctx, cancel = context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, koncliPath, "lease", "create", leaseName, "--ttl", "1m", "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create lease: %v, output: %s", err, string(output))
	}

	t.Logf("Created lease: %s", string(output))

	// Wait for lease to be ready
	err = wait.PollImmediate(2*time.Second, 10*time.Second, func() (bool, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		lease := &syncv1.Lease{}
		err := k8sClient.Get(ctx, client.ObjectKey{Name: leaseName, Namespace: namespace}, lease)
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
	ctx, cancel = context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	cmd = exec.CommandContext(ctx, koncliPath, "lease", "acquire", leaseName, "--holder", "worker-1", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to acquire lease: %v, output: %s", err, string(output))
	}

	// Verify lease state
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lease := &syncv1.Lease{}
	err = k8sClient.Get(ctx, client.ObjectKey{Name: leaseName, Namespace: namespace}, lease)
	if err != nil {
		t.Fatalf("Failed to get lease: %v", err)
	}

	if lease.Status.Holder != "worker-1" {
		t.Errorf("Expected holder 'worker-1', got '%s'", lease.Status.Holder)
	}

	// Release lease using CLI
	ctx, cancel = context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	cmd = exec.CommandContext(ctx, koncliPath, "lease", "release", leaseName, "--holder", "worker-1", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to release lease: %v, output: %s", err, string(output))
	}

	// Cleanup
	ctx, cancel = context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	cmd = exec.CommandContext(ctx, koncliPath, "lease", "delete", leaseName, "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to delete lease: %v, output: %s", err, string(output))
	}
}