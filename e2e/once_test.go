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

func createOnce(t *testing.T, name, namespace string, extraArgs ...string) {
	args := []string{"once", "create", name, "-n", namespace}
	args = append(args, extraArgs...)
	cmd := exec.Command("../bin/koncli", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create once: %v, output: %s", err, string(output))
	}
	t.Logf("Created once: %s", string(output))
}

func waitForOnce(t *testing.T, k8sClient client.Client, name, namespace string) {
	err := wait.PollImmediate(2*time.Second, 10*time.Second, func() (bool, error) {
		once := &syncv1.Once{}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: namespace}, once)
		if err != nil {
			t.Logf("Waiting for once %s: %v", name, err)
			return false, nil
		}
		t.Logf("Once %s found with status: %+v", name, once.Status)
		return true, nil
	})
	if err != nil {
		t.Fatalf("Once was not ready: %v", err)
	}
}

func cleanupOnce(t *testing.T, name, namespace string) {
	cmd := exec.Command("../bin/koncli", "once", "delete", name, "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Failed to delete once: %v, output: %s", err, string(output))
	} else {
		t.Logf("Deleted once: %s", string(output))
	}
}

func TestE2EOnce(t *testing.T) {
	k8sClient, err := setupClient()
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}

	namespace := "default"
	onceName := fmt.Sprintf("e2e-test-once-%d", time.Now().Unix())

	createOnce(t, onceName, namespace)
	waitForOnce(t, k8sClient, onceName, namespace)
	defer cleanupOnce(t, onceName, namespace)

	// Check once status
	cmd := exec.Command("../bin/koncli", "once", "check", onceName, "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to check once: %v, output: %s", err, string(output))
	}

	t.Logf("Once check: %s", string(output))

	// Verify once is not executed
	once := &syncv1.Once{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: onceName, Namespace: namespace}, once)
	if err != nil {
		t.Fatalf("Failed to get once: %v", err)
	}

	if once.Status.Executed {
		t.Errorf("Expected once to not be executed initially")
	}

	if once.Status.Phase != syncv1.OncePhasePending {
		t.Errorf("Expected phase 'Pending', got '%s'", once.Status.Phase)
	}
}

func TestE2EOnceList(t *testing.T) {
	k8sClient, err := setupClient()
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}

	namespace := "default"
	onceName := fmt.Sprintf("e2e-test-once-list-%d", time.Now().Unix())

	createOnce(t, onceName, namespace)
	waitForOnce(t, k8sClient, onceName, namespace)
	defer cleanupOnce(t, onceName, namespace)

	// List onces
	cmd := exec.Command("../bin/koncli", "once", "list", "-n", namespace)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to list onces: %v, output: %s", err, string(output))
	}

	t.Logf("Once list output: %s", string(output))
}

func TestE2EOnceWithTTL(t *testing.T) {
	k8sClient, err := setupClient()
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}

	namespace := "default"
	onceName := fmt.Sprintf("e2e-test-once-ttl-%d", time.Now().Unix())

	createOnce(t, onceName, namespace, "--ttl", "1m")
	waitForOnce(t, k8sClient, onceName, namespace)
	defer cleanupOnce(t, onceName, namespace)

	// Verify TTL is set
	once := &syncv1.Once{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: onceName, Namespace: namespace}, once)
	if err != nil {
		t.Fatalf("Failed to get once: %v", err)
	}

	if once.Spec.TTL == nil {
		t.Error("Expected TTL to be set")
	}
}
