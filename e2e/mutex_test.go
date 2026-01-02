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

func TestE2EMutex(t *testing.T) {
	k8sClient, err := setupClient()
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}

	namespace := "default"
	mutexName := fmt.Sprintf("e2e-test-mutex-%d", time.Now().Unix())

	// Create mutex using CLI
	cmd := exec.Command("../bin/koncli", "mutex", "create", mutexName, "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create mutex: %v, output: %s", err, string(output))
	}

	t.Logf("Created mutex: %s", string(output))

	// Wait for mutex to be ready
	err = wait.PollImmediate(2*time.Second, 10*time.Second, func() (bool, error) {
		mutex := &syncv1.Mutex{}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: mutexName, Namespace: namespace}, mutex)
		if err != nil {
			t.Logf("Waiting for mutex %s: %v", mutexName, err)
			return false, nil
		}
		t.Logf("Mutex %s found with status: %+v", mutexName, mutex.Status)
		return true, nil
	})
	if err != nil {
		t.Fatalf("Mutex was not ready: %v", err)
	}

	// Lock mutex using CLI
	cmd = exec.Command("../bin/koncli", "mutex", "lock", mutexName, "--holder", "worker-1", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to lock mutex: %v, output: %s", err, string(output))
	}

	t.Logf("Locked mutex: %s", string(output))

	// Verify mutex state
	mutex := &syncv1.Mutex{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: mutexName, Namespace: namespace}, mutex)
	if err != nil {
		t.Fatalf("Failed to get mutex: %v", err)
	}

	if mutex.Status.Phase != syncv1.MutexPhaseLocked {
		t.Errorf("Expected phase 'Locked', got '%s'", mutex.Status.Phase)
	}

	if mutex.Status.Holder != "worker-1" {
		t.Errorf("Expected holder 'worker-1', got '%s'", mutex.Status.Holder)
	}

	// Unlock mutex using CLI
	cmd = exec.Command("../bin/koncli", "mutex", "unlock", mutexName, "--holder", "worker-1", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to unlock mutex: %v, output: %s", err, string(output))
	}

	t.Logf("Unlocked mutex: %s", string(output))

	// Verify mutex is unlocked
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: mutexName, Namespace: namespace}, mutex)
	if err != nil {
		t.Fatalf("Failed to get mutex after unlock: %v", err)
	}

	if mutex.Status.Phase != syncv1.MutexPhaseUnlocked {
		t.Errorf("Expected phase 'Unlocked', got '%s'", mutex.Status.Phase)
	}

	if mutex.Status.Holder != "" {
		t.Errorf("Expected no holder, got '%s'", mutex.Status.Holder)
	}

	// Cleanup
	cmd = exec.Command("../bin/koncli", "mutex", "delete", mutexName, "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to delete mutex: %v, output: %s", err, string(output))
	}

	t.Logf("Deleted mutex: %s", string(output))
}

func TestE2EMutexWithTTL(t *testing.T) {
	k8sClient, err := setupClient()
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}

	namespace := "default"
	mutexName := fmt.Sprintf("e2e-test-mutex-ttl-%d", time.Now().Unix())

	// Create mutex with TTL using CLI
	cmd := exec.Command("../bin/koncli", "mutex", "create", mutexName, "--ttl", "10s", "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create mutex: %v, output: %s", err, string(output))
	}

	t.Logf("Created mutex with TTL: %s", string(output))

	// Wait for mutex to be ready
	err = wait.PollImmediate(2*time.Second, 10*time.Second, func() (bool, error) {
		mutex := &syncv1.Mutex{}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: mutexName, Namespace: namespace}, mutex)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("Mutex was not ready: %v", err)
	}

	// Lock mutex
	cmd = exec.Command("../bin/koncli", "mutex", "lock", mutexName, "--holder", "worker-1", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to lock mutex: %v, output: %s", err, string(output))
	}

	// Verify mutex is locked
	mutex := &syncv1.Mutex{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: mutexName, Namespace: namespace}, mutex)
	if err != nil {
		t.Fatalf("Failed to get mutex: %v", err)
	}

	if mutex.Status.Phase != syncv1.MutexPhaseLocked {
		t.Errorf("Expected phase 'Locked', got '%s'", mutex.Status.Phase)
	}

	if mutex.Status.ExpiresAt == nil {
		t.Error("Expected ExpiresAt to be set")
	}

	// Wait for TTL expiration
	t.Logf("Waiting for TTL expiration...")
	time.Sleep(12 * time.Second)

	// Verify mutex is auto-unlocked
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: mutexName, Namespace: namespace}, mutex)
	if err != nil {
		t.Fatalf("Failed to get mutex after TTL: %v", err)
	}

	if mutex.Status.Phase != syncv1.MutexPhaseUnlocked {
		t.Errorf("Expected phase 'Unlocked' after TTL, got '%s'", mutex.Status.Phase)
	}

	// Cleanup
	cmd = exec.Command("../bin/koncli", "mutex", "delete", mutexName, "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to delete mutex: %v, output: %s", err, string(output))
	}
}

