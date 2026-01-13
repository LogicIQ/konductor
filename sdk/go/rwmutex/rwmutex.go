package rwmutex

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

// RWMutex represents an acquired rwmutex lock
type RWMutex struct {
	client *konductor.Client
	name   string
	holder string
	isRead bool
}

func (m *RWMutex) Unlock(ctx context.Context) error {
	var rwmutex syncv1.RWMutex
	if err := m.client.K8sClient().Get(ctx, types.NamespacedName{
		Name:      m.name,
		Namespace: m.client.Namespace(),
	}, &rwmutex); err != nil {
		return fmt.Errorf("failed to get rwmutex: %w", err)
	}

	if m.isRead {
		return m.runlock(ctx)
	}
	return m.wunlock(ctx)
}

func (m *RWMutex) runlock(ctx context.Context) error {
	return m.client.RetryWithBackoff(ctx, func() error {
		var rw syncv1.RWMutex
		if err := m.client.K8sClient().Get(ctx, types.NamespacedName{
			Name: m.name, Namespace: m.client.Namespace(),
		}, &rw); err != nil {
			return err
		}

		holders := []string{}
		for _, h := range rw.Status.ReadHolders {
			if h != m.holder {
				holders = append(holders, h)
			}
		}
		rw.Status.ReadHolders = holders

		if len(holders) == 0 {
			rw.Status.Phase = syncv1.RWMutexPhaseUnlocked
			rw.Status.LockedAt = nil
			rw.Status.ExpiresAt = nil
		}

		return m.client.K8sClient().Status().Update(ctx, &rw)
	}, nil)
}

func (m *RWMutex) wunlock(ctx context.Context) error {
	return m.client.RetryWithBackoff(ctx, func() error {
		var rw syncv1.RWMutex
		if err := m.client.K8sClient().Get(ctx, types.NamespacedName{
			Name: m.name, Namespace: m.client.Namespace(),
		}, &rw); err != nil {
			return err
		}

		if rw.Status.WriteHolder != m.holder {
			return fmt.Errorf("cannot unlock: not the holder")
		}

		rw.Status.Phase = syncv1.RWMutexPhaseUnlocked
		rw.Status.WriteHolder = ""
		rw.Status.LockedAt = nil
		rw.Status.ExpiresAt = nil

		return m.client.K8sClient().Status().Update(ctx, &rw)
	}, nil)
}

func (m *RWMutex) Holder() string {
	return m.holder
}

func (m *RWMutex) Name() string {
	return m.name
}

func getHolder(options *konductor.Options) string {
	if options.Holder != "" {
		return options.Holder
	}
	if hostname := os.Getenv("HOSTNAME"); hostname != "" {
		return hostname
	}
	return fmt.Sprintf("sdk-%d", time.Now().Unix())
}

func getWaitConfig(timeout time.Duration) *konductor.WaitConfig {
	config := &konductor.WaitConfig{
		InitialDelay: 1 * time.Second,
		MaxDelay:     5 * time.Second,
		Factor:       1.5,
		Jitter:       0.1,
		Timeout:      30 * time.Second,
	}
	if timeout > 0 {
		config.Timeout = timeout
	}
	return config
}

func RLock(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) (*RWMutex, error) {
	options := &konductor.Options{Timeout: 0}
	for _, opt := range opts {
		opt(options)
	}

	holder := getHolder(options)

	rwmutex := &syncv1.RWMutex{}
	rwmutex.Name = name
	rwmutex.Namespace = c.Namespace()

	config := getWaitConfig(options.Timeout)

	// Wait for write lock to be released
	err := c.WaitForCondition(ctx, rwmutex, func(obj interface{}) bool {
		rw := obj.(*syncv1.RWMutex)
		return rw.Status.WriteHolder == ""
	}, config)

	if err != nil {
		return nil, fmt.Errorf("timeout acquiring read lock on %s: %w", name, err)
	}

	// Now try to acquire read lock
	err = c.RetryWithBackoff(ctx, func() error {
		var rw syncv1.RWMutex
		if err := c.K8sClient().Get(ctx, types.NamespacedName{
			Name: name, Namespace: c.Namespace(),
		}, &rw); err != nil {
			return err
		}

		if rw.Status.WriteHolder != "" {
			return fmt.Errorf("write locked by %s", rw.Status.WriteHolder)
		}

		rw.Status.Phase = syncv1.RWMutexPhaseReadLocked
		rw.Status.ReadHolders = append(rw.Status.ReadHolders, holder)

		if rw.Status.LockedAt == nil {
			lockedAt := metav1.Now()
			rw.Status.LockedAt = &lockedAt
		}

		if rw.Spec.TTL != nil {
			expiresAt := metav1.NewTime(time.Now().Add(rw.Spec.TTL.Duration))
			rw.Status.ExpiresAt = &expiresAt
		}

		return c.K8sClient().Status().Update(ctx, &rw)
	}, &konductor.WaitConfig{InitialDelay: 100 * time.Millisecond, MaxDelay: 1 * time.Second, Timeout: 5 * time.Second})

	if err != nil {
		return nil, err
	}

	// Wait for confirmation
	mutex := &RWMutex{client: c, name: name, holder: holder, isRead: true}
	return mutex, c.WaitForCondition(ctx, rwmutex, func(obj interface{}) bool {
		rw := obj.(*syncv1.RWMutex)
		for _, h := range rw.Status.ReadHolders {
			if h == holder {
				return true
			}
		}
		return false
	}, &konductor.WaitConfig{InitialDelay: 100 * time.Millisecond, MaxDelay: 1 * time.Second, Timeout: 2 * time.Second})
}

