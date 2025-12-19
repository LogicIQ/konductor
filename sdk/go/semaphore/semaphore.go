// Package semaphore provides semaphore coordination primitives for the Konductor SDK.
//
// Semaphores are used to limit the number of concurrent operations or resources.
// They maintain a count of available permits and allow operations to acquire
// and release these permits in a distributed manner.
//
// Example usage:
//
//	// Acquire a permit
//	permit, err := client.AcquireSemaphore(ctx, "api-rate-limit",
//		client.WithTTL(5*time.Minute),
//		client.WithTimeout(30*time.Second))
//	if err != nil {
//		return err
//	}
//	defer permit.Release()
//
//	// Perform rate-limited operation
//	return callAPI()
//
// Or use the convenience method:
//
//	err := client.WithSemaphore(ctx, "api-rate-limit", func() error {
//		return callAPI()
//	}, client.WithTTL(5*time.Minute))
package semaphore

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
	konductor "github.com/LogicIQ/konductor/sdk/go/client"
)



// Acquire acquires a permit from the specified semaphore.
func Acquire(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) (*konductor.Permit, error) {
	options := &konductor.Options{
		TTL:     10 * time.Minute, // Default TTL
		Timeout: 0,                // No timeout by default
	}

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

	// Check if semaphore exists
	var semaphore syncv1.Semaphore
	if err := c.K8sClient().Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace(),
	}, &semaphore); err != nil {
		return nil, fmt.Errorf("failed to get semaphore %s: %w", name, err)
	}

	permitID := fmt.Sprintf("%s-%s-%d", name, holder, time.Now().UnixNano())
	startTime := time.Now()

	for {
		// Check semaphore status first
		if err := c.K8sClient().Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: c.Namespace(),
		}, &semaphore); err != nil {
			return nil, fmt.Errorf("failed to get semaphore %s: %w", name, err)
		}

		// If no permits available, wait without creating
		if semaphore.Status.Available <= 0 {
			// Check timeout
			if options.Timeout > 0 && time.Since(startTime) > options.Timeout {
				return nil, fmt.Errorf("timeout waiting for semaphore %s", name)
			}

			// Wait before retrying
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(1 * time.Second):
				continue
			}
		}

		// Create permit (optimistic approach)
		permit := &syncv1.Permit{
			ObjectMeta: metav1.ObjectMeta{
				Name:      permitID,
				Namespace: c.Namespace(),
				Labels: map[string]string{
					"semaphore": name,
				},
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
			// If creation fails due to name conflict, generate new ID and retry
			if strings.Contains(err.Error(), "already exists") {
				permitID = fmt.Sprintf("%s-%s-%d", name, holder, time.Now().UnixNano())
				continue
			}
			return nil, fmt.Errorf("failed to create permit: %w", err)
		}

		// Wait for permit to be granted
		for i := 0; i < 10; i++ {
			var createdPermit syncv1.Permit
			if err := c.K8sClient().Get(ctx, types.NamespacedName{
				Name:      permitID,
				Namespace: c.Namespace(),
			}, &createdPermit); err == nil && createdPermit.Status.Phase == syncv1.PermitPhaseGranted {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		// Successfully created permit
		p := konductor.NewPermit(c, name, holder, ctx)
		return p, nil
	}
}



// With executes a function while holding a semaphore permit.
func With(c *konductor.Client, ctx context.Context, name string, fn func() error, opts ...konductor.Option) error {
	permit, err := Acquire(c, ctx, name, opts...)
	if err != nil {
		return err
	}
	defer permit.Release()

	return fn()
}

// List returns all semaphores in the current namespace.
func List(c *konductor.Client, ctx context.Context) ([]syncv1.Semaphore, error) {
	var semaphores syncv1.SemaphoreList
	if err := c.K8sClient().List(ctx, &semaphores, client.InNamespace(c.Namespace())); err != nil {
		return nil, fmt.Errorf("failed to list semaphores: %w", err)
	}
	return semaphores.Items, nil
}

// Get returns a specific semaphore by name.
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

// Create creates a new semaphore.
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

	return c.K8sClient().Create(ctx, semaphore)
}

// Delete deletes a semaphore.
func Delete(c *konductor.Client, ctx context.Context, name string) error {
	semaphore := &syncv1.Semaphore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
	}
	return c.K8sClient().Delete(ctx, semaphore)
}

// Update updates a semaphore.
func Update(c *konductor.Client, ctx context.Context, semaphore *syncv1.Semaphore) error {
	return c.K8sClient().Update(ctx, semaphore)
}

