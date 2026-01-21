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

func TestE2ERWMutex(t *testing.T) {
	k8sClient, err := setupClient()
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}

	namespace := "default"
	rwmutexName := fmt.Sprintf("e2e-test-rwmutex-%d", time.Now().Unix())

	// Create rwmutex using CLI
	cmd := exec.Command("../bin/koncli", "rwmutex", "create", rwmutexName, "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create rwmutex: %v, output: %s", err, string(output))
	}

	t.Logf("Created rwmutex: %s", string(output))

	// Wait for rwmutex to be ready
	err = wait.PollImmediate(2*time.Second, 10*time.Second, func() (bool, error) {
		rwmutex := &syncv1.RWMutex{}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: rwmutexName, Namespace: namespace}, rwmutex)
		if err != nil {
			t.Logf("Waiting for rwmutex %s: %v", rwmutexName, err)
			return false, nil
		}
		t.Logf("RWMutex %s found with status: %+v", rwmutexName, rwmutex.Status)
		return true, nil
	})
	if err != nil {
		t.Fatalf("RWMutex was not ready: %v", err)
	}

	// Acquire read lock using CLI
	cmd = exec.Command("../bin/koncli", "rwmutex", "rlock", rwmutexName, "--holder", "reader-1", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to acquire read lock: %v, output: %s", err, string(output))
	}

	t.Logf("Acquired read lock: %s", string(output))

	// Verify rwmutex state
	rwmutex := &syncv1.RWMutex{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: rwmutexName, Namespace: namespace}, rwmutex)
	if err != nil {
		t.Fatalf("Failed to get rwmutex: %v", err)
	}

	if rwmutex.Status.Phase != syncv1.RWMutexPhaseReadLocked {
		t.Errorf("Expected phase 'ReadLocked', got '%s'", rwmutex.Status.Phase)
	}

	if len(rwmutex.Status.ReadHolders) != 1 || rwmutex.Status.ReadHolders[0] != "reader-1" {
		t.Errorf("Expected read holder 'reader-1', got %v", rwmutex.Status.ReadHolders)
	}

	// Release read lock using CLI
	cmd = exec.Command("../bin/koncli", "rwmutex", "unlock", rwmutexName, "--holder", "reader-1", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to unlock rwmutex: %v, output: %s", err, string(output))
	}

	t.Logf("Unlocked rwmutex: %s", string(output))

	// Verify rwmutex is unlocked
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: rwmutexName, Namespace: namespace}, rwmutex)
	if err != nil {
		t.Fatalf("Failed to get rwmutex after unlock: %v", err)
	}

	if rwmutex.Status.Phase != syncv1.RWMutexPhaseUnlocked {
		t.Errorf("Expected phase 'Unlocked', got '%s'", rwmutex.Status.Phase)
	}

	// Cleanup
	cmd = exec.Command("../bin/koncli", "rwmutex", "delete", rwmutexName, "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to delete rwmutex: %v, output: %s", err, string(output))
	}

	t.Logf("Deleted rwmutex: %s", string(output))
}

func TestE2ERWMutexMultipleReaders(t *testing.T) {
	k8sClient, err := setupClient()
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}

	namespace := "default"
	rwmutexName := fmt.Sprintf("e2e-test-rwmutex-readers-%d", time.Now().Unix())

	// Create rwmutex
	cmd := exec.Command("../bin/koncli", "rwmutex", "create", rwmutexName, "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create rwmutex: %v, output: %s", err, string(output))
	}

	// Wait for rwmutex to be ready
	err = wait.PollImmediate(2*time.Second, 10*time.Second, func() (bool, error) {
		rwmutex := &syncv1.RWMutex{}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: rwmutexName, Namespace: namespace}, rwmutex)
		return err == nil, nil
	})
	if err != nil {
		t.Fatalf("RWMutex was not ready: %v", err)
	}

	// Acquire multiple read locks
	for i := 1; i <= 3; i++ {
		holder := fmt.Sprintf("reader-%d", i)
		cmd = exec.Command("../bin/koncli", "rwmutex", "rlock", rwmutexName, "--holder", holder, "-n", namespace)
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to acquire read lock for %s: %v, output: %s", holder, err, string(output))
		}
		t.Logf("Acquired read lock for %s", holder)
	}

	// Verify all readers are present
	rwmutex := &syncv1.RWMutex{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: rwmutexName, Namespace: namespace}, rwmutex)
	if err != nil {
		t.Fatalf("Failed to get rwmutex: %v", err)
	}

	if len(rwmutex.Status.ReadHolders) != 3 {
		t.Errorf("Expected 3 read holders, got %d", len(rwmutex.Status.ReadHolders))
	}

	// Release all read locks
	for i := 1; i <= 3; i++ {
		holder := fmt.Sprintf("reader-%d", i)
		cmd = exec.Command("../bin/koncli", "rwmutex", "unlock", rwmutexName, "--holder", holder, "-n", namespace)
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to unlock for %s: %v, output: %s", holder, err, string(output))
		}
	}

	// Cleanup
	cmd = exec.Command("../bin/koncli", "rwmutex", "delete", rwmutexName, "-n", namespace)
	cmd.CombinedOutput()
}

