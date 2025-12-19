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

// Wait waits for a barrier to open
func Wait(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) error {
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

// Arrive signals arrival at a barrier
func Arrive(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) error {
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

// With executes a function and then signals arrival at a barrier
func With(c *konductor.Client, ctx context.Context, name string, fn func() error, opts ...konductor.Option) error {
	if err := fn(); err != nil {
		return err
	}
	return Arrive(c, ctx, name, opts...)
}

// WaitAndArrive waits for a barrier to open, executes a function, then signals arrival at another barrier
func WaitAndArrive(c *konductor.Client, ctx context.Context, waitBarrier, arriveBarrier string, fn func() error, opts ...konductor.Option) error {
	// Wait for the first barrier
	if err := Wait(c, ctx, waitBarrier, opts...); err != nil {
		return fmt.Errorf("failed to wait for barrier %s: %w", waitBarrier, err)
	}

	// Execute function
	if err := fn(); err != nil {
		return err
	}

	// Signal arrival at second barrier
	if err := Arrive(c, ctx, arriveBarrier, opts...); err != nil {
		return fmt.Errorf("failed to arrive at barrier %s: %w", arriveBarrier, err)
	}

	return nil
}

// List returns all barriers in the namespace
func List(c *konductor.Client, ctx context.Context) ([]syncv1.Barrier, error) {
	var barriers syncv1.BarrierList
	if err := c.K8sClient().List(ctx, &barriers, client.InNamespace(c.Namespace())); err != nil {
		return nil, fmt.Errorf("failed to list barriers: %w", err)
	}
	return barriers.Items, nil
}

// Get returns a specific barrier
func Get(c *konductor.Client, ctx context.Context, name string) (*syncv1.Barrier, error) {
	var barrier syncv1.Barrier
	if err := c.K8sClient().Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace(),
	}, &barrier); err != nil {
		return nil, fmt.Errorf("failed to get barrier %s: %w", name, err)
	}
	return &barrier, nil
}

// GetStatus returns the current status of a barrier
func GetStatus(c *konductor.Client, ctx context.Context, name string) (*syncv1.BarrierStatus, error) {
	barrier, err := Get(c, ctx, name)
	if err != nil {
		return nil, err
	}
	return &barrier.Status, nil
}

// Create creates a new barrier.
func Create(c *konductor.Client, ctx context.Context, name string, expected int32, opts ...konductor.Option) error {
	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
		Spec: syncv1.BarrierSpec{
			Expected: expected,
		},
	}
	return c.K8sClient().Create(ctx, barrier)
}

// Delete deletes a barrier.
func Delete(c *konductor.Client, ctx context.Context, name string) error {
	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
	}
	return c.K8sClient().Delete(ctx, barrier)
}

// Update updates a barrier.
func Update(c *konductor.Client, ctx context.Context, barrier *syncv1.Barrier) error {
	return c.K8sClient().Update(ctx, barrier)
}

// CreateBarrier creates a new barrier with the specified expected arrivals
func CreateBarrier(c *konductor.Client, ctx context.Context, name string, expected int32, opts ...konductor.Option) error {
	options := &konductor.Options{}
	for _, opt := range opts {
		opt(options)
	}

	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
		Spec: syncv1.BarrierSpec{
			Expected: expected,
		},
	}

	if options.Timeout > 0 {
		barrier.Spec.Timeout = &metav1.Duration{Duration: options.Timeout}
	}

	if options.Quorum > 0 {
		barrier.Spec.Quorum = &options.Quorum
	}

	if err := c.K8sClient().Create(ctx, barrier); err != nil {
		return fmt.Errorf("failed to create barrier %s: %w", name, err)
	}

	return nil
}

// DeleteBarrier deletes a barrier and all its associated arrivals
func DeleteBarrier(c *konductor.Client, ctx context.Context, name string) error {
	barrier := &syncv1.Barrier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
	}

	if err := c.K8sClient().Delete(ctx, barrier); err != nil {
		return fmt.Errorf("failed to delete barrier %s: %w", name, err)
	}

	return nil
}
