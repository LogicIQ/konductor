package client

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WaitConfig struct {
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	Factor        float64
	Jitter        float64
	Timeout       time.Duration
	OperatorDelay time.Duration
}

func DefaultWaitConfig() *WaitConfig {
	return &WaitConfig{
		InitialDelay:  500 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		Factor:        1.5,
		Jitter:        0.1,
		Timeout:       30 * time.Second,
		OperatorDelay: 2 * time.Second,
	}
}

func calculateBackoffSteps(initialDelay, maxDelay time.Duration, factor float64, timeout time.Duration) int {
	if initialDelay <= 0 {
		initialDelay = 1 * time.Millisecond
	}
	if factor <= 1.0 {
		factor = 1.1
	}
	elapsed := time.Duration(0)
	current := initialDelay
	steps := 0
	for elapsed < timeout {
		steps++
		elapsed += current
		prev := current
		current = time.Duration(float64(current) * factor)
		if current > maxDelay {
			current = maxDelay
		}
		if current == prev && current == maxDelay {
			remaining := timeout - elapsed
			additionalSteps := remaining / current
			if additionalSteps > 1000000 {
				additionalSteps = 1000000
			}
			steps += int(additionalSteps)
			break
		}
	}
	return steps
}

func (c *Client) WaitForCondition(ctx context.Context, obj client.Object, condition func(client.Object) bool, config *WaitConfig) error {
	if config == nil {
		config = DefaultWaitConfig()
	}

	// Mandatory wait for operator processing
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(config.OperatorDelay):
	}

	// Polling with exponential backoff
	backoff := wait.Backoff{
		Duration: config.InitialDelay,
		Factor:   config.Factor,
		Jitter:   config.Jitter,
		Steps:    calculateBackoffSteps(config.InitialDelay, config.MaxDelay, config.Factor, config.Timeout),
		Cap:      config.MaxDelay,
	}

	return wait.ExponentialBackoff(backoff, func() (bool, error) {
		if err := c.k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return condition(obj), nil
	})
}

func (c *Client) RetryWithBackoff(ctx context.Context, fn func() error, config *WaitConfig) error {
	if config == nil {
		config = DefaultWaitConfig()
	}

	backoff := wait.Backoff{
		Duration: config.InitialDelay,
		Factor:   config.Factor,
		Jitter:   config.Jitter,
		Steps:    calculateBackoffSteps(config.InitialDelay, config.MaxDelay, config.Factor, config.Timeout),
		Cap:      config.MaxDelay,
	}

	return wait.ExponentialBackoff(backoff, func() (bool, error) {
		err := fn()
		if err == nil {
			return true, nil
		}
		if errors.IsConflict(err) {
			return false, nil // Retry conflicts
		}
		return false, err // Don't retry other errors
	})
}
