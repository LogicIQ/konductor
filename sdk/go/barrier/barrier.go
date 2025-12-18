package barrier

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

// WaitBarrier waits for a barrier to open
func WaitBarrier(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) error {
	options := &konductor.Options{
		Timeout: 0, // No timeout by default
	}

	for _, opt := range opts {
		opt(options)
	}

	startTime := time.Now()

	for {
		var barrier syncv1.Barrier
		if err := c.K8sClient().Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: c.Namespace(),
		}, &barrier); err != nil {
			return fmt.Errorf("failed to get barrier %s: %w", name, err)
		}

		switch barrier.Status.Phase {
		case syncv1.BarrierPhaseOpen:
			return nil
		case syncv1.BarrierPhaseFailed:
			return fmt.Errorf("barrier %s failed", name)
		case syncv1.BarrierPhaseWaiting:
			// Check timeout
			if options.Timeout > 0 && time.Since(startTime) > options.Timeout {
				return fmt.Errorf("timeout waiting for barrier %s", name)
			}

			// Wait before retrying
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
				// Continue loop
			}
		}
	}
}

// ArriveBarrier signals arrival at a barrier
func ArriveBarrier(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) error {
	options := &konductor.Options{}

	for _, opt := range opts {
		opt(options)
	}

	// Get holder identifier
	holder := options.Holder
	if holder == "" {
		holder = os.Getenv("HOSTNAME")
		if holder == "" {
			holder = fmt.Sprintf("sdk-%d", time.Now().Unix())
		}
	}

	// Create arrival
	arrival := &syncv1.Arrival{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", name, holder),
			Namespace: c.Namespace(),
			Labels: map[string]string{
				"barrier": name,
			},
		},
		Spec: syncv1.ArrivalSpec{
			Barrier: name,
			Holder:  holder,
		},
	}

	if err := c.K8sClient().Create(ctx, arrival); err != nil {
		return fmt.Errorf("failed to create arrival: %w", err)
	}

	return nil
}

// WithBarrier executes a function and then signals arrival at a barrier
func WithBarrier(c *konductor.Client, ctx context.Context, name string, fn func() error, opts ...konductor.Option) error {
	if err := fn(); err != nil {
		return err
	}
	return ArriveBarrier(c, ctx, name, opts...)
}

// WaitAndArrive waits for a barrier to open, executes a function, then signals arrival at another barrier
func WaitAndArrive(c *konductor.Client, ctx context.Context, waitBarrier, arriveBarrier string, fn func() error, opts ...konductor.Option) error {
	// Wait for the first barrier
	if err := WaitBarrier(c, ctx, waitBarrier, opts...); err != nil {
		return fmt.Errorf("failed to wait for barrier %s: %w", waitBarrier, err)
	}

	// Execute function
	if err := fn(); err != nil {
		return err
	}

	// Signal arrival at second barrier
	if err := ArriveBarrier(c, ctx, arriveBarrier, opts...); err != nil {
		return fmt.Errorf("failed to arrive at barrier %s: %w", arriveBarrier, err)
	}

	return nil
}

// ListBarriers returns all barriers in the namespace
func ListBarriers(c *konductor.Client, ctx context.Context) ([]syncv1.Barrier, error) {
	var barriers syncv1.BarrierList
	if err := c.K8sClient().List(ctx, &barriers, client.InNamespace(c.Namespace())); err != nil {
		return nil, fmt.Errorf("failed to list barriers: %w", err)
	}
	return barriers.Items, nil
}

// GetBarrier returns a specific barrier
func GetBarrier(c *konductor.Client, ctx context.Context, name string) (*syncv1.Barrier, error) {
	var barrier syncv1.Barrier
	if err := c.K8sClient().Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace(),
	}, &barrier); err != nil {
		return nil, fmt.Errorf("failed to get barrier %s: %w", name, err)
	}
	return &barrier, nil
}

// GetBarrierStatus returns the current status of a barrier
func GetBarrierStatus(c *konductor.Client, ctx context.Context, name string) (*syncv1.BarrierStatus, error) {
	barrier, err := GetBarrier(c, ctx, name)
	if err != nil {
		return nil, err
	}
	return &barrier.Status, nil
}