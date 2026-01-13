// Package client provides the core Konductor SDK client for interacting with coordination primitives.
//
// The client package offers a high-level interface for working with semaphores, barriers,
// leases, and gates in Kubernetes environments. It handles Kubernetes client setup,
// namespace management, and provides convenient methods for coordination operations.
//
// Example usage:
//
//	client, err := client.New(&client.Config{Namespace: "my-app"})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Use semaphores, barriers, leases, gates...
//	permit, err := client.AcquireSemaphore(ctx, "api-limit")
package client

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

// Client provides access to konductor coordination primitives.
// It wraps a Kubernetes client and provides high-level methods for
// working with semaphores, barriers, leases, and gates.
type Client struct {
	k8sClient client.Client
	namespace string
}

// Config holds client configuration options.
// All fields are optional and will use sensible defaults if not specified.
type Config struct {
	// Namespace specifies the Kubernetes namespace to operate in.
	// Defaults to "default" if not specified.
	Namespace string
	// Kubeconfig path for authentication (currently unused, uses in-cluster or default config)
	Kubeconfig string
}

// New creates a new konductor client with the specified configuration.
// It automatically sets up the Kubernetes client and scheme registration.
//
// Returns an error if the Kubernetes client cannot be created or if
// the konductor types cannot be registered with the scheme.
func New(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = &Config{}
	}

	// Get Kubernetes config
	k8sConfig, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	// Build scheme with konductor types
	scheme := runtime.NewScheme()
	if err := syncv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add konductor types to scheme: %w", err)
	}

	// Create Kubernetes client
	k8sClient, err := client.New(k8sConfig, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	namespace := cfg.Namespace
	if namespace == "" {
		namespace = "default"
	}

	return &Client{
		k8sClient: k8sClient,
		namespace: namespace,
	}, nil
}

// NewFromClient creates a konductor client from an existing Kubernetes client.
// This is useful when you already have a configured Kubernetes client and want
// to reuse it for konductor operations.
//
// The namespace parameter specifies which Kubernetes namespace to operate in.
// If empty, defaults to "default".
func NewFromClient(k8sClient client.Client, namespace string) *Client {
	if namespace == "" {
		namespace = "default"
	}
	return &Client{
		k8sClient: k8sClient,
		namespace: namespace,
	}
}

// Namespace returns the current namespace that this client operates in.
func (c *Client) Namespace() string {
	return c.namespace
}

// WithNamespace returns a new client instance configured to operate in the specified namespace.
// The original client is not modified. This allows for easy multi-namespace operations.
//
// Example:
//
//	defaultClient := client.New(&client.Config{})
//	testClient := defaultClient.WithNamespace("test")
func (c *Client) WithNamespace(namespace string) *Client {
	return &Client{
		k8sClient: c.k8sClient,
		namespace: namespace,
	}
}

// K8sClient returns the underlying Kubernetes client.
// This can be used for advanced operations not covered by the konductor SDK.
func (c *Client) K8sClient() client.Client {
	return c.k8sClient
}

// ReleaseSemaphorePermit releases a semaphore permit.
func (c *Client) ReleaseSemaphorePermit(ctx context.Context, semaphoreName, holder string) error {
	permitName := fmt.Sprintf("%s-%s", semaphoreName, holder)
	permit := &syncv1.Permit{}
	permit.Name = permitName
	permit.Namespace = c.namespace
	if err := c.k8sClient.Delete(ctx, permit); err != nil {
		return fmt.Errorf("failed to delete permit %s: %w", permitName, err)
	}
	return nil
}

// ReleaseLease releases a lease.
func (c *Client) ReleaseLease(ctx context.Context, leaseName, holder string) error {
	requestName := fmt.Sprintf("%s-%s", leaseName, holder)
	request := &syncv1.LeaseRequest{}
	request.Name = requestName
	request.Namespace = c.namespace
	if err := c.k8sClient.Delete(ctx, request); err != nil {
		return fmt.Errorf("failed to delete lease request %s: %w", requestName, err)
	}
	return nil
}

// ListPermits returns all permits for a specific semaphore.
func (c *Client) ListPermits(ctx context.Context, semaphoreName string) ([]syncv1.Permit, error) {
	var permits syncv1.PermitList
	if err := c.k8sClient.List(ctx, &permits, client.InNamespace(c.namespace),
		client.MatchingLabels{"semaphore": semaphoreName}); err != nil {
		return nil, fmt.Errorf("failed to list permits: %w", err)
	}
	return permits.Items, nil
}