func TestE2EMutexConcurrency(t *testing.T) {
	k8sClient, err := setupClient()
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}

	namespace := "default"
	mutexName := fmt.Sprintf("e2e-test-mutex-concurrent-%d", time.Now().Unix())

	// Create mutex
	cmd := exec.Command("../bin/koncli", "mutex", "create", mutexName, "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create mutex: %v, output: %s", err, string(output))
	}

	// Wait for mutex to be ready
	err = wait.PollImmediate(2*time.Second, 10*time.Second, func() (bool, error) {
		mutex := &syncv1.Mutex{}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: mutexName, Namespace: namespace}, mutex)
		return err == nil, nil
	})
	if err != nil {
		t.Fatalf("Mutex was not ready: %v", err)
	}

	// Worker 1 locks the mutex
	cmd = exec.Command("../bin/koncli", "mutex", "lock", mutexName, "--holder", "worker-1", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to lock mutex: %v, output: %s", err, string(output))
	}

	// Worker 2 tries to lock (should timeout)
	cmd = exec.Command("../bin/koncli", "mutex", "lock", mutexName, "--holder", "worker-2", "--timeout", "2s", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err == nil {
		t.Error("Expected worker-2 to fail acquiring lock, but it succeeded")
	}
	t.Logf("Worker-2 correctly failed to acquire lock: %s", string(output))

	// Verify mutex is still held by worker-1
	mutex := &syncv1.Mutex{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: mutexName, Namespace: namespace}, mutex)
	if err != nil {
		t.Fatalf("Failed to get mutex: %v", err)
	}

	if mutex.Status.Holder != "worker-1" {
		t.Errorf("Expected holder 'worker-1', got '%s'", mutex.Status.Holder)
	}

	// Worker 1 unlocks
	cmd = exec.Command("../bin/koncli", "mutex", "unlock", mutexName, "--holder", "worker-1", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to unlock mutex: %v, output: %s", err, string(output))
	}

	// Worker 2 can now lock
	cmd = exec.Command("../bin/koncli", "mutex", "lock", mutexName, "--holder", "worker-2", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to lock mutex for worker-2: %v, output: %s", err, string(output))
	}

	// Verify mutex is held by worker-2
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: mutexName, Namespace: namespace}, mutex)
	if err != nil {
		t.Fatalf("Failed to get mutex: %v", err)
	}

	if mutex.Status.Holder != "worker-2" {
		t.Errorf("Expected holder 'worker-2', got '%s'", mutex.Status.Holder)
	}

	// Cleanup
	cmd = exec.Command("../bin/koncli", "mutex", "unlock", mutexName, "--holder", "worker-2", "-n", namespace)
	cmd.CombinedOutput()

	cmd = exec.Command("../bin/koncli", "mutex", "delete", mutexName, "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to delete mutex: %v, output: %s", err, string(output))
	}
}

func TestE2EMutexList(t *testing.T) {
	k8sClient, err := setupClient()
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}

	namespace := "default"
	mutexName := fmt.Sprintf("e2e-test-mutex-list-%d", time.Now().Unix())

	// Create mutex
	cmd := exec.Command("../bin/koncli", "mutex", "create", mutexName, "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create mutex: %v, output: %s", err, string(output))
	}

	// Wait for mutex to be ready
	err = wait.PollImmediate(2*time.Second, 10*time.Second, func() (bool, error) {
		mutex := &syncv1.Mutex{}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: mutexName, Namespace: namespace}, mutex)
		return err == nil, nil
	})
	if err != nil {
		t.Fatalf("Mutex was not ready: %v", err)
	}

	// List mutexes
	cmd = exec.Command("../bin/koncli", "mutex", "list", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to list mutexes: %v, output: %s", err, string(output))
	}

	t.Logf("Mutex list output: %s", string(output))

	// Cleanup
	cmd = exec.Command("../bin/koncli", "mutex", "delete", mutexName, "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to delete mutex: %v, output: %s", err, string(output))
	}
}
