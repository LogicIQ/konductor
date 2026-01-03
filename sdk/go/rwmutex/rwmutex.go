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

func (m *RWMutex) Unlock() error {
	var rwmutex syncv1.RWMutex
	if err := m.client.K8sClient().Get(context.Background(), types.NamespacedName{
		Name:      m.name,
		Namespace: m.client.Namespace(),
	}, &rwmutex); err != nil {
		return fmt.Errorf("failed to get rwmutex: %w", err)
	}

	if m.isRead {
		return m.runlock(&rwmutex)
	}
	return m.wunlock(&rwmutex)
}

func (m *RWMutex) runlock(rwmutex *syncv1.RWMutex) error {
	holders := []string{}
	for _, h := range rwmutex.Status.ReadHolders {
		if h != m.holder {
			holders = append(holders, h)
		}
	}
	rwmutex.Status.ReadHolders = holders

	if len(holders) == 0 {
		rwmutex.Status.Phase = syncv1.RWMutexPhaseUnlocked
		rwmutex.Status.LockedAt = nil
		rwmutex.Status.ExpiresAt = nil
	}

	return m.client.K8sClient().Status().Update(context.Background(), rwmutex)
}

func (m *RWMutex) wunlock(rwmutex *syncv1.RWMutex) error {
	if rwmutex.Status.WriteHolder != m.holder {
		return fmt.Errorf("cannot unlock: not the holder")
	}

	rwmutex.Status.Phase = syncv1.RWMutexPhaseUnlocked
	rwmutex.Status.WriteHolder = ""
	rwmutex.Status.LockedAt = nil
	rwmutex.Status.ExpiresAt = nil

	return m.client.K8sClient().Status().Update(context.Background(), rwmutex)
}

func (m *RWMutex) Holder() string {
	return m.holder
}

func (m *RWMutex) Name() string {
	return m.name
}

func RLock(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) (*RWMutex, error) {
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
		var rwmutex syncv1.RWMutex
		if err := c.K8sClient().Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: c.Namespace(),
		}, &rwmutex); err != nil {
			return nil, fmt.Errorf("failed to get rwmutex: %w", err)
		}

		if rwmutex.Status.WriteHolder == "" {
			rwmutex.Status.Phase = syncv1.RWMutexPhaseReadLocked
			rwmutex.Status.ReadHolders = append(rwmutex.Status.ReadHolders, holder)
			
			if rwmutex.Status.LockedAt == nil {
				lockedAt := metav1.Now()
				rwmutex.Status.LockedAt = &lockedAt
			}

			if rwmutex.Spec.TTL != nil {
				expiresAt := metav1.NewTime(time.Now().Add(rwmutex.Spec.TTL.Duration))
				rwmutex.Status.ExpiresAt = &expiresAt
			}

			if err := c.K8sClient().Status().Update(ctx, &rwmutex); err != nil {
				if options.Timeout > 0 && time.Since(startTime) > options.Timeout {
					return nil, fmt.Errorf("timeout acquiring read lock on %s", name)
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			return &RWMutex{
				client: c,
				name:   name,
				holder: holder,
				isRead: true,
			}, nil
		}

		if options.Timeout > 0 && time.Since(startTime) > options.Timeout {
			return nil, fmt.Errorf("timeout acquiring read lock on %s", name)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}

func Lock(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) (*RWMutex, error) {
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
		var rwmutex syncv1.RWMutex
		if err := c.K8sClient().Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: c.Namespace(),
		}, &rwmutex); err != nil {
			return nil, fmt.Errorf("failed to get rwmutex: %w", err)
		}

		if rwmutex.Status.WriteHolder == "" && len(rwmutex.Status.ReadHolders) == 0 {
			rwmutex.Status.Phase = syncv1.RWMutexPhaseWriteLocked
			rwmutex.Status.WriteHolder = holder
			lockedAt := metav1.Now()
			rwmutex.Status.LockedAt = &lockedAt

			if rwmutex.Spec.TTL != nil {
				expiresAt := metav1.NewTime(time.Now().Add(rwmutex.Spec.TTL.Duration))
				rwmutex.Status.ExpiresAt = &expiresAt
			}

			if err := c.K8sClient().Status().Update(ctx, &rwmutex); err != nil {
				if options.Timeout > 0 && time.Since(startTime) > options.Timeout {
					return nil, fmt.Errorf("timeout acquiring write lock on %s", name)
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			return &RWMutex{
				client: c,
				name:   name,
				holder: holder,
				isRead: false,
			}, nil
		}

		if options.Timeout > 0 && time.Since(startTime) > options.Timeout {
			return nil, fmt.Errorf("timeout acquiring write lock on %s", name)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
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
