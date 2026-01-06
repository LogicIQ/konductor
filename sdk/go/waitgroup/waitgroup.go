package waitgroup

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

// Add increments the counter by delta
func Add(c *konductor.Client, ctx context.Context, name string, delta int32) error {
	var wg syncv1.WaitGroup
	if err := c.K8sClient().Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace(),
	}, &wg); err != nil {
		return fmt.Errorf("failed to get waitgroup: %w", err)
	}

	wg.Status.Counter += delta
	if wg.Status.Counter < 0 {
		wg.Status.Counter = 0
	}

	if wg.Status.Counter <= 0 {
		wg.Status.Phase = syncv1.WaitGroupPhaseDone
	} else {
		wg.Status.Phase = syncv1.WaitGroupPhaseWaiting
	}

	if err := c.K8sClient().Status().Update(ctx, &wg); err != nil {
		return fmt.Errorf("failed to update waitgroup: %w", err)
	}

	return nil
}

// Done decrements the counter by 1
func Done(c *konductor.Client, ctx context.Context, name string) error {
	return Add(c, ctx, name, -1)
}

// Wait blocks until the counter is zero
func Wait(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) error {
	options := &konductor.Options{Timeout: 0}
	for _, opt := range opts {
		opt(options)
	}

	startTime := time.Now()

	for {
		var wg syncv1.WaitGroup
		if err := c.K8sClient().Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: c.Namespace(),
		}, &wg); err != nil {
			return fmt.Errorf("failed to get waitgroup: %w", err)
		}

		if wg.Status.Counter <= 0 {
			return nil
		}

		if options.Timeout > 0 && time.Since(startTime) > options.Timeout {
			return fmt.Errorf("timeout waiting for waitgroup %s", name)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}

// GetCounter returns the current counter value
func GetCounter(c *konductor.Client, ctx context.Context, name string) (int32, error) {
	wg, err := Get(c, ctx, name)
	if err != nil {
		return 0, err
	}
	return wg.Status.Counter, nil
}

func Create(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) error {
	options := &konductor.Options{}
	for _, opt := range opts {
		opt(options)
	}

	wg := &syncv1.WaitGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
		Spec: syncv1.WaitGroupSpec{},
	}

	if options.TTL > 0 {
		wg.Spec.TTL = &metav1.Duration{Duration: options.TTL}
	}

	return c.K8sClient().Create(ctx, wg)
}

func Delete(c *konductor.Client, ctx context.Context, name string) error {
	wg := &syncv1.WaitGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
	}
	return c.K8sClient().Delete(ctx, wg)
}

func Get(c *konductor.Client, ctx context.Context, name string) (*syncv1.WaitGroup, error) {
	var wg syncv1.WaitGroup
	if err := c.K8sClient().Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace(),
	}, &wg); err != nil {
		return nil, fmt.Errorf("failed to get waitgroup %s: %w", name, err)
	}
	return &wg, nil
}

func List(c *konductor.Client, ctx context.Context) ([]syncv1.WaitGroup, error) {
	var wgs syncv1.WaitGroupList
	if err := c.K8sClient().List(ctx, &wgs, client.InNamespace(c.Namespace())); err != nil {
		return nil, fmt.Errorf("failed to list waitgroups: %w", err)
	}
	return wgs.Items, nil
}
