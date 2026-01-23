//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

var (
	loggerOnce sync.Once
	clientOnce sync.Once
	sharedClient client.Client
	setupError error
)

func getKoncliPath() string {
	if path := os.Getenv("KONCLI_PATH"); path != "" {
		return path
	}
	return filepath.Join("..", "bin", "koncli")
}

func getOperatorNamespace() string {
	if ns := os.Getenv("OPERATOR_NAMESPACE"); ns != "" {
		return ns
	}
	return "konductor-system"
}

func getOperatorDeploymentName() string {
	if name := os.Getenv("OPERATOR_DEPLOYMENT_NAME"); name != "" {
		return name
	}
	return "konductor-controller-manager"
}

func setupClient() (client.Client, error) {
	// Initialize logger once to avoid controller-runtime warnings
	loggerOnce.Do(func() {
		log.SetLogger(logr.Discard())
	})

	// Setup client once and reuse
	clientOnce.Do(func() {
		cfg, err := config.GetConfig()
		if err != nil {
			setupError = fmt.Errorf("failed to get config: %w", err)
			return
		}

		scheme := runtime.NewScheme()
		if err := clientgoscheme.AddToScheme(scheme); err != nil {
			setupError = fmt.Errorf("failed to add client-go scheme: %w", err)
			return
		}
		if err := syncv1.AddToScheme(scheme); err != nil {
			setupError = fmt.Errorf("failed to add sync scheme: %w", err)
			return
		}

		sharedClient, err = client.New(cfg, client.Options{Scheme: scheme})
		if err != nil {
			setupError = fmt.Errorf("failed to create client: %w", err)
			return
		}

		// Wait for operator to be ready
		if err := waitForOperator(sharedClient); err != nil {
			setupError = fmt.Errorf("operator not ready: %w", err)
			return
		}
	})

	return sharedClient, setupError
}

func waitForOperator(k8sClient client.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return wait.PollImmediateWithContext(ctx, 2*time.Second, 30*time.Second, func(ctx context.Context) (bool, error) {
		deployment := &appsv1.Deployment{}
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      getOperatorDeploymentName(),
			Namespace: getOperatorNamespace(),
		}, deployment)
		if err != nil {
			return false, nil // Keep waiting
		}

		// Check if deployment is available
		for _, condition := range deployment.Status.Conditions {
			if condition.Type == appsv1.DeploymentAvailable && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	})
}