func TestE2ERWMutexWriteLock(t *testing.T) {
	k8sClient, err := setupClient()
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}

	namespace := "default"
	rwmutexName := fmt.Sprintf("e2e-test-rwmutex-write-%d", time.Now().Unix())

	// Create rwmutex
	cmd := exec.Command("../bin/koncli", "rwmutex", "create", rwmutexName, "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create rwmutex: %v, output: %s", err, string(output))
	}

	// Wait for rwmutex to be ready
	err = wait.PollImmediate(2*time.Second, 10*time.Second, func() (bool, error) {
		rwmutex := &syncv1.RWMutex{}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: rwmutexName, Namespace: namespace}, rwmutex)
		return err == nil, nil
	})
	if err != nil {
		t.Fatalf("RWMutex was not ready: %v", err)
	}

	// Acquire write lock
	cmd = exec.Command("../bin/koncli", "rwmutex", "lock", rwmutexName, "--holder", "writer-1", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to acquire write lock: %v, output: %s", err, string(output))
	}

	// Verify write lock
	rwmutex := &syncv1.RWMutex{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: rwmutexName, Namespace: namespace}, rwmutex)
	if err != nil {
		t.Fatalf("Failed to get rwmutex: %v", err)
	}

	if rwmutex.Status.Phase != syncv1.RWMutexPhaseWriteLocked {
		t.Errorf("Expected phase 'WriteLocked', got '%s'", rwmutex.Status.Phase)
	}

	if rwmutex.Status.WriteHolder != "writer-1" {
		t.Errorf("Expected write holder 'writer-1', got '%s'", rwmutex.Status.WriteHolder)
	}

	// Try to acquire read lock (should timeout)
	cmd = exec.Command("../bin/koncli", "rwmutex", "rlock", rwmutexName, "--holder", "reader-1", "--timeout", "2s", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err == nil {
		t.Error("Expected reader to fail acquiring lock while write locked, but it succeeded")
	}
	t.Logf("Reader correctly failed to acquire lock: %s", string(output))

	// Release write lock
	cmd = exec.Command("../bin/koncli", "rwmutex", "unlock", rwmutexName, "--holder", "writer-1", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to unlock: %v, output: %s", err, string(output))
	}

	// Cleanup
	cmd = exec.Command("../bin/koncli", "rwmutex", "delete", rwmutexName, "-n", namespace)
	cmd.CombinedOutput()
}

func TestE2ERWMutexWithTTL(t *testing.T) {
	k8sClient, err := setupClient()
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}

	namespace := "default"
	rwmutexName := fmt.Sprintf("e2e-test-rwmutex-ttl-%d", time.Now().Unix())

	// Create rwmutex with TTL
	cmd := exec.Command("../bin/koncli", "rwmutex", "create", rwmutexName, "--ttl", "10s", "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create rwmutex: %v, output: %s", err, string(output))
	}

	// Wait for rwmutex to be ready
	err = wait.PollImmediate(2*time.Second, 10*time.Second, func() (bool, error) {
		rwmutex := &syncv1.RWMutex{}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: rwmutexName, Namespace: namespace}, rwmutex)
		return err == nil, nil
	})
	if err != nil {
		t.Fatalf("RWMutex was not ready: %v", err)
	}

	// Acquire write lock
	cmd = exec.Command("../bin/koncli", "rwmutex", "lock", rwmutexName, "--holder", "writer-1", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to acquire write lock: %v, output: %s", err, string(output))
	}

	// Verify lock and TTL
	rwmutex := &syncv1.RWMutex{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: rwmutexName, Namespace: namespace}, rwmutex)
	if err != nil {
		t.Fatalf("Failed to get rwmutex: %v", err)
	}

	if rwmutex.Status.Phase != syncv1.RWMutexPhaseWriteLocked {
		t.Errorf("Expected phase 'WriteLocked', got '%s'", rwmutex.Status.Phase)
	}

	if rwmutex.Status.ExpiresAt == nil {
		t.Error("Expected ExpiresAt to be set")
	}

	// Wait for TTL expiration
	t.Logf("Waiting for TTL expiration...")
	err = wait.PollImmediate(1*time.Second, 15*time.Second, func() (bool, error) {
		rwmutex := &syncv1.RWMutex{}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: rwmutexName, Namespace: namespace}, rwmutex)
		if err != nil {
			return false, err
		}
		return rwmutex.Status.Phase == syncv1.RWMutexPhaseUnlocked, nil
	})
	if err != nil {
		t.Fatalf("RWMutex did not unlock after TTL: %v", err)
	}

	// Verify auto-unlock
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: rwmutexName, Namespace: namespace}, rwmutex)
	if err != nil {
		t.Fatalf("Failed to get rwmutex after TTL: %v", err)
	}

	if rwmutex.Status.Phase != syncv1.RWMutexPhaseUnlocked {
		t.Errorf("Expected phase 'Unlocked' after TTL, got '%s'", rwmutex.Status.Phase)
	}

	// Cleanup
	cmd = exec.Command("../bin/koncli", "rwmutex", "delete", rwmutexName, "-n", namespace)
	cmd.CombinedOutput()
}

func TestE2ERWMutexList(t *testing.T) {
	k8sClient, err := setupClient()
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}

	namespace := "default"
	rwmutexName := fmt.Sprintf("e2e-test-rwmutex-list-%d", time.Now().Unix())

	// Create rwmutex
	cmd := exec.Command("../bin/koncli", "rwmutex", "create", rwmutexName, "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create rwmutex: %v, output: %s", err, string(output))
	}

	// Wait for rwmutex to be ready
	err = wait.PollImmediate(2*time.Second, 10*time.Second, func() (bool, error) {
		rwmutex := &syncv1.RWMutex{}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: rwmutexName, Namespace: namespace}, rwmutex)
		return err == nil, nil
	})
	if err != nil {
		t.Fatalf("RWMutex was not ready: %v", err)
	}

	// List rwmutexes
	cmd = exec.Command("../bin/koncli", "rwmutex", "list", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to list rwmutexes: %v, output: %s", err, string(output))
	}

	t.Logf("RWMutex list output: %s", string(output))

	// Cleanup
	cmd = exec.Command("../bin/koncli", "rwmutex", "delete", rwmutexName, "-n", namespace)
	cmd.CombinedOutput()
}
