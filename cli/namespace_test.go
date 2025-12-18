package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

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
		name           string
		setupFunc      func(t *testing.T) func()
		expectedNS     string
		expectedOutput string
	}{
		{
			name: "service account namespace file",
			setupFunc: func(t *testing.T) func() {
				// Create temporary service account namespace file
				tmpDir := t.TempDir()
				saDir := filepath.Join(tmpDir, "var", "run", "secrets", "kubernetes.io", "serviceaccount")
				err := os.MkdirAll(saDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create SA dir: %v", err)
				}

				nsFile := filepath.Join(saDir, "namespace")
				err = ioutil.WriteFile(nsFile, []byte("production"), 0644)
				if err != nil {
					t.Fatalf("Failed to write namespace file: %v", err)
				}

				return func() {
					// cleanup
				}
			},
			expectedNS: "production",
		},
		{
			name: "POD_NAMESPACE environment variable",
			setupFunc: func(t *testing.T) func() {
				os.Setenv("POD_NAMESPACE", "test-namespace")
				os.Unsetenv("NAMESPACE")
				return func() {}
			},
			expectedNS: "test-namespace",
		},
		{
			name: "NAMESPACE environment variable",
			setupFunc: func(t *testing.T) func() {
				os.Unsetenv("POD_NAMESPACE")
				os.Setenv("NAMESPACE", "env-namespace")
				return func() {}
			},
			expectedNS: "env-namespace",
		},
		{
			name: "kubeconfig context namespace",
			setupFunc: func(t *testing.T) func() {
				os.Unsetenv("POD_NAMESPACE")
				os.Unsetenv("NAMESPACE")

				// Create temporary kubeconfig
				tmpDir := t.TempDir()
				kubeconfigPath := filepath.Join(tmpDir, "config")

				config := &clientcmdapi.Config{
					CurrentContext: "test-context",
					Contexts: map[string]*clientcmdapi.Context{
						"test-context": {
							Namespace: "kubeconfig-namespace",
						},
					},
				}

				err := clientcmd.WriteToFile(*config, kubeconfigPath)
				if err != nil {
					t.Fatalf("Failed to write kubeconfig: %v", err)
				}

				kubeconfig = kubeconfigPath
				return func() {}
			},
			expectedNS: "kubeconfig-namespace",
		},
		{
			name: "fallback to default",
			setupFunc: func(t *testing.T) func() {
				os.Unsetenv("POD_NAMESPACE")
				os.Unsetenv("NAMESPACE")
				kubeconfig = "/nonexistent/path"
				return func() {}
			},
			expectedNS: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setupFunc(t)
			defer cleanup()

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

func TestDetectNamespaceEmptyValues(t *testing.T) {
	// Save original values
	origPodNS := os.Getenv("POD_NAMESPACE")
	origNS := os.Getenv("NAMESPACE")

	defer func() {
		os.Setenv("POD_NAMESPACE", origPodNS)
		os.Setenv("NAMESPACE", origNS)
	}()

	// Set empty values
	os.Setenv("POD_NAMESPACE", "")
	os.Setenv("NAMESPACE", "valid-namespace")

	result := detectNamespace()

	// Should skip empty POD_NAMESPACE and use NAMESPACE
	if result != "valid-namespace" {
		t.Errorf("Expected to skip empty POD_NAMESPACE, got %v", result)
	}
}