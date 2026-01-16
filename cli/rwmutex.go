package main

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	konductor "github.com/LogicIQ/konductor/sdk/go/client"
	"github.com/LogicIQ/konductor/sdk/go/rwmutex"
)

func newRWMutexCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rwmutex",
		Short: "Manage read-write mutexes",
		Long:  "Lock (read/write), unlock, and manage read-write mutexes",
	}

	cmd.AddCommand(newRWMutexCreateCmd())
	cmd.AddCommand(newRWMutexDeleteCmd())
	cmd.AddCommand(newRWMutexRLockCmd())
	cmd.AddCommand(newRWMutexLockCmd())
	cmd.AddCommand(newRWMutexUnlockCmd())
	cmd.AddCommand(newRWMutexListCmd())

	return cmd
}

func newRWMutexRLockCmd() *cobra.Command {
	var (
		timeout time.Duration
		holder  string
	)

	cmd := &cobra.Command{
		Use:   "rlock <rwmutex-name>",
		Short: "Acquire read lock",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			if holder == "" {
				holder = os.Getenv("HOSTNAME")
				if holder == "" {
					return errors.New("holder must be specified or HOSTNAME must be set")
				}
			}

			client := konductor.NewFromClient(k8sClient, namespace)

			var opts []konductor.Option
			opts = append(opts, konductor.WithHolder(holder))
			if timeout > 0 {
				opts = append(opts, konductor.WithTimeout(timeout))
			}

			rwm, err := rwmutex.RLock(client, ctx, name, opts...)
			if err != nil {
				return err
			}

			logger.Info("Acquired read lock", zap.String("rwmutex", name), zap.String("holder", rwm.Holder()))
			return nil
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Timeout for waiting (e.g., 30s, 5m)")
	cmd.Flags().StringVar(&holder, "holder", "", "Lock holder identifier (defaults to hostname)")

	return cmd
}

func newRWMutexLockCmd() *cobra.Command {
	var (
		timeout time.Duration
		holder  string
	)

	cmd := &cobra.Command{
		Use:   "lock <rwmutex-name>",
		Short: "Acquire write lock",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			if holder == "" {
				holder = os.Getenv("HOSTNAME")
				if holder == "" {
					return errors.New("holder must be specified or HOSTNAME must be set")
				}
			}

			client := konductor.NewFromClient(k8sClient, namespace)

			var opts []konductor.Option
			opts = append(opts, konductor.WithHolder(holder))
			if timeout > 0 {
				opts = append(opts, konductor.WithTimeout(timeout))
			}

			rwm, err := rwmutex.Lock(client, ctx, name, opts...)
			if err != nil {
				return err
			}

			logger.Info("Acquired write lock", zap.String("rwmutex", name), zap.String("holder", rwm.Holder()))
			return nil
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Timeout for waiting (e.g., 30s, 5m)")
	cmd.Flags().StringVar(&holder, "holder", "", "Lock holder identifier (defaults to hostname)")

	return cmd
}

func newRWMutexUnlockCmd() *cobra.Command {
	var holder string

	cmd := &cobra.Command{
		Use:   "unlock <rwmutex-name>",
		Short: "Release lock",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			if holder == "" {
				holder = os.Getenv("HOSTNAME")
				if holder == "" {
					return errors.New("holder must be specified or HOSTNAME must be set")
				}
			}

			client := konductor.NewFromClient(k8sClient, namespace)

			m, err := rwmutex.Get(client, ctx, name)
			if err != nil {
				return err
			}

			isReader := false
			for _, h := range m.Status.ReadHolders {
				if h == holder {
					isReader = true
					break
				}
			}

			if !isReader && m.Status.WriteHolder != holder {
				return errors.New("cannot unlock: not a holder")
			}

			rwm := &rwmutex.RWMutex{}
			if isReader {
				holders := []string{}
				for _, h := range m.Status.ReadHolders {
					if h != holder {
						holders = append(holders, h)
					}
				}
				m.Status.ReadHolders = holders
				if len(holders) == 0 {
					m.Status.Phase = "Unlocked"
					m.Status.LockedAt = nil
					m.Status.ExpiresAt = nil
				}
			} else {
				m.Status.Phase = "Unlocked"
				m.Status.WriteHolder = ""
				m.Status.LockedAt = nil
				m.Status.ExpiresAt = nil
			}

			if err := client.K8sClient().Status().Update(ctx, m); err != nil {
				return err
			}

			logger.Info("Released lock", zap.String("rwmutex", name), zap.String("holder", holder))
			_ = rwm
			return nil
		},
	}

	cmd.Flags().StringVar(&holder, "holder", "", "Lock holder identifier (defaults to hostname)")

	return cmd
}

func newRWMutexListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all rwmutexes",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			client := konductor.NewFromClient(k8sClient, namespace)

			rwmutexes, err := rwmutex.List(client, ctx)
			if err != nil {
				return err
			}

			if len(rwmutexes) == 0 {
				logger.Info("No rwmutexes found")
				return nil
			}

			for _, m := range rwmutexes {
				writeHolder := m.Status.WriteHolder
				if writeHolder == "" {
					writeHolder = "N/A"
				}

				locked := "N/A"
				if m.Status.LockedAt != nil {
					locked = m.Status.LockedAt.Format("15:04:05")
				}

				logger.Info("RWMutex",
					zap.String("name", m.Name),
					zap.String("writeHolder", writeHolder),
					zap.Int("readers", len(m.Status.ReadHolders)),
					zap.String("phase", string(m.Status.Phase)),
					zap.String("locked", locked),
				)
			}

			return nil
		},
	}

	return cmd
}

func newRWMutexCreateCmd() *cobra.Command {
	var ttl time.Duration

	cmd := &cobra.Command{
		Use:   "create <rwmutex-name>",
		Short: "Create a rwmutex",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			client := konductor.NewFromClient(k8sClient, namespace)

			var opts []konductor.Option
			if ttl > 0 {
				opts = append(opts, konductor.WithTTL(ttl))
			}
			if err := rwmutex.Create(client, ctx, name, opts...); err != nil {
				return err
			}

			logger.Info("Created rwmutex", zap.String("rwmutex", name))
			return nil
		},
	}

	cmd.Flags().DurationVar(&ttl, "ttl", 0, "Optional TTL for automatic unlock")

	return cmd
}

func newRWMutexDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <rwmutex-name>",
		Short: "Delete a rwmutex",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			client := konductor.NewFromClient(k8sClient, namespace)

			if err := rwmutex.Delete(client, ctx, name); err != nil {
				return err
			}

			logger.Info("Deleted rwmutex", zap.String("rwmutex", name))
			return nil
		},
	}

	return cmd
}
