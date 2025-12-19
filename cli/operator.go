package main

import (
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
	healthURL := "http://" + svcName + "." + operatorNamespace + ".svc.cluster.local:8081"

	version, health := checkHealthWithVersion(healthURL + "/healthz")
	ready := checkHealth(healthURL + "/readyz")

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
	cfg, _ := rest.InClusterConfig()
	client := &http.Client{
		Timeout:   2 * time.Second,
		Transport: getTransport(cfg),
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", "unavailable"
	}
	defer resp.Body.Close()

	version := resp.Header.Get("X-Konductor-Version")
	if resp.StatusCode == http.StatusOK {
		return version, "OK"
	}
	return version, resp.Status
}

func getTransport(cfg *rest.Config) http.RoundTripper {
	if cfg != nil && cfg.Transport != nil {
		return cfg.Transport
	}
	return http.DefaultTransport
}