func Lock(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) (*RWMutex, error) {
	options := &konductor.Options{Timeout: 0}
	for _, opt := range opts {
		opt(options)
	}

	holder := getHolder(options)

	rwmutex := &syncv1.RWMutex{}
	rwmutex.Name = name
	rwmutex.Namespace = c.Namespace()

	config := getWaitConfig(options.Timeout)

	// Wait for rwmutex to be completely unlocked
	err := c.WaitForCondition(ctx, rwmutex, func(obj interface{}) bool {
		rw := obj.(*syncv1.RWMutex)
		return rw.Status.WriteHolder == "" && len(rw.Status.ReadHolders) == 0
	}, config)

	if err != nil {
		return nil, fmt.Errorf("timeout acquiring write lock on %s: %w", name, err)
	}

	// Now try to acquire write lock
	err = c.RetryWithBackoff(ctx, func() error {
		var rw syncv1.RWMutex
		if err := c.K8sClient().Get(ctx, types.NamespacedName{
			Name: name, Namespace: c.Namespace(),
		}, &rw); err != nil {
			return err
		}

		if rw.Status.WriteHolder != "" || len(rw.Status.ReadHolders) > 0 {
			return fmt.Errorf("rwmutex locked")
		}

		rw.Status.Phase = syncv1.RWMutexPhaseWriteLocked
		rw.Status.WriteHolder = holder
		lockedAt := metav1.Now()
		rw.Status.LockedAt = &lockedAt

		if rw.Spec.TTL != nil {
			expiresAt := metav1.NewTime(time.Now().Add(rw.Spec.TTL.Duration))
			rw.Status.ExpiresAt = &expiresAt
		}

		return c.K8sClient().Status().Update(ctx, &rw)
	}, &konductor.WaitConfig{InitialDelay: 100 * time.Millisecond, MaxDelay: 1 * time.Second, Timeout: 5 * time.Second})

	if err != nil {
		return nil, err
	}

	// Wait for confirmation
	mutex := &RWMutex{client: c, name: name, holder: holder, isRead: false}
	return mutex, c.WaitForCondition(ctx, rwmutex, func(obj interface{}) bool {
		rw := obj.(*syncv1.RWMutex)
		return rw.Status.Phase == syncv1.RWMutexPhaseWriteLocked && rw.Status.WriteHolder == holder
	}, &konductor.WaitConfig{InitialDelay: 100 * time.Millisecond, MaxDelay: 1 * time.Second, Timeout: 2 * time.Second})
}

func Create(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) error {
	options := &konductor.Options{}
	for _, opt := range opts {
		opt(options)
	}

	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
		Spec: syncv1.RWMutexSpec{},
	}

	if options.TTL > 0 {
		rwmutex.Spec.TTL = &metav1.Duration{Duration: options.TTL}
	}

	return c.K8sClient().Create(ctx, rwmutex)
}

func Delete(c *konductor.Client, ctx context.Context, name string) error {
	rwmutex := &syncv1.RWMutex{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace(),
		},
	}
	return c.K8sClient().Delete(ctx, rwmutex)
}

func Get(c *konductor.Client, ctx context.Context, name string) (*syncv1.RWMutex, error) {
	var rwmutex syncv1.RWMutex
	if err := c.K8sClient().Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: c.Namespace(),
	}, &rwmutex); err != nil {
		return nil, fmt.Errorf("failed to get rwmutex %s: %w", name, err)
	}
	return &rwmutex, nil
}

func List(c *konductor.Client, ctx context.Context) ([]syncv1.RWMutex, error) {
	var rwmutexes syncv1.RWMutexList
	if err := c.K8sClient().List(ctx, &rwmutexes, client.InNamespace(c.Namespace())); err != nil {
		return nil, fmt.Errorf("failed to list rwmutexes: %w", err)
	}
	return rwmutexes.Items, nil
}
