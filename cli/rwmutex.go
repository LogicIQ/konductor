package main

import (
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
			ctx := cmd.Context()

			var err error
			holder, err = validateHolder(holder)
			if err != nil {
				return err
			}

			client := konductor.NewFromClient(k8sClient, namespace)

			opts := []konductor.Option{
				konductor.WithHolder(holder),
			}
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
			ctx := cmd.Context()

			var err error
			holder, err = validateHolder(holder)
			if err != nil {
				return err
			}

			client := konductor.NewFromClient(k8sClient, namespace)

			opts := []konductor.Option{
				konductor.WithHolder(holder),
			}
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
			ctx := cmd.Context()

			var err error
			holder, err = validateHolder(holder)
			if err != nil {
				return err
			}

			client := konductor.NewFromClient(k8sClient, namespace)

			if err := rwmutex.Unlock(client, ctx, name, holder); err != nil {
				return err
			}

			logger.Info("Released lock", zap.String("rwmutex", name), zap.String("holder", holder))
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
			ctx := cmd.Context()

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
			ctx := cmd.Context()

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
			ctx := cmd.Context()

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
