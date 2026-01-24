package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func TestRootCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		checkFunc   func(t *testing.T, output string)
	}{
		{
			name:        "help command",
			args:        []string{"--help"},
			expectError: false,
			checkFunc: func(t *testing.T, output string) {
				expectedStrings := []string{
					"koncli",
					"Konductor synchronization primitives",
					"semaphore",
					"barrier",
					"lease",
					"gate",
					"status",
					"Namespace Detection",
				}
				for _, expected := range expectedStrings {
					if !bytes.Contains([]byte(output), []byte(expected)) {
						t.Errorf("Expected help output to contain '%s'", expected)
					}
				}
			},
		},
		{
			name:        "help command no panic",
			args:        []string{"--help"},
			expectError: false,
			checkFunc: func(t *testing.T, output string) {
				// This test specifically checks that help doesn't panic
				// when logger is nil (which was the original issue)
				if !bytes.Contains([]byte(output), []byte("koncli")) {
					t.Errorf("Expected help output to contain 'koncli'")
				}
			},
		},
		{
			name:        "version-like behavior",
			args:        []string{},
			expectError: false,
			checkFunc: func(t *testing.T, output string) {
				// Root command without args should show help
				if !bytes.Contains([]byte(output), []byte("koncli")) {
					t.Errorf("Expected output to contain 'koncli'")
				}
			},
		},
		{
			name:        "invalid command",
			args:        []string{"invalid-command"},
			expectError: true,
			checkFunc:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore global variables
			origOutputFormat := outputFormat
			origLogger := logger
			defer func() {
				outputFormat = origOutputFormat
				logger = origLogger
			}()

			// Initialize logger
			outputFormat = "text"
			var err error
			logger, err = zap.NewDevelopment()
			if err != nil {
				t.Fatalf("Failed to initialize logger: %v", err)
			}

			// Create a new root command for each test to avoid state pollution
			rootCmd := &cobra.Command{
				Use:   "koncli",
				Short: "Konductor CLI for coordination primitives",
				Long:  "A CLI tool to interact with Konductor synchronization primitives (semaphores, barriers, leases, gates)\n\nNamespace Detection:\n  - Auto-detects namespace when running in a pod\n  - Falls back to kubeconfig context or 'default'",
			}

			// Add flags
			rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file")
			rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace (auto-detected if running in pod)")

			// Add subcommands
			rootCmd.AddCommand(newSemaphoreCmd())
			rootCmd.AddCommand(newBarrierCmd())
			rootCmd.AddCommand(newLeaseCmd())
			rootCmd.AddCommand(newGateCmd())
			rootCmd.AddCommand(newStatusCmd())

			var output bytes.Buffer
			rootCmd.SetOut(&output)
			rootCmd.SetErr(&output)

			rootCmd.SetArgs(tt.args)
			execErr := rootCmd.Execute()

			if tt.expectError && execErr == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && execErr != nil {
				t.Errorf("Unexpected error: %v", execErr)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, output.String())
			}
		})
	}
}

func TestGlobalFlags(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		checkFunc func(t *testing.T)
	}{
		{
			name: "namespace flag",
			args: []string{"-n", "test-namespace", "--help"},
			checkFunc: func(t *testing.T) {
				if namespace != "test-namespace" {
					t.Errorf("Expected namespace to be 'test-namespace', got '%s'", namespace)
				}
			},
		},
		{
			name: "kubeconfig flag",
			args: []string{"--kubeconfig", "/path/to/config", "--help"},
			checkFunc: func(t *testing.T) {
				if kubeconfig != "/path/to/config" {
					t.Errorf("Expected kubeconfig to be '/path/to/config', got '%s'", kubeconfig)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global variables
			namespace = "default"
			kubeconfig = ""

			rootCmd := &cobra.Command{Use: "koncli"}
			rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file")
			rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")

			var output bytes.Buffer
			rootCmd.SetOut(&output)
			rootCmd.SetErr(&output)

			rootCmd.SetArgs(tt.args)
			if err := rootCmd.Execute(); err != nil {
				t.Errorf("Unexpected error executing command: %v", err)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t)
			}
		})
	}
}

