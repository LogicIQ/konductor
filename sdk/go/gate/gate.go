package gate

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
	konductor "github.com/LogicIQ/konductor/sdk/go/client"
)

// WaitGate waits for a gate to open (all conditions met)
func (c *konductor.Client) WaitGate(ctx context.Context, name string, opts ...konductor.Option) error {
	options := &konductor.Options{
		Timeout: 0, // No timeout by default
	}

	for _, opt := range opts {
		opt(options)
	}

	startTime := time.Now()

	for {
		var gate syncv1.Gate
		if err := c.K8sClient().Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: c.Namespace(),
		}, &gate); err != nil {
			return fmt.Errorf("failed to get gate %s: %w", name, err)
		}

		switch gate.Status.Phase {
		case syncv1.GatePhaseOpen:
			return nil
		case syncv1.GatePhaseFailed:
			return fmt.Errorf("gate %s failed", name)
		case syncv1.GatePhaseWaiting:
			// Check timeout
			if options.Timeout > 0 && time.Since(startTime) > options.Timeout {
				return fmt.Errorf("timeout waiting for gate %s", name)
			}

			// Wait before retrying
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Second):
				// Continue loop
			}
		}
	}
}

// CheckGate checks if a gate is open without waiting
func (c *konductor.Client) CheckGate(ctx context.Context, name string) (bool, error) {
	var gate syncv1.Gate
	if err := c.K8sClient().Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace(),
	}, &gate); err != nil {
		return false, fmt.Errorf("failed to get gate %s: %w", name, err)
	}

	return gate.Status.Phase == syncv1.GatePhaseOpen, nil
}

// GetGateConditions returns the current status of all gate conditions
func (c *konductor.Client) GetGateConditions(ctx context.Context, name string) ([]syncv1.GateConditionStatus, error) {
	var gate syncv1.Gate
	if err := c.K8sClient().Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace(),
	}, &gate); err != nil {
		return nil, fmt.Errorf("failed to get gate %s: %w", name, err)
	}

	return gate.Status.ConditionStatuses, nil
}

// WaitForConditions waits for specific conditions to be met
func (c *konductor.Client) WaitForConditions(ctx context.Context, name string, conditionNames []string, opts ...konductor.Option) error {
	options := &konductor.Options{
		Timeout: 0, // No timeout by default
	}

	for _, opt := range opts {
		opt(options)
	}

	startTime := time.Now()

	for {
		conditions, err := c.GetGateConditions(ctx, name)
		if err != nil {
			return err
		}

		// Check if all specified conditions are met
		allMet := true
		for _, condName := range conditionNames {
			found := false
			for _, condition := range conditions {
				if condition.Name == condName {
					found = true
					if !condition.Met {
						allMet = false
						break
					}
				}
			}
			if !found {
				allMet = false
				break
			}
		}

		if allMet {
			return nil
		}

		// Check timeout
		if options.Timeout > 0 && time.Since(startTime) > options.Timeout {
			return fmt.Errorf("timeout waiting for conditions in gate %s", name)
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
			// Continue loop
		}
	}
}

// WithGate executes a function after waiting for a gate to open
func (c *konductor.Client) WithGate(ctx context.Context, name string, fn func() error, opts ...konductor.Option) error {
	if err := c.WaitGate(ctx, name, opts...); err != nil {
		return err
	}
	return fn()
}

// ListGates returns all gates in the namespace
func (c *konductor.Client) ListGates(ctx context.Context) ([]syncv1.Gate, error) {
	var gates syncv1.GateList
	if err := c.K8sClient().List(ctx, &gates, client.InNamespace(c.Namespace())); err != nil {
		return nil, fmt.Errorf("failed to list gates: %w", err)
	}
	return gates.Items, nil
}

// GetGate returns a specific gate
func (c *konductor.Client) GetGate(ctx context.Context, name string) (*syncv1.Gate, error) {
	var gate syncv1.Gate
	if err := c.K8sClient().Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace(),
	}, &gate); err != nil {
		return nil, fmt.Errorf("failed to get gate %s: %w", name, err)
	}
	return &gate, nil
}

// GetGateStatus returns the current status of a gate
func (c *konductor.Client) GetGateStatus(ctx context.Context, name string) (*syncv1.GateStatus, error) {
	gate, err := c.GetGate(ctx, name)
	if err != nil {
		return nil, err
	}
	return &gate.Status, nil
}