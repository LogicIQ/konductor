package client

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RetryConfig defines retry behavior for operator updates
type RetryConfig struct {
	// InitialDelay before first retry
	InitialDelay time.Duration
	// MaxDelay between retries
	MaxDelay time.Duration
	// Factor for exponential backoff
	Factor float64
	// MaxRetries before giving up
	MaxRetries int
	// OperatorDelay to account for controller processing
	OperatorDelay time.Duration
}

// DefaultRetryConfig provides sensible defaults for Kubernetes operations
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		Factor:        2.0,
		MaxRetries:    10,
		OperatorDelay: 2 * time.Second, // Wait for operator to process
	}
}

// RetryOnConflict retries operations that may have resource version conflicts
func (c *Client) RetryOnConflict(ctx context.Context, fn func() error) error {
	config := DefaultRetryConfig()
	
	backoff := wait.Backoff{
		Duration: config.InitialDelay,
		Factor:   config.Factor,
		Jitter:   0.1,
		Steps:    config.MaxRetries,
		Cap:      config.MaxDelay,
	}

	return wait.ExponentialBackoff(backoff, func() (bool, error) {
		err := fn()
		if err == nil {
			return true, nil
		}

		// Retry on conflict errors
		if errors.IsConflict(err) {
			return false, nil
		}

		// Don't retry other errors
		return false, err
	})
}

// WaitForUpdate waits for operator to process changes before continuing
func (c *Client) WaitForUpdate(ctx context.Context, obj client.Object, checkFn func(client.Object) bool) error {
	config := DefaultRetryConfig()
	
	// First wait for operator processing time
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(config.OperatorDelay):
	}

	// Then poll for the expected state
	backoff := wait.Backoff{
		Duration: 500 * time.Millisecond,
		Factor:   1.5,
		Jitter:   0.1,
		Steps:    20, // Up to ~30 seconds total
		Cap:      2 * time.Second,
	}

	return wait.ExponentialBackoff(backoff, func() (bool, error) {
		if err := c.k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
			if errors.IsNotFound(err) {
				return false, nil // Keep waiting
			}
			return false, err
		}

		return checkFn(obj), nil
	})
}

// UpdateWithRetry performs optimistic locking updates with retry
func (c *Client) UpdateWithRetry(ctx context.Context, obj client.Object, updateFn func(client.Object) error) error {
	return c.RetryOnConflict(ctx, func() error {
		// Get latest version
		if err := c.k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
			return err
		}

		// Apply changes
		if err := updateFn(obj); err != nil {
			return err
		}

		// Update with latest resource version
		return c.k8sClient.Update(ctx, obj)
	})
}

// StatusUpdateWithRetry performs status updates with retry and conflict handling
func (c *Client) StatusUpdateWithRetry(ctx context.Context, obj client.Object, updateFn func(client.Object) error) error {
	return c.RetryOnConflict(ctx, func() error {
		// Get latest version
		if err := c.k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
			return err
		}

		// Apply status changes
		if err := updateFn(obj); err != nil {
			return err
		}

		// Update status with latest resource version
		return c.k8sClient.Status().Update(ctx, obj)
	})
}