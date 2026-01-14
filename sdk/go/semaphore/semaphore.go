package semaphore

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

func Acquire(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) (*konductor.Permit, error) {
	options := &konductor.Options{TTL: 10 * time.Minute, Timeout: 0}
	for _, opt := range opts {
		opt(options)
	}

	holder := options.Holder
	if holder == "" {
		holder = os.Getenv("HOSTNAME")
		if holder == "" {
			holder = fmt.Sprintf("sdk-%d", time.Now().Unix())
		}
	}

	var semaphore syncv1.Semaphore
	if err := c.K8sClient().Get(ctx, types.NamespacedName{
		Name: name, Namespace: c.Namespace(),
	}, &semaphore); err != nil {
		return nil, fmt.Errorf("failed to get semaphore %s: %w", name, err)
	}

	// Check if permits are available (for production)
	if semaphore.Status.Available <= 0 && options.Timeout > 0 {
		config := &konductor.WaitConfig{
			InitialDelay: 1 * time.Second,
			MaxDelay:     5 * time.Second,
			Factor:       1.5,
			Jitter:       0.1,
			Timeout:      options.Timeout,
		}

		// Wait for available permits
		err := c.WaitForCondition(ctx, &semaphore, func(obj interface{}) bool {
			s := obj.(*syncv1.Semaphore)
			return s.Status.Available > 0
		}, config)

		if err != nil {
			return nil, fmt.Errorf("timeout waiting for semaphore %s: %w", name, err)
		}
	}

	permitID := fmt.Sprintf("%s-%s-%d", name, holder, time.Now().UnixNano())

	// Create permit
	ctrlTrue := true
	permit := &syncv1.Permit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      permitID,
			Namespace: c.Namespace(),
			Labels:    map[string]string{"semaphore": name},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "sync.konductor.io/v1",
				Kind:               "Semaphore",
				Name:               semaphore.Name,
				UID:                semaphore.UID,
				Controller:         &ctrlTrue,
				BlockOwnerDeletion: &ctrlTrue,
			}},
		},
		Spec: syncv1.PermitSpec{
			Semaphore: name,
			Holder:    holder,
		},
	}

	if options.TTL > 0 {
		permit.Spec.TTL = &metav1.Duration{Duration: options.TTL}
	}

	if err := c.K8sClient().Create(ctx, permit); err != nil {
		return nil, fmt.Errorf("failed to create permit: %w", err)
	}

	// Only wait for permit grant confirmation if timeout is specified (production)
	if options.Timeout > 0 {
		config := &konductor.WaitConfig{
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Timeout:      5 * time.Second,
		}

		err := c.WaitForCondition(ctx, permit, func(obj interface{}) bool {
			p := obj.(*syncv1.Permit)
			return p.Status.Phase == syncv1.PermitPhaseGranted
		}, config)

		if err != nil {
			if deleteErr := c.K8sClient().Delete(ctx, permit); deleteErr != nil {
				return nil, fmt.Errorf("failed to wait for permit grant and failed to cleanup permit: %w (cleanup error: %v)", err, deleteErr)
			}
			return nil, err
		}
	}

	return konductor.NewPermit(c, name, holder, ctx), nil
}

func With(c *konductor.Client, ctx context.Context, name string, fn func() error, opts ...konductor.Option) error {
	permit, err := Acquire(c, ctx, name, opts...)
	if err != nil {
		return err
	}
	defer permit.Release(ctx)

	return fn()
}

func List(c *konductor.Client, ctx context.Context) ([]syncv1.Semaphore, error) {
	var semaphores syncv1.SemaphoreList
	if err := c.K8sClient().List(ctx, &semaphores, client.InNamespace(c.Namespace())); err != nil {
		return nil, fmt.Errorf("failed to list semaphores: %w", err)
	}
	return semaphores.Items, nil
}

func Get(c *konductor.Client, ctx context.Context, name string) (*syncv1.Semaphore, error) {
	var semaphore syncv1.Semaphore
	if err := c.K8sClient().Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace(),
	}, &semaphore); err != nil {
		return nil, fmt.Errorf("failed to get semaphore %s: %w", name, err)
	}
	return &semaphore, nil
}

func Create(c *konductor.Client, ctx context.Context, name string, permits int32, opts ...konductor.Option) error {
	options := &konductor.Options{}
	for _, opt := range opts {
		opt(options)
	}

	semaphore := &syncv1.Semaphore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
		Spec: syncv1.SemaphoreSpec{
			Permits: permits,
		},
	}

	if options.TTL > 0 {
		semaphore.Spec.TTL = &metav1.Duration{Duration: options.TTL}
	}

	if err := c.K8sClient().Create(ctx, semaphore); err != nil {
		return fmt.Errorf("failed to create semaphore %s: %w", name, err)
	}
	return nil
}

func Delete(c *konductor.Client, ctx context.Context, name string) error {
	semaphore := &syncv1.Semaphore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
	}
	err := c.K8sClient().Delete(ctx, semaphore)
	if err != nil {
		return fmt.Errorf("failed to delete semaphore %s: %w", name, err)
	}
	return nil
}

func Update(c *konductor.Client, ctx context.Context, semaphore *syncv1.Semaphore) error {
	if err := c.K8sClient().Update(ctx, semaphore); err != nil {
		return fmt.Errorf("failed to update semaphore %s: %w", semaphore.Name, err)
	}
	return nil
}
