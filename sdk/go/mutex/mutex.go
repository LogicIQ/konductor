package mutex

import (
	"context"
	"fmt"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
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
	// Use retry for unlock to handle concurrent operations
	return m.client.RetryWithBackoff(context.Background(), func() error {
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
	}, nil)
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

	mutex := &syncv1.Mutex{}
	mutex.Name = name
	mutex.Namespace = c.Namespace()

	config := &konductor.WaitConfig{
		InitialDelay: 1 * time.Second,
		MaxDelay: 5 * time.Second,
		Factor: 1.5,
		Jitter: 0.1,
		Timeout: 30 * time.Second,
	}

	if options.Timeout > 0 {
		config.Timeout = options.Timeout
	}

	// Wait for mutex to be unlocked
	err := c.WaitForCondition(ctx, mutex, func(obj interface{}) bool {
		m := obj.(*syncv1.Mutex)
		return m.Status.Phase != syncv1.MutexPhaseLocked || m.Status.Holder == ""
	}, config)

	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context cancelled while waiting for mutex %s: %w", name, ctx.Err())
		}
		return nil, fmt.Errorf("timeout acquiring mutex %s: %w", name, err)
	}

	// Now try to acquire the lock
	err = c.RetryWithBackoff(ctx, func() error {
		var m syncv1.Mutex
		if err := c.K8sClient().Get(ctx, types.NamespacedName{
			Name: name, Namespace: c.Namespace(),
		}, &m); err != nil {
			return err
		}

		// Atomic check: only proceed if truly unlocked
		if m.Status.Phase == syncv1.MutexPhaseLocked && m.Status.Holder != "" {
			return fmt.Errorf("mutex locked by %s", m.Status.Holder)
		}

		// Atomic set: this will fail with 409 if another pod modified it
		m.Status.Phase = syncv1.MutexPhaseLocked
		m.Status.Holder = holder
		lockedAt := metav1.Now()
		m.Status.LockedAt = &lockedAt

		if m.Spec.TTL != nil {
			expiresAt := metav1.NewTime(time.Now().Add(m.Spec.TTL.Duration))
			m.Status.ExpiresAt = &expiresAt
		}

		// Critical: Update will fail with conflict if resource version changed
		return c.K8sClient().Status().Update(ctx, &m)
	}, &konductor.WaitConfig{InitialDelay: 100 * time.Millisecond, MaxDelay: 1 * time.Second, Timeout: 5 * time.Second})

	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context cancelled while acquiring mutex %s: %w", name, ctx.Err())
		}
		return nil, fmt.Errorf("failed to acquire mutex lock %s: %w", name, err)
	}

	// Wait for confirmation
	mutexObj := &Mutex{client: c, name: name, holder: holder}
	if err := c.WaitForCondition(ctx, mutex, func(obj interface{}) bool {
		m := obj.(*syncv1.Mutex)
		return m.Status.Phase == syncv1.MutexPhaseLocked && m.Status.Holder == holder
	}, &konductor.WaitConfig{InitialDelay: 10 * time.Millisecond, MaxDelay: 100 * time.Millisecond, Timeout: 2 * time.Second}); err != nil {
		return nil, fmt.Errorf("failed to confirm mutex lock: %w", err)
	}
	
	return mutexObj, nil
}

func TryLock(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) (*Mutex, error) {
	opts = append(opts, konductor.WithTimeout(1*time.Millisecond))
	return Lock(c, ctx, name, opts...)
}

func With(c *konductor.Client, ctx context.Context, name string, fn func() error, opts ...konductor.Option) (err error) {
	mutex, err := Lock(c, ctx, name, opts...)
	if err != nil {
		return err
	}
	defer func() {
		if unlockErr := mutex.Unlock(); unlockErr != nil {
			if err == nil {
				err = fmt.Errorf("function succeeded but failed to unlock mutex: %w", unlockErr)
			}
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

	err := c.K8sClient().Create(ctx, mutex)
	if err != nil && errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func Delete(c *konductor.Client, ctx context.Context, name string) error {
	mutex := &syncv1.Mutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
	}
	err := c.K8sClient().Delete(ctx, mutex)
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	return err
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

func Unlock(c *konductor.Client, ctx context.Context, name, holder string) error {
	return c.RetryWithBackoff(ctx, func() error {
		var mutex syncv1.Mutex
		if err := c.K8sClient().Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: c.Namespace(),
		}, &mutex); err != nil {
			return fmt.Errorf("failed to get mutex: %w", err)
		}

		if mutex.Status.Holder != holder {
			return fmt.Errorf("cannot unlock: not the holder")
		}

		mutex.Status.Phase = syncv1.MutexPhaseUnlocked
		mutex.Status.Holder = ""
		mutex.Status.LockedAt = nil
		mutex.Status.ExpiresAt = nil

		return c.K8sClient().Status().Update(ctx, &mutex)
	}, nil)
}
