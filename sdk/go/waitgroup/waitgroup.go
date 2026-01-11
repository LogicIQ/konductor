package waitgroup

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
	konductor "github.com/LogicIQ/konductor/sdk/go/client"
)

// Add increments the counter by delta with atomic operation protection
func Add(c *konductor.Client, ctx context.Context, name string, delta int32) error {
	var originalCounter int32
	
	// Retry on conflicts with atomic read-modify-write
	err := c.RetryWithBackoff(ctx, func() error {
		var wg syncv1.WaitGroup
		if err := c.K8sClient().Get(ctx, types.NamespacedName{
			Name: name, Namespace: c.Namespace(),
		}, &wg); err != nil {
			return err
		}
		
		// Store original for confirmation
		originalCounter = wg.Status.Counter
		
		// Atomic increment - this will fail with conflict if another pod modified it
		wg.Status.Counter += delta
		if wg.Status.Counter < 0 {
			wg.Status.Counter = 0
		}
		
		if wg.Status.Counter <= 0 {
			wg.Status.Phase = syncv1.WaitGroupPhaseDone
		} else {
			wg.Status.Phase = syncv1.WaitGroupPhaseWaiting
		}
		
		// This update will fail with 409 Conflict if resource version changed
		return c.K8sClient().Status().Update(ctx, &wg)
	}, nil)
	
	if err != nil {
		return fmt.Errorf("failed to update waitgroup: %w", err)
	}
	
	// Wait for confirmation of change
	wg := &syncv1.WaitGroup{}
	wg.Name = name
	wg.Namespace = c.Namespace()
	
	if err := c.WaitForCondition(ctx, wg, func(obj interface{}) bool {
		waitGroup := obj.(*syncv1.WaitGroup)
		return waitGroup.Status.Counter != originalCounter
	}, nil); err != nil {
		return fmt.Errorf("failed to confirm waitgroup update: %w", err)
	}
	
	return nil
}

// Done decrements the counter by 1
func Done(c *konductor.Client, ctx context.Context, name string) error {
	return Add(c, ctx, name, -1)
}

// Wait blocks until counter is zero with exponential backoff
func Wait(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) error {
	options := &konductor.Options{Timeout: 0}
	for _, opt := range opts {
		opt(options)
	}
	
	wg := &syncv1.WaitGroup{}
	wg.Name = name
	wg.Namespace = c.Namespace()
	
	config := &konductor.WaitConfig{
		InitialDelay: 500 * time.Millisecond,
		MaxDelay: 5 * time.Second,
		Factor: 1.5,
		Jitter: 0.1,
		Timeout: 30 * time.Second,
	}
	
	if options.Timeout > 0 {
		config.Timeout = options.Timeout
	}
	
	if err := c.WaitForCondition(ctx, wg, func(obj interface{}) bool {
		waitGroup := obj.(*syncv1.WaitGroup)
		return waitGroup.Status.Counter <= 0
	}, config); err != nil {
		return fmt.Errorf("failed to wait for waitgroup %s: %w", name, err)
	}
	
	return nil
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

	// Use retry for create operations to handle name conflicts
	return c.RetryWithBackoff(ctx, func() error {
		err := c.K8sClient().Create(ctx, wg)
		if err != nil && errors.IsAlreadyExists(err) {
			// Resource already exists, this is not an error for idempotent create
			return nil
		}
		return err
	}, nil)
}

func Delete(c *konductor.Client, ctx context.Context, name string) error {
	wg := &syncv1.WaitGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
	}
	err := c.K8sClient().Delete(ctx, wg)
	if err != nil && errors.IsNotFound(err) {
		// Resource doesn't exist, this is not an error for idempotent delete
		return nil
	}
	return err
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
