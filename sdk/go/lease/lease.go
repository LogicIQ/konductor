package lease

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

// Lease represents an acquired lease
type Lease struct {
	client    *konductor.Client
	name      string
	requestID string
	holder    string
	ctx       context.Context
	cancelCtx context.CancelFunc
}

// Release releases the lease
func (l *Lease) Release() error {
	if l.cancelCtx != nil {
		l.cancelCtx()
	}

	request := &syncv1.LeaseRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      l.requestID,
			Namespace: l.client.Namespace(),
		},
	}

	return l.client.K8sClient().Delete(context.Background(), request)
}

// Holder returns the lease holder identifier
func (l *Lease) Holder() string {
	return l.holder
}

// Name returns the lease name
func (l *Lease) Name() string {
	return l.name
}

// Acquire acquires a lease
func Acquire(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) (*Lease, error) {
	options := &konductor.Options{
		Timeout:  0, // No timeout by default
		Priority: 0, // Default priority
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

	requestID := fmt.Sprintf("%s-%s", name, holder)

	// Create lease request
	request := &syncv1.LeaseRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      requestID,
			Namespace: c.Namespace(),
			Labels: map[string]string{
				"lease": name,
			},
		},
		Spec: syncv1.LeaseRequestSpec{
			Lease:  name,
			Holder: holder,
		},
	}

	if options.Priority > 0 {
		request.Spec.Priority = &options.Priority
	}

	if err := c.K8sClient().Create(ctx, request); err != nil {
		return nil, fmt.Errorf("failed to create lease request: %w", err)
	}

	startTime := time.Now()

	for {
		// Check request status
		if err := c.K8sClient().Get(ctx, types.NamespacedName{
			Name:      requestID,
			Namespace: c.Namespace(),
		}, request); err != nil {
			return nil, fmt.Errorf("failed to get lease request: %w", err)
		}

		switch request.Status.Phase {
		case syncv1.LeaseRequestPhaseGranted:
			// Create lease object
			leaseCtx, cancelCtx := context.WithCancel(context.Background())
			return &Lease{
				client:    c,
				name:      name,
				requestID: requestID,
				holder:    holder,
				ctx:       leaseCtx,
				cancelCtx: cancelCtx,
			}, nil

		case syncv1.LeaseRequestPhaseDenied:
			return nil, fmt.Errorf("lease request denied for %s", name)

		case syncv1.LeaseRequestPhasePending:
			// Check timeout
			if options.Timeout > 0 && time.Since(startTime) > options.Timeout {
				// Clean up request
				c.K8sClient().Delete(ctx, request)
				return nil, fmt.Errorf("timeout waiting for lease %s", name)
			}

			// Wait before retrying
			select {
			case <-ctx.Done():
				// Clean up request
				c.K8sClient().Delete(context.Background(), request)
				return nil, ctx.Err()
			case <-time.After(5 * time.Second):
				// Continue loop
			}
		}
	}
}

// With executes a function while holding a lease
func With(c *konductor.Client, ctx context.Context, name string, fn func() error, opts ...konductor.Option) error {
	lease, err := Acquire(c, ctx, name, opts...)
	if err != nil {
		return err
	}
	defer lease.Release()

	return fn()
}

// TryAcquire attempts to acquire a lease without waiting
func TryAcquire(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) (*Lease, error) {
	// Add zero timeout to make it non-blocking
	opts = append(opts, konductor.WithTimeout(1*time.Second))
	return Acquire(c, ctx, name, opts...)
}

// List returns all leases in the namespace
func List(c *konductor.Client, ctx context.Context) ([]syncv1.Lease, error) {
	var leases syncv1.LeaseList
	if err := c.K8sClient().List(ctx, &leases, client.InNamespace(c.Namespace())); err != nil {
		return nil, fmt.Errorf("failed to list leases: %w", err)
	}
	return leases.Items, nil
}

// Get returns a specific lease
func Get(c *konductor.Client, ctx context.Context, name string) (*syncv1.Lease, error) {
	var lease syncv1.Lease
	if err := c.K8sClient().Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace(),
	}, &lease); err != nil {
		return nil, fmt.Errorf("failed to get lease %s: %w", name, err)
	}
	return &lease, nil
}

// IsAvailable checks if a lease is available for acquisition
func IsAvailable(c *konductor.Client, ctx context.Context, name string) (bool, error) {
	lease, err := Get(c, ctx, name)
	if err != nil {
		return false, err
	}
	return lease.Status.Phase == syncv1.LeasePhaseAvailable, nil
}

// Create creates a new lease.
func Create(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) error {
	options := &konductor.Options{TTL: 10 * time.Minute}
	for _, opt := range opts {
		opt(options)
	}

	lease := &syncv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
		Spec: syncv1.LeaseSpec{
			TTL: metav1.Duration{Duration: options.TTL},
		},
	}
	return c.K8sClient().Create(ctx, lease)
}

// Delete deletes a lease.
func Delete(c *konductor.Client, ctx context.Context, name string) error {
	lease := &syncv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
	}
	return c.K8sClient().Delete(ctx, lease)
}

// Update updates a lease.
func Update(c *konductor.Client, ctx context.Context, lease *syncv1.Lease) error {
	return c.K8sClient().Update(ctx, lease)
}

// CreateLease creates a new lease
func CreateLease(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) error {
	options := &konductor.Options{}
	for _, opt := range opts {
		opt(options)
	}

	lease := &syncv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
		Spec: syncv1.LeaseSpec{},
	}

	if options.TTL > 0 {
		lease.Spec.TTL = metav1.Duration{Duration: options.TTL}
	}

	if err := c.K8sClient().Create(ctx, lease); err != nil {
		return fmt.Errorf("failed to create lease %s: %w", name, err)
	}

	return nil
}

// DeleteLease deletes a lease and all its associated requests
func DeleteLease(c *konductor.Client, ctx context.Context, name string) error {
	lease := &syncv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
	}

	if err := c.K8sClient().Delete(ctx, lease); err != nil {
		return fmt.Errorf("failed to delete lease %s: %w", name, err)
	}

	return nil
}
