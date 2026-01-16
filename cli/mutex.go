package main

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	konductor "github.com/LogicIQ/konductor/sdk/go/client"
	"github.com/LogicIQ/konductor/sdk/go/mutex"
)

func newMutexCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mutex",
		Short: "Manage mutexes",
		Long:  "Lock, unlock, and manage mutexes for mutual exclusion",
	}

	cmd.AddCommand(newMutexCreateCmd())
	cmd.AddCommand(newMutexDeleteCmd())
	cmd.AddCommand(newMutexLockCmd())
	cmd.AddCommand(newMutexUnlockCmd())
	cmd.AddCommand(newMutexListCmd())

	return cmd
}

func createMutexClient() *konductor.Client {
	return konductor.NewFromClient(k8sClient, namespace)
}

func newMutexLockCmd() *cobra.Command {
	var (
		timeout time.Duration
		holder  string
	)

	cmd := &cobra.Command{
		Use:   "lock <mutex-name>",
		Short: "Lock a mutex",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mutexName := args[0]
			ctx := context.Background()

			client := createMutexClient()

			var opts []konductor.Option
			if holder != "" {
				opts = append(opts, konductor.WithHolder(holder))
			}
			if timeout > 0 {
				opts = append(opts, konductor.WithTimeout(timeout))
			}

			mutexObj, err := mutex.Lock(client, ctx, mutexName, opts...)
			if err != nil {
				return err
			}

			logger.Info("Locked mutex", zap.String("mutex", mutexName), zap.String("holder", mutexObj.Holder()))
			return nil
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Timeout for waiting (e.g., 30s, 5m)")
	cmd.Flags().StringVar(&holder, "holder", "", "Lock holder identifier (defaults to hostname)")

	return cmd
}

func newMutexUnlockCmd() *cobra.Command {
	var holder string

	cmd := &cobra.Command{
		Use:   "unlock <mutex-name>",
		Short: "Unlock a mutex",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mutexName := args[0]
			ctx := context.Background()

			if holder == "" {
				holder = os.Getenv("HOSTNAME")
				if holder == "" {
					return errors.New("holder must be specified or HOSTNAME must be set")
				}
			}

			client := createMutexClient()

			if err := mutex.Unlock(client, ctx, mutexName, holder); err != nil {
				return err
			}

			logger.Info("Unlocked mutex", zap.String("mutex", mutexName), zap.String("holder", holder))
			return nil
		},
	}

	cmd.Flags().StringVar(&holder, "holder", "", "Lock holder identifier (defaults to hostname)")

	return cmd
}

func newMutexListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all mutexes",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			client := createMutexClient()

			mutexes, err := mutex.List(client, ctx)
			if err != nil {
				return err
			}

			if len(mutexes) == 0 {
				logger.Info("No mutexes found")
				return nil
			}

			for _, m := range mutexes {
				holder := m.Status.Holder
				if holder == "" {
					holder = "N/A"
				}

				locked := "N/A"
				if m.Status.LockedAt != nil {
					locked = m.Status.LockedAt.Format("15:04:05")
				}

				logger.Info("Mutex",
					zap.String("name", m.Name),
					zap.String("holder", holder),
					zap.String("phase", string(m.Status.Phase)),
					zap.String("locked", locked),
				)
			}

			return nil
		},
	}

	return cmd
}

func newMutexCreateCmd() *cobra.Command {
	var ttl time.Duration

	cmd := &cobra.Command{
		Use:   "create <mutex-name>",
		Short: "Create a mutex",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mutexName := args[0]
			ctx := context.Background()

			client := createMutexClient()

			var opts []konductor.Option
			if ttl > 0 {
				opts = append(opts, konductor.WithTTL(ttl))
			}
			if err := mutex.Create(client, ctx, mutexName, opts...); err != nil {
				return err
			}

			logger.Info("Created mutex", zap.String("mutex", mutexName))
			return nil
		},
	}

	cmd.Flags().DurationVar(&ttl, "ttl", 0, "Optional TTL for automatic unlock")

	return cmd
}

func newMutexDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <mutex-name>",
		Short: "Delete a mutex",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mutexName := args[0]
			ctx := context.Background()

			client := createMutexClient()

			if err := mutex.Delete(client, ctx, mutexName); err != nil {
				return err
			}

			logger.Info("Deleted mutex", zap.String("mutex", mutexName))
			return nil
		},
	}

	return cmd
}