// ListLeaseRequests returns all lease requests for a specific lease.
func (c *Client) ListLeaseRequests(ctx context.Context, leaseName string) ([]syncv1.LeaseRequest, error) {
	var requests syncv1.LeaseRequestList
	if err := c.k8sClient.List(ctx, &requests, client.InNamespace(c.namespace),
		client.MatchingLabels{"lease": leaseName}); err != nil {
		return nil, fmt.Errorf("failed to list lease requests: %w", err)
	}
	return requests.Items, nil
}

// Options contains common configuration options for coordination operations.
// Use the With* functions to create options in a fluent style.
type Options struct {
	// TTL specifies how long a resource should live before automatic cleanup
	TTL time.Duration
	// Timeout specifies how long to wait for an operation to complete
	Timeout time.Duration
	// Priority is used for lease acquisition ordering (higher values win)
	Priority int32
	// Holder identifies the entity holding a resource (defaults to hostname)
	Holder string
	// Quorum specifies minimum arrivals needed to open a barrier
	Quorum int32
}

// Option is a function that configures Options.
// This pattern allows for flexible, readable configuration.
type Option func(*Options)

// Permit represents an acquired semaphore permit.
type Permit struct {
	client    *Client
	name      string
	permitID  string
	holder    string
	ctx       context.Context
	cancelCtx context.CancelFunc
}

// NewPermit creates a new permit instance.
func NewPermit(client *Client, name, holder string, ctx context.Context) *Permit {
	return &Permit{
		client: client,
		name:   name,
		holder: holder,
		ctx:    ctx,
	}
}

func (p *Permit) Release(ctx context.Context) error {
	if p.cancelCtx != nil {
		p.cancelCtx()
	}
	if err := p.client.ReleaseSemaphorePermit(ctx, p.name, p.holder); err != nil {
		if client.IgnoreNotFound(err) == nil {
			return nil
		}
		return fmt.Errorf("failed to release permit %s for holder %s: %w", p.name, p.holder, err)
	}
	return nil
}

// Holder returns the permit holder identifier.
func (p *Permit) Holder() string {
	return p.holder
}

// Name returns the semaphore name.
func (p *Permit) Name() string {
	return p.name
}

// LeaseHandle represents an acquired lease.
type LeaseHandle struct {
	client    *Client
	name      string
	holder    string
	ctx       context.Context
	cancelCtx context.CancelFunc
}

// Release releases the lease.
func (l *LeaseHandle) Release(ctx context.Context) error {
	if l.cancelCtx != nil {
		l.cancelCtx()
	}
	if err := l.client.ReleaseLease(ctx, l.name, l.holder); err != nil {
		if client.IgnoreNotFound(err) == nil {
			return nil
		}
		return fmt.Errorf("failed to release lease %s for holder %s: %w", l.name, l.holder, err)
	}
	return nil
}

// Holder returns the lease holder identifier.
func (l *LeaseHandle) Holder() string {
	return l.holder
}

// Name returns the lease name.
func (l *LeaseHandle) Name() string {
	return l.name
}

// WithTTL sets the time-to-live for the operation.
// Resources with TTL will be automatically cleaned up after expiration.
//
// Example:
//
//	client.AcquireSemaphore(ctx, "api-limit", client.WithTTL(5*time.Minute))
func WithTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.TTL = ttl
	}
}

// WithTimeout sets the timeout for waiting operations.
// If the operation cannot complete within this time, it will fail.
//
// Example:
//
//	client.AcquireSemaphore(ctx, "api-limit", client.WithTimeout(30*time.Second))
func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.Timeout = timeout
	}
}

// WithPriority sets the priority for lease operations.
// Higher priority requests will be granted leases before lower priority ones.
//
// Example:
//
//	client.AcquireLease(ctx, "singleton", client.WithPriority(10))
func WithPriority(priority int32) Option {
	return func(o *Options) {
		o.Priority = priority
	}
}

// WithHolder sets the holder identifier for the operation.
// If not specified, defaults to the hostname or a generated identifier.
//
// Example:
//
//	client.AcquireSemaphore(ctx, "api-limit", client.WithHolder("worker-1"))
func WithHolder(holder string) Option {
	return func(o *Options) {
		o.Holder = holder
	}
}

// WithQuorum sets the minimum number of arrivals needed to open a barrier.
// If not specified, all expected arrivals are required.
//
// Example:
//
//	client.CreateBarrier(ctx, "stage-gate", 10, client.WithQuorum(7))
func WithQuorum(quorum int32) Option {
	return func(o *Options) {
		o.Quorum = quorum
	}
}
