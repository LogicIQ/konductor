package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
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
					"Konductor CLI for coordination primitives",
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
			err := rootCmd.Execute()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
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
			rootCmd.Execute() // Ignore error for flag testing

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
		command          string
		expectedSubcmds  []string
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
	// Save original values
	origPodNS := os.Getenv("POD_NAMESPACE")
	origNS := os.Getenv("NAMESPACE")
	origKubeconfig := kubeconfig

	// Clean up after test
	defer func() {
		os.Setenv("POD_NAMESPACE", origPodNS)
		os.Setenv("NAMESPACE", origNS)
		kubeconfig = origKubeconfig
	}()

	tests := []struct {
		name       string
		setupFunc  func(t *testing.T)
		expectedNS string
	}{
		{
			name: "POD_NAMESPACE environment variable",
			setupFunc: func(t *testing.T) {
				os.Setenv("POD_NAMESPACE", "test-namespace")
				os.Unsetenv("NAMESPACE")
			},
			expectedNS: "test-namespace",
		},
		{
			name: "NAMESPACE environment variable",
			setupFunc: func(t *testing.T) {
				os.Unsetenv("POD_NAMESPACE")
				os.Setenv("NAMESPACE", "env-namespace")
			},
			expectedNS: "env-namespace",
		},
		{
			name: "fallback to default",
			setupFunc: func(t *testing.T) {
				os.Unsetenv("POD_NAMESPACE")
				os.Unsetenv("NAMESPACE")
				kubeconfig = "/nonexistent/path"
			},
			expectedNS: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc(t)
			result := detectNamespace()
			if result != tt.expectedNS {
				t.Errorf("detectNamespace() = %v, want %v", result, tt.expectedNS)
			}
		})
	}
}

func TestDetectNamespacePriority(t *testing.T) {
	// Save original values
	origPodNS := os.Getenv("POD_NAMESPACE")
	origNS := os.Getenv("NAMESPACE")

	defer func() {
		os.Setenv("POD_NAMESPACE", origPodNS)
		os.Setenv("NAMESPACE", origNS)
	}()

	// Set both environment variables
	os.Setenv("POD_NAMESPACE", "pod-namespace")
	os.Setenv("NAMESPACE", "env-namespace")

	result := detectNamespace()

	// POD_NAMESPACE should have higher priority
	if result != "pod-namespace" {
		t.Errorf("Expected POD_NAMESPACE to have priority, got %v", result)
	}
}

// Test environment variable cleanup
func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	
	// Cleanup
	os.Unsetenv("POD_NAMESPACE")
	os.Unsetenv("NAMESPACE")
	
	os.Exit(code)
}