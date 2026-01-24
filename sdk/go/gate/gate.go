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

func Wait(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) error {
	options := &konductor.Options{Timeout: 0}
	for _, opt := range opts {
		opt(options)
	}

	gate := &syncv1.Gate{}
	gate.Name = name
	gate.Namespace = c.Namespace()

	config := &konductor.WaitConfig{
		InitialDelay: 2 * time.Second,
		MaxDelay:     10 * time.Second,
		Factor:       1.5,
		Jitter:       0.1,
		Timeout:      30 * time.Second,
	}

	if options.Timeout > 0 {
		config.Timeout = options.Timeout
	}

	err := c.WaitForCondition(ctx, gate, func(obj client.Object) bool {
		g := obj.(*syncv1.Gate)
		switch g.Status.Phase {
		case syncv1.GatePhaseOpen:
			return true
		case syncv1.GatePhaseFailed:
			return true // Will be handled as error after condition returns
		default:
			return false
		}
	}, config)

	if err != nil {
		return err
	}

	// Check final state after wait completes
	var finalGate syncv1.Gate
	if err := c.K8sClient().Get(ctx, types.NamespacedName{
		Name: name, Namespace: c.Namespace(),
	}, &finalGate); err != nil {
		return fmt.Errorf("failed to get gate %s: %w", name, err)
	}

	if finalGate.Status.Phase == syncv1.GatePhaseFailed {
		return fmt.Errorf("gate %s failed", name)
	}

	return nil
}

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

func WaitForConditions(c *konductor.Client, ctx context.Context, name string, conditionNames []string, opts ...konductor.Option) error {
	if len(conditionNames) == 0 {
		return nil
	}

	options := &konductor.Options{
		Timeout: 0,
	}

	for _, opt := range opts {
		opt(options)
	}

	startTime := time.Now()
	delay := 1 * time.Second
	maxDelay := 10 * time.Second

	for {
		conditions, err := GetConditions(c, ctx, name)
		if err != nil {
			return fmt.Errorf("failed to get conditions for gate %s: %w", name, err)
		}

		// Create map for O(1) lookup
		conditionMap := make(map[string]bool)
		for _, condition := range conditions {
			conditionMap[condition.Name] = condition.Met
		}

		allMet := true
		for _, condName := range conditionNames {
			if met, found := conditionMap[condName]; !found || !met {
				allMet = false
				break
			}
		}

		if allMet {
			return nil
		}

		if options.Timeout > 0 && time.Since(startTime) > options.Timeout {
			return fmt.Errorf("timeout waiting for conditions in gate %s", name)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			delay = min(time.Duration(float64(delay)*1.5), maxDelay)
		}
	}
}

func With(c *konductor.Client, ctx context.Context, name string, fn func() error, opts ...konductor.Option) error {
	if err := Wait(c, ctx, name, opts...); err != nil {
		return err
	}
	return fn()
}

func List(c *konductor.Client, ctx context.Context) ([]syncv1.Gate, error) {
	var gates syncv1.GateList
	if err := c.K8sClient().List(ctx, &gates, client.InNamespace(c.Namespace())); err != nil {
		return nil, fmt.Errorf("failed to list gates: %w", err)
	}
	return gates.Items, nil
}

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

func GetStatus(c *konductor.Client, ctx context.Context, name string) (*syncv1.GateStatus, error) {
	gate, err := Get(c, ctx, name)
	if err != nil {
		return nil, err
	}
	return &gate.Status, nil
}

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

func Delete(c *konductor.Client, ctx context.Context, name string) error {
	gate := &syncv1.Gate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
	}
	return c.K8sClient().Delete(ctx, gate)
}

func Update(c *konductor.Client, ctx context.Context, gate *syncv1.Gate) error {
	return c.K8sClient().Update(ctx, gate)
}

func Open(c *konductor.Client, ctx context.Context, name string) error {
	gate := &syncv1.Gate{}
	gate.Name = name
	gate.Namespace = c.Namespace()

	err := c.RetryWithBackoff(ctx, func() error {
		var g syncv1.Gate
		if err := c.K8sClient().Get(ctx, types.NamespacedName{
			Name: name, Namespace: c.Namespace(),
		}, &g); err != nil {
			return err
		}
		g.Status.Phase = syncv1.GatePhaseOpen
		return c.K8sClient().Status().Update(ctx, &g)
	}, nil)

	if err != nil {
		return err
	}

	// Wait for confirmation
	return c.WaitForCondition(ctx, gate, func(obj client.Object) bool {
		g := obj.(*syncv1.Gate)
		return g.Status.Phase == syncv1.GatePhaseOpen
	}, nil)
}

func Close(c *konductor.Client, ctx context.Context, name string) error {
	gate := &syncv1.Gate{}
	gate.Name = name
	gate.Namespace = c.Namespace()

	err := c.RetryWithBackoff(ctx, func() error {
		var g syncv1.Gate
		if err := c.K8sClient().Get(ctx, types.NamespacedName{
			Name: name, Namespace: c.Namespace(),
		}, &g); err != nil {
			return err
		}
		g.Status.Phase = syncv1.GatePhaseWaiting
		return c.K8sClient().Status().Update(ctx, &g)
	}, nil)

	if err != nil {
		return err
	}

	// Wait for confirmation
	return c.WaitForCondition(ctx, gate, func(obj client.Object) bool {
		g := obj.(*syncv1.Gate)
		return g.Status.Phase == syncv1.GatePhaseWaiting
	}, nil)
}
