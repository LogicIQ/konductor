package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"
)

func newOperatorCmd() *cobra.Command {
	var svcName string
	var operatorNamespace string

	cmd := &cobra.Command{
		Use:   "operator",
		Short: "Check operator status",
		Long:  "Check the health, readiness, and version of the Konductor operator",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOperatorStatus(cmd, args, svcName, operatorNamespace)
		},
	}

	cmd.Flags().StringVar(&svcName, "service", "konductor-controller-manager", "Operator service name")
	cmd.Flags().StringVar(&operatorNamespace, "operator-namespace", "konductor-system", "Operator namespace")

	return cmd
}

func runOperatorStatus(cmd *cobra.Command, args []string, svcName, operatorNamespace string) error {
	// Validate input parameters to prevent SSRF
	if !isValidServiceName(svcName) || !isValidNamespace(operatorNamespace) {
		return fmt.Errorf("invalid service name or namespace")
	}
	
	healthURL := "http://" + svcName + "." + operatorNamespace + ".svc.cluster.local:8081"

	version, health := checkHealthWithVersion(healthURL + "/healthz")
	ready := checkHealth(healthURL + "/readyz")

	// Print formatted output for CLI users
	fmt.Printf("Operator Service: %s\n", svcName)
	fmt.Printf("Namespace: %s\n", operatorNamespace)
	fmt.Printf("Version: %s\n", version)
	fmt.Printf("Health: %s\n", health)
	fmt.Printf("Ready: %s\n", ready)

	// Also log for structured logging
	logger.Info("Operator status",
		zap.String("service", svcName),
		zap.String("namespace", operatorNamespace),
		zap.String("version", version),
		zap.String("health", health),
		zap.String("ready", ready),
	)

	return nil
}

func checkHealth(url string) string {
	_, status := checkHealthWithVersion(url)
	return status
}

func checkHealthWithVersion(url string) (string, string) {
	// Validate URL to prevent SSRF
	if !isValidHealthURL(url) {
		return "", "invalid URL"
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to default transport if not in cluster
		cfg = nil
	}
	client := &http.Client{
		Timeout:   2 * time.Second,
		Transport: getRestrictedTransport(cfg),
	}

	resp, err := client.Get(url)
	if err != nil {
		// Check if error is due to blocked host
		if contains(err.Error(), "blocked request") {
			return "", "blocked"
		}
		// Log the error for debugging
		logger.Debug("Health check failed", zap.String("url", sanitizeURL(url)), zap.Error(err))
		return "", "unavailable"
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.Debug("Failed to close response body", zap.Error(closeErr))
		}
	}()

	version := resp.Header.Get("X-Konductor-Version")
	if resp.StatusCode == http.StatusOK {
		return version, "OK"
	}
	if resp.Status == "" {
		return version, fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return version, resp.Status
}

func isValidHealthURL(url string) bool {
	// Only allow cluster-local service URLs for health checks
	// Also allow localhost/127.0.0.1 for testing
	if len(url) == 0 {
		return false
	}
	
	// Check protocol
	if len(url) < 7 {
		return false
	}
	if url[:7] != "http://" {
		if len(url) < 8 || url[:8] != "https://" {
			return false
		}
	}
	
	// Check for allowed hosts
	allowedHosts := []string{".svc.cluster.local:", "127.0.0.1:", "localhost:"}
	hostFound := false
	for _, host := range allowedHosts {
		if contains(url, host) {
			hostFound = true
			break
		}
	}
	if !hostFound {
		return false
	}
	
	// Check for allowed endpoints
	allowedEndpoints := []string{"/healthz", "/readyz"}
	for _, endpoint := range allowedEndpoints {
		if contains(url, endpoint) {
			return true
		}
	}
	
	// Allow test URLs without specific endpoints
	return isTestURL(url)
}

func isTestURL(url string) bool {
	// Allow test URLs that don't have health endpoints
	// Validate that it's actually a test URL to prevent abuse
	if len(url) == 0 {
		return false
	}
	return contains(url, "127.0.0.1:") || contains(url, "localhost:")
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func isValidServiceName(name string) bool {
	// Kubernetes service name validation
	if len(name) == 0 || len(name) > 63 {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return false
		}
	}
	return name[0] != '-' && name[len(name)-1] != '-'
}

func isValidNamespace(namespace string) bool {
	// Kubernetes namespace validation
	if len(namespace) == 0 || len(namespace) > 63 {
		return false
	}
	for _, r := range namespace {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return false
		}
	}
	return namespace[0] != '-' && namespace[len(namespace)-1] != '-'
}

func getRestrictedTransport(cfg *rest.Config) http.RoundTripper {
	baseTransport := getTransport(cfg)
	return &restrictedTransport{base: baseTransport}
}

type restrictedTransport struct {
	base http.RoundTripper
}

func (rt *restrictedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Additional SSRF protection at transport level
	host := req.URL.Host
	if !isAllowedHost(host) {
		return nil, fmt.Errorf("blocked request to disallowed host")
	}
	// Only allow GET requests
	if req.Method != http.MethodGet {
		return nil, fmt.Errorf("blocked non-GET request: %s", req.Method)
	}
	return rt.base.RoundTrip(req)
}

func isAllowedHost(host string) bool {
	// Only allow specific hosts for health checks
	allowedPatterns := []string{
		".svc.cluster.local:",
		"127.0.0.1:",
		"localhost:",
	}
	for _, pattern := range allowedPatterns {
		if contains(host, pattern) {
			return true
		}
	}
	return false
}

func sanitizeURL(url string) string {
	// Remove sensitive information from URL for logging
	if len(url) == 0 {
		return "<empty>"
	}
	// Only show the path component for security
	if contains(url, "/healthz") {
		return "<service>/healthz"
	}
	if contains(url, "/readyz") {
		return "<service>/readyz"
	}
	return "<sanitized>"
}

func getTransport(cfg *rest.Config) http.RoundTripper {
	if cfg != nil && cfg.Transport != nil {
		return cfg.Transport
	}
	return http.DefaultTransport
}
