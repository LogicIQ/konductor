//go:build e2e

package e2e

import (
	"os/exec"
	"strings"
	"testing"
)

func TestE2EOperatorStatus(t *testing.T) {
	if _, err := setupClient(); err != nil {
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
	expectedFields := []string{"Operator Service:", "Namespace:", "Health:", "Ready:"}
	for _, field := range expectedFields {
		if !strings.Contains(outputStr, field) {
			t.Errorf("Expected output to contain '%s'", field)
		}
	}
}
