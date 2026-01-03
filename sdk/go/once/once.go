package once

import (
	"context"
	"fmt"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
	konductor "github.com/LogicIQ/konductor/sdk/go/client"
)

// Do executes the function if it hasn't been executed yet
// Returns true if this call executed the function, false if already executed
func Do(c *konductor.Client, ctx context.Context, name string, fn func() error, opts ...konductor.Option) (bool, error) {
	options := &konductor.Options{}
	for _, opt := range opts {
		opt(options)
	}

	executor := options.Holder
	if executor == "" {
		executor = os.Getenv("HOSTNAME")
		if executor == "" {
			executor = fmt.Sprintf("sdk-%d", time.Now().Unix())
		}
	}

	var once syncv1.Once
	if err := c.K8sClient().Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace(),
	}, &once); err != nil {
		return false, fmt.Errorf("failed to get once: %w", err)
	}

	// Check if already executed
	if once.Status.Executed {
		return false, nil
	}

	// Try to mark as executed
	once.Status.Executed = true
	once.Status.Executor = executor
	executedAt := metav1.Now()
	once.Status.ExecutedAt = &executedAt
	once.Status.Phase = syncv1.OncePhaseExecuted

	if err := c.K8sClient().Status().Update(ctx, &once); err != nil {
		// Someone else got it first
		return false, nil
	}

	// Execute the function
	if err := fn(); err != nil {
		return true, fmt.Errorf("execution failed: %w", err)
	}

	return true, nil
}

// IsExecuted checks if the once has been executed
func IsExecuted(c *konductor.Client, ctx context.Context, name string) (bool, error) {
	once, err := Get(c, ctx, name)
	if err != nil {
		return false, err
	}
	return once.Status.Executed, nil
}

// Create creates a new once
func Create(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) error {
	options := &konductor.Options{}
	for _, opt := range opts {
		opt(options)
	}

	once := &syncv1.Once{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
		Spec: syncv1.OnceSpec{},
	}

	if options.TTL > 0 {
		once.Spec.TTL = &metav1.Duration{Duration: options.TTL}
	}

	return c.K8sClient().Create(ctx, once)
}

func Delete(c *konductor.Client, ctx context.Context, name string) error {
	once := &syncv1.Once{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
	}
	return c.K8sClient().Delete(ctx, once)
}

func Get(c *konductor.Client, ctx context.Context, name string) (*syncv1.Once, error) {
	var once syncv1.Once
	if err := c.K8sClient().Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace(),
	}, &once); err != nil {
		return nil, fmt.Errorf("failed to get once %s: %w", name, err)
	}
	return &once, nil
}

func List(c *konductor.Client, ctx context.Context) ([]syncv1.Once, error) {
	var onces syncv1.OnceList
	if err := c.K8sClient().List(ctx, &onces, client.InNamespace(c.Namespace())); err != nil {
		return nil, fmt.Errorf("failed to list onces: %w", err)
	}
	return onces.Items, nil
}