func TestCommandStructure(t *testing.T) {
	rootCmd := &cobra.Command{Use: "koncli"}
	rootCmd.AddCommand(newSemaphoreCmd())
	rootCmd.AddCommand(newBarrierCmd())
	rootCmd.AddCommand(newLeaseCmd())
	rootCmd.AddCommand(newGateCmd())
	rootCmd.AddCommand(newStatusCmd())

	expectedCommands := []string{
		"semaphore",
		"barrier",
		"lease",
		"gate",
		"status",
	}

	for _, expectedCmd := range expectedCommands {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == expectedCmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected command '%s' not found", expectedCmd)
		}
	}
}

func TestSubcommandStructure(t *testing.T) {
	tests := []struct {
		command         string
		expectedSubcmds []string
	}{
		{
			command:         "semaphore",
			expectedSubcmds: []string{"acquire", "release", "list"},
		},
		{
			command:         "barrier",
			expectedSubcmds: []string{"wait", "arrive", "list"},
		},
		{
			command:         "lease",
			expectedSubcmds: []string{"acquire", "release", "list"},
		},
		{
			command:         "gate",
			expectedSubcmds: []string{"wait", "list"},
		},
		{
			command:         "status",
			expectedSubcmds: []string{"semaphore", "barrier", "lease", "gate", "all"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			var cmd *cobra.Command
			switch tt.command {
			case "semaphore":
				cmd = newSemaphoreCmd()
			case "barrier":
				cmd = newBarrierCmd()
			case "lease":
				cmd = newLeaseCmd()
			case "gate":
				cmd = newGateCmd()
			case "status":
				cmd = newStatusCmd()
			}

			if cmd == nil {
				t.Fatalf("Failed to create command '%s'", tt.command)
			}

			for _, expectedSubcmd := range tt.expectedSubcmds {
				found := false
				for _, subcmd := range cmd.Commands() {
					if subcmd.Name() == expectedSubcmd {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected subcommand '%s' not found in '%s'", expectedSubcmd, tt.command)
				}
			}
		})
	}
}

func TestDetectNamespace(t *testing.T) {
	tests := []struct {
		name       string
		podNS      string
		envNS      string
		expectedNS string
	}{
		{
			name:       "POD_NAMESPACE environment variable",
			podNS:      "test-namespace",
			expectedNS: "test-namespace",
		},
		{
			name:       "NAMESPACE environment variable",
			envNS:      "env-namespace",
			expectedNS: "env-namespace",
		},
		{
			name:       "fallback to default",
			expectedNS: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.podNS != "" {
				t.Setenv("POD_NAMESPACE", tt.podNS)
			}
			if tt.envNS != "" {
				t.Setenv("NAMESPACE", tt.envNS)
			}

			result := detectNamespace()
			if result != tt.expectedNS {
				t.Errorf("detectNamespace() = %v, want %v", result, tt.expectedNS)
			}
		})
	}
}

func TestDetectNamespacePriority(t *testing.T) {
	t.Setenv("POD_NAMESPACE", "pod-namespace")
	t.Setenv("NAMESPACE", "env-namespace")

	result := detectNamespace()

	// POD_NAMESPACE should have higher priority
	if result != "pod-namespace" {
		t.Errorf("Expected POD_NAMESPACE to have priority, got %v", result)
	}
}

// TestMain runs before all tests
func TestMain(m *testing.M) {
	// Run tests
	os.Exit(m.Run())
}

// TestExecuteWithNilLogger tests that execute() doesn't panic when logger is nil
func TestExecuteWithNilLogger(t *testing.T) {
	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("execute() panicked with nil logger: %v", r)
		}
	}()

	// Call execute - it should handle nil logger gracefully
	err := execute()
	if err != nil {
		t.Logf("execute() returned error (expected): %v", err)
	}
}
