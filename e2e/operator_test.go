//go:build e2e

package e2e

import (
	"os/exec"
	"strings"
	"testing"
)

func TestE2EOperatorStatus(t *testing.T) {
	_, err := setupClient()
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}

	// Check operator status using CLI
	cmd := exec.Command(getKoncliPath(), "operator", "--operator-namespace", "konductor-system")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to check operator status: %v, output: %s", err, string(output))
	}

	outputStr := string(output)
	t.Logf("Operator status: %s", outputStr)

	// Verify output contains expected fields
	if !strings.Contains(outputStr, "Operator Service:") {
		t.Error("Expected output to contain 'Operator Service:'")
	}
	if !strings.Contains(outputStr, "Namespace:") {
		t.Error("Expected output to contain 'Namespace:'")
	}
	if !strings.Contains(outputStr, "Health:") {
		t.Error("Expected output to contain 'Health:'")
	}
	if !strings.Contains(outputStr, "Ready:") {
		t.Error("Expected output to contain 'Ready:'")
	}
}
