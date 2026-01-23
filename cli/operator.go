package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
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

func isValidHealthURL(rawURL string) bool {
	// Only allow cluster-local service URLs for health checks
	// Also allow localhost/127.0.0.1 for testing
	if len(rawURL) == 0 {
		return false
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// Only allow http/https schemes
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}

	// Extract hostname (without port)
	hostname := parsed.Hostname()
	if hostname == "" {
		return false
	}

	// Check for allowed hosts using strict suffix/exact matching
	isAllowed := false
	if strings.HasSuffix(hostname, ".svc.cluster.local") {
		isAllowed = true
	} else if hostname == "127.0.0.1" || hostname == "localhost" || strings.HasPrefix(hostname, "127.") {
		isAllowed = true
	}

	if !isAllowed {
		return false
	}

	// Allow test URLs from localhost/127.* without specific endpoints
	if hostname == "127.0.0.1" || hostname == "localhost" || strings.HasPrefix(hostname, "127.") {
		return true
	}

	// Check for allowed endpoints (must be exact path prefix)
	allowedEndpoints := []string{"/healthz", "/readyz"}
	for _, endpoint := range allowedEndpoints {
		if parsed.Path == endpoint || strings.HasPrefix(parsed.Path, endpoint+"/") {
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

// isDisallowedIP checks if the IP should be blocked to prevent SSRF attacks.
// This blocks private IPs (RFC 1918), multicast, and unspecified addresses.
// Loopback (127.0.0.1) is allowed for testing purposes.
func isDisallowedIP(hostIP string) bool {
	ip := net.ParseIP(hostIP)
	if ip == nil {
		return true // Block if we can't parse the IP
	}
	// Allow loopback for testing purposes
	if ip.IsLoopback() {
		return false
	}
	// Block multicast, unspecified, and private IPs
	return ip.IsMulticast() || ip.IsUnspecified() || ip.IsPrivate()
}

// safeTransport creates an HTTP transport that validates resolved IP addresses
// to prevent SSRF attacks via DNS rebinding
func safeTransport(timeout time.Duration) *http.Transport {
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: timeout}
			c, err := dialer.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			ip, _, err := net.SplitHostPort(c.RemoteAddr().String())
			if err != nil {
				_ = c.Close()
				return nil, errors.New("failed to parse remote address")
			}
			if isDisallowedIP(ip) {
				_ = c.Close()
				return nil, errors.New("connection to disallowed IP address blocked")
			}
			return c, nil
		},
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: timeout}
			// #nosec G402 - InsecureSkipVerify is false by default, MinVersion enforces TLS 1.2+
			tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
			c, err := tls.DialWithDialer(dialer, network, addr, tlsConfig)
			if err != nil {
				return nil, err
			}
			ip, _, err := net.SplitHostPort(c.RemoteAddr().String())
			if err != nil {
				_ = c.Close()
				return nil, errors.New("failed to parse remote address")
			}
			if isDisallowedIP(ip) {
				_ = c.Close()
				return nil, errors.New("connection to disallowed IP address blocked")
			}
			return c, nil
		},
		TLSHandshakeTimeout: timeout,
	}
}

func getRestrictedTransport(cfg *rest.Config) http.RoundTripper {
	// Use safe transport with IP validation for SSRF protection
	baseTransport := safeTransport(2 * time.Second)

	// If we have a rest config with custom transport settings, wrap it
	if cfg != nil && cfg.Transport != nil {
		return &restrictedTransport{base: cfg.Transport}
	}
	return &restrictedTransport{base: baseTransport}
}

type restrictedTransport struct {
	base http.RoundTripper
}

func (rt *restrictedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Additional SSRF protection at transport level
	host := req.URL.Host
	if !isAllowedHost(host) {
		return nil, errors.New("blocked request to disallowed host")
	}
	// Only allow GET requests
	if req.Method != http.MethodGet {
		return nil, fmt.Errorf("blocked non-GET request: %s", req.Method)
	}
	return rt.base.RoundTrip(req)
}

func isAllowedHost(host string) bool {
	// Only allow specific hosts for health checks
	if len(host) == 0 {
		return false
	}

	// Parse host to separate hostname from port
	hostname, _, err := net.SplitHostPort(host)
	if err != nil {
		// No port specified, use host as-is
		hostname = host
	}

	// Strict validation - must end with cluster-local or be localhost/127.0.0.1
	if strings.HasSuffix(hostname, ".svc.cluster.local") {
		// Additional check: ensure it's actually a cluster service (no path traversal)
		return !strings.Contains(hostname, "..")
	}
	// Allow localhost and 127.0.0.1 for testing
	if hostname == "127.0.0.1" || hostname == "localhost" || strings.HasPrefix(hostname, "127.") {
		return true
	}
	return false
}

func sanitizeURL(rawURL string) string {
	// Remove sensitive information from URL for logging
	if len(rawURL) == 0 {
		return "<empty>"
	}
	// Only show the path component for security
	if strings.Contains(rawURL, "/healthz") {
		return "<service>/healthz"
	}
	if strings.Contains(rawURL, "/readyz") {
		return "<service>/readyz"
	}
	return "<sanitized>"
}
