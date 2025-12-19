package gate

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
	konductor "github.com/LogicIQ/konductor/sdk/go/client"
)

// Wait waits for a gate to open (all conditions met)
func Wait(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) error {
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

// Check checks if a gate is open without waiting
func Check(c *konductor.Client, ctx context.Context, name string) (bool, error) {
	var gate syncv1.Gate
	if err := c.K8sClient().Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace(),
	}, &gate); err != nil {
		return false, fmt.Errorf("failed to get gate %s: %w", name, err)
	}

	return gate.Status.Phase == syncv1.GatePhaseOpen, nil
}

// GetConditions returns the current status of all gate conditions
func GetConditions(c *konductor.Client, ctx context.Context, name string) ([]syncv1.GateConditionStatus, error) {
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
func WaitForConditions(c *konductor.Client, ctx context.Context, name string, conditionNames []string, opts ...konductor.Option) error {
	options := &konductor.Options{
		Timeout: 0, // No timeout by default
	}

	for _, opt := range opts {
		opt(options)
	}

	startTime := time.Now()

	for {
		conditions, err := GetConditions(c, ctx, name)
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

// With executes a function after waiting for a gate to open
func With(c *konductor.Client, ctx context.Context, name string, fn func() error, opts ...konductor.Option) error {
	if err := Wait(c, ctx, name, opts...); err != nil {
		return err
	}
	return fn()
}

// List returns all gates in the namespace
func List(c *konductor.Client, ctx context.Context) ([]syncv1.Gate, error) {
	var gates syncv1.GateList
	if err := c.K8sClient().List(ctx, &gates, client.InNamespace(c.Namespace())); err != nil {
		return nil, fmt.Errorf("failed to list gates: %w", err)
	}
	return gates.Items, nil
}

// Get returns a specific gate
func Get(c *konductor.Client, ctx context.Context, name string) (*syncv1.Gate, error) {
	var gate syncv1.Gate
	if err := c.K8sClient().Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace(),
	}, &gate); err != nil {
		return nil, fmt.Errorf("failed to get gate %s: %w", name, err)
	}
	return &gate, nil
}

// GetStatus returns the current status of a gate
func GetStatus(c *konductor.Client, ctx context.Context, name string) (*syncv1.GateStatus, error) {
	gate, err := Get(c, ctx, name)
	if err != nil {
		return nil, err
	}
	return &gate.Status, nil
}

// Create creates a new gate.
func Create(c *konductor.Client, ctx context.Context, name string) error {
	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
		Spec: syncv1.GateSpec{
			Conditions: []syncv1.GateCondition{},
		},
	}
	return c.K8sClient().Create(ctx, gate)
}

// Delete deletes a gate.
func Delete(c *konductor.Client, ctx context.Context, name string) error {
	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
	}
	return c.K8sClient().Delete(ctx, gate)
}

// Update updates a gate.
func Update(c *konductor.Client, ctx context.Context, gate *syncv1.Gate) error {
	return c.K8sClient().Update(ctx, gate)
}

// Open manually opens a gate.
func Open(c *konductor.Client, ctx context.Context, name string) error {
	gate, err := Get(c, ctx, name)
	if err != nil {
		return err
	}
	gate.Status.Phase = syncv1.GatePhaseOpen
	return c.K8sClient().Status().Update(ctx, gate)
}

// Close manually closes a gate.
func Close(c *konductor.Client, ctx context.Context, name string) error {
	gate, err := Get(c, ctx, name)
	if err != nil {
		return err
	}
	gate.Status.Phase = syncv1.GatePhaseWaiting
	return c.K8sClient().Status().Update(ctx, gate)
}
