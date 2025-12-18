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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
	konductor "github.com/LogicIQ/konductor/sdk/go/client"
)

// Permit represents an acquired semaphore permit.
// It provides methods to release the permit and query its properties.
// Permits automatically handle TTL renewal if configured.
type Permit struct {
	client    *konductor.Client
	name      string
	permitID  string
	holder    string
	ctx       context.Context
	cancelCtx context.CancelFunc
}

// Release releases the semaphore permit, making it available for other operations.
// This should always be called when the protected operation is complete,
// typically using defer immediately after acquiring the permit.
//
// Returns an error if the permit cannot be released (e.g., network issues).
func (p *Permit) Release() error {
	if p.cancelCtx != nil {
		p.cancelCtx()
	}

	permit := &syncv1.Permit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.permitID,
			Namespace: p.client.Namespace(),
		},
	}

	return p.client.K8sClient().Delete(context.Background(), permit)
}

// Holder returns the permit holder identifier.
// This is the unique identifier for the entity that acquired this permit.
func (p *Permit) Holder() string {
	return p.holder
}

// Name returns the semaphore name that this permit was acquired from.
func (p *Permit) Name() string {
	return p.name
}

// AcquireSemaphore acquires a permit from the specified semaphore.
// It will wait until a permit becomes available or the context is cancelled.
//
// The function supports various options:
//   - WithTTL: Set automatic permit expiration
//   - WithTimeout: Set maximum wait time
//   - WithHolder: Set custom holder identifier
//
// Returns a Permit that must be released when done, or an error if
// acquisition fails or times out.
//
// Example:
//
//	permit, err := client.AcquireSemaphore(ctx, "database-connections",
//		client.WithTTL(10*time.Minute),
//		client.WithTimeout(30*time.Second))
//	if err != nil {
//		return err
//	}
//	defer permit.Release()
func (c *konductor.Client) AcquireSemaphore(ctx context.Context, name string, opts ...konductor.Option) (*Permit, error) {
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

	permitID := fmt.Sprintf("%s-%s", name, holder)
	startTime := time.Now()

	for {
		// Check if permits are available
		if err := c.K8sClient().Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: c.Namespace(),
		}, &semaphore); err != nil {
			return nil, fmt.Errorf("failed to refresh semaphore %s: %w", name, err)
		}

		if semaphore.Status.Available > 0 {
			// Create permit
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
				return nil, fmt.Errorf("failed to create permit: %w", err)
			}

			// Create permit object with auto-renewal context
			permitCtx, cancelCtx := context.WithCancel(context.Background())
			p := &Permit{
				client:    c,
				name:      name,
				permitID:  permitID,
				holder:    holder,
				ctx:       permitCtx,
				cancelCtx: cancelCtx,
			}

			// Start TTL renewal if specified
			if options.TTL > 0 {
				go p.renewTTL(options.TTL)
			}

			return p, nil
		}

		// Check timeout
		if options.Timeout > 0 && time.Since(startTime) > options.Timeout {
			return nil, fmt.Errorf("timeout waiting for semaphore %s", name)
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			// Continue loop
		}
	}
}

// renewTTL periodically renews the permit TTL
func (p *Permit) renewTTL(ttl time.Duration) {
	ticker := time.NewTicker(ttl / 3) // Renew at 1/3 of TTL
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			// Update permit with new TTL
			var permit syncv1.Permit
			if err := p.client.K8sClient().Get(p.ctx, types.NamespacedName{
				Name:      p.permitID,
				Namespace: p.client.Namespace(),
			}, &permit); err != nil {
				continue // Permit might be deleted
			}

			permit.Spec.TTL = &metav1.Duration{Duration: ttl}
			if err := p.client.K8sClient().Update(p.ctx, &permit); err != nil {
				continue // Continue trying
			}
		}
	}
}

// WithSemaphore executes a function while holding a semaphore permit.
// This is a convenience method that automatically acquires and releases the permit.
// The permit is guaranteed to be released even if the function panics.
//
// Example:
//
//	err := client.WithSemaphore(ctx, "api-rate-limit", func() error {
//		return makeAPICall()
//	}, client.WithTTL(5*time.Minute))
func (c *konductor.Client) WithSemaphore(ctx context.Context, name string, fn func() error, opts ...konductor.Option) error {
	permit, err := c.AcquireSemaphore(ctx, name, opts...)
	if err != nil {
		return err
	}
	defer permit.Release()

	return fn()
}

// ListSemaphores returns all semaphores in the current namespace.
// This is useful for monitoring or administrative operations.
//
// Returns a slice of Semaphore objects or an error if the list operation fails.
func (c *konductor.Client) ListSemaphores(ctx context.Context) ([]syncv1.Semaphore, error) {
	var semaphores syncv1.SemaphoreList
	if err := c.K8sClient().List(ctx, &semaphores, client.InNamespace(c.Namespace())); err != nil {
		return nil, fmt.Errorf("failed to list semaphores: %w", err)
	}
	return semaphores.Items, nil
}

// GetSemaphore returns a specific semaphore by name.
// This allows you to inspect the current state of a semaphore,
// including available permits and current usage.
//
// Returns the Semaphore object or an error if it doesn't exist or cannot be retrieved.
func (c *konductor.Client) GetSemaphore(ctx context.Context, name string) (*syncv1.Semaphore, error) {
	var semaphore syncv1.Semaphore
	if err := c.K8sClient().Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace(),
	}, &semaphore); err != nil {
		return nil, fmt.Errorf("failed to get semaphore %s: %w", name, err)
	}
	return &semaphore, nil
}