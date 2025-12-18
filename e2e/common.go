//go:build e2e

package e2e

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

func setupClient() (client.Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	scheme, err := syncv1.SchemeBuilder.Build()
	if err != nil {
		return nil, err
	}

	return client.New(cfg, client.Options{Scheme: scheme})
}