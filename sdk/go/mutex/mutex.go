package mutex

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

// Mutex represents an acquired mutex lock
type Mutex struct {
	client *konductor.Client
	name   string
	holder string
}

func (m *Mutex) Unlock() error {
	var mutex syncv1.Mutex
	if err := m.client.K8sClient().Get(context.Background(), types.NamespacedName{
		Name:      m.name,
		Namespace: m.client.Namespace(),
	}, &mutex); err != nil {
		return fmt.Errorf("failed to get mutex: %w", err)
	}

	if mutex.Status.Holder != m.holder {
		return fmt.Errorf("cannot unlock: not the holder")
	}

	mutex.Status.Phase = syncv1.MutexPhaseUnlocked
	mutex.Status.Holder = ""
	mutex.Status.LockedAt = nil
	mutex.Status.ExpiresAt = nil

	return m.client.K8sClient().Status().Update(context.Background(), &mutex)
}

func (m *Mutex) Holder() string {
	return m.holder
}

func (m *Mutex) Name() string {
	return m.name
}

func Lock(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) (*Mutex, error) {
	options := &konductor.Options{Timeout: 0}
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

	startTime := time.Now()

	for {
		var mutex syncv1.Mutex
		if err := c.K8sClient().Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: c.Namespace(),
		}, &mutex); err != nil {
			return nil, fmt.Errorf("failed to get mutex: %w", err)
		}

		if mutex.Status.Phase == syncv1.MutexPhaseUnlocked || mutex.Status.Holder == "" {
			mutex.Status.Phase = syncv1.MutexPhaseLocked
			mutex.Status.Holder = holder
			lockedAt := metav1.Now()
			mutex.Status.LockedAt = &lockedAt

			if mutex.Spec.TTL != nil {
				expiresAt := metav1.NewTime(time.Now().Add(mutex.Spec.TTL.Duration))
				mutex.Status.ExpiresAt = &expiresAt
			}

			if err := c.K8sClient().Status().Update(ctx, &mutex); err != nil {
				if options.Timeout > 0 && time.Since(startTime) > options.Timeout {
					return nil, fmt.Errorf("timeout acquiring mutex %s", name)
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			return &Mutex{
				client: c,
				name:   name,
				holder: holder,
			}, nil
		}

		if options.Timeout > 0 && time.Since(startTime) > options.Timeout {
			return nil, fmt.Errorf("timeout acquiring mutex %s", name)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}

func TryLock(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) (*Mutex, error) {
	opts = append(opts, konductor.WithTimeout(1*time.Millisecond))
	return Lock(c, ctx, name, opts...)
}

func With(c *konductor.Client, ctx context.Context, name string, fn func() error, opts ...konductor.Option) error {
	mutex, err := Lock(c, ctx, name, opts...)
	if err != nil {
		return err
	}
	defer func() {
		if unlockErr := mutex.Unlock(); unlockErr != nil {
			// TODO: Add proper logging for unlock errors
			// Cannot return error from defer, so we silently handle it for now
			_ = unlockErr
		}
	}()

	return fn()
}

func Create(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) error {
	options := &konductor.Options{}
	for _, opt := range opts {
		opt(options)
	}

	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
		Spec: syncv1.MutexSpec{},
	}

	if options.TTL > 0 {
		mutex.Spec.TTL = &metav1.Duration{Duration: options.TTL}
	}

	return c.K8sClient().Create(ctx, mutex)
}

func Delete(c *konductor.Client, ctx context.Context, name string) error {
	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
	}
	return c.K8sClient().Delete(ctx, mutex)
}

func Get(c *konductor.Client, ctx context.Context, name string) (*syncv1.Mutex, error) {
	var mutex syncv1.Mutex
	if err := c.K8sClient().Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace(),
	}, &mutex); err != nil {
		return nil, fmt.Errorf("failed to get mutex %s: %w", name, err)
	}
	return &mutex, nil
}

func List(c *konductor.Client, ctx context.Context) ([]syncv1.Mutex, error) {
	var mutexes syncv1.MutexList
	if err := c.K8sClient().List(ctx, &mutexes, client.InNamespace(c.Namespace())); err != nil {
		return nil, fmt.Errorf("failed to list mutexes: %w", err)
	}
	return mutexes.Items, nil
}

func IsLocked(c *konductor.Client, ctx context.Context, name string) (bool, error) {
	mutex, err := Get(c, ctx, name)
	if err != nil {
		return false, err
	}
	return mutex.Status.Phase == syncv1.MutexPhaseLocked, nil
}
