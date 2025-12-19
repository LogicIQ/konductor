package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
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
	fmt.Printf("Operator Service: %s\n", svcName)
	fmt.Printf("Namespace: %s\n", operatorNamespace)

	baseURL := fmt.Sprintf("http://%s.%s.svc.cluster.local:8081", svcName, operatorNamespace)

	if version := callEndpoint(baseURL + "/version"); version != "" {
		fmt.Printf("Version: %s\n", version)
	}

	fmt.Printf("Health: %s\n", checkHealth(baseURL+"/healthz"))
	fmt.Printf("Ready: %s\n", checkHealth(baseURL+"/readyz"))

	return nil
}

func callEndpoint(url string) string {
	cfg, _ := rest.InClusterConfig()
	client := &http.Client{
		Timeout:   2 * time.Second,
		Transport: getTransport(cfg),
	}

	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]string
	if json.Unmarshal(body, &result) == nil {
		return result["version"]
	}
	return ""
}

func checkHealth(url string) string {
	cfg, _ := rest.InClusterConfig()
	client := &http.Client{
		Timeout:   2 * time.Second,
		Transport: getTransport(cfg),
	}

	resp, err := client.Get(url)
	if err != nil {
		return "unavailable"
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return "OK"
	}
	return resp.Status
}

func getTransport(cfg *rest.Config) http.RoundTripper {
	if cfg != nil && cfg.Transport != nil {
		return cfg.Transport
	}
	return http.DefaultTransport
}
