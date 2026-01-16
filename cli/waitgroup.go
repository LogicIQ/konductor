package main

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	konductor "github.com/LogicIQ/konductor/sdk/go/client"
	"github.com/LogicIQ/konductor/sdk/go/waitgroup"
)

func newWaitGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "waitgroup",
		Short: "Manage waitgroups",
		Long:  "Coordinate dynamic number of workers",
	}

	// Create shared client for all waitgroup commands
	client := konductor.NewFromClient(k8sClient, namespace)

	cmd.AddCommand(newWaitGroupCreateCmd(client))
	cmd.AddCommand(newWaitGroupDeleteCmd(client))
	cmd.AddCommand(newWaitGroupAddCmd(client))
	cmd.AddCommand(newWaitGroupDoneCmd(client))
	cmd.AddCommand(newWaitGroupWaitCmd(client))
	cmd.AddCommand(newWaitGroupListCmd(client))

	return cmd
}

func newWaitGroupAddCmd(client *konductor.Client) *cobra.Command {
	var delta int32

	cmd := &cobra.Command{
		Use:   "add <waitgroup-name>",
		Short: "Add to waitgroup counter",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			if err := waitgroup.Add(client, ctx, name, delta); err != nil {
				logger.Error("Failed to add to waitgroup", zap.Error(err))
				return err
			}

			logger.Info("Added to waitgroup", zap.String("waitgroup", name), zap.Int32("delta", delta))
			return nil
		},
	}

	cmd.Flags().Int32Var(&delta, "delta", 1, "Amount to add to counter")

	return cmd
}

func newWaitGroupDoneCmd(client *konductor.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "done <waitgroup-name>",
		Short: "Decrement waitgroup counter",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			if err := waitgroup.Done(client, ctx, name); err != nil {
				logger.Error("Failed to call done on waitgroup", zap.Error(err))
				return err
			}

			logger.Info("Done called on waitgroup", zap.String("waitgroup", name))
			return nil
		},
	}

	return cmd
}

func newWaitGroupWaitCmd(client *konductor.Client) *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "wait <waitgroup-name>",
		Short: "Wait for waitgroup counter to reach zero",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			var opts []konductor.Option
			if timeout > 0 {
				opts = append(opts, konductor.WithTimeout(timeout))
			}

			if err := waitgroup.Wait(client, ctx, name, opts...); err != nil {
				logger.Error("Failed to wait for waitgroup", zap.Error(err))
				return err
			}

			logger.Info("WaitGroup complete", zap.String("waitgroup", name))
			return nil
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Timeout for waiting (e.g., 30s, 5m)")

	return cmd
}

func newWaitGroupListCmd(client *konductor.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all waitgroups",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			wgs, err := waitgroup.List(client, ctx)
			if err != nil {
				logger.Error("Failed to list waitgroups", zap.Error(err))
				return err
			}

			if len(wgs) == 0 {
				logger.Info("No waitgroups found")
				return nil
			}

			for _, wg := range wgs {
				logger.Info("WaitGroup",
					zap.String("name", wg.Name),
					zap.Int32("counter", wg.Status.Counter),
					zap.String("phase", string(wg.Status.Phase)),
				)
			}

			return nil
		},
	}

	return cmd
}

func newWaitGroupCreateCmd(client *konductor.Client) *cobra.Command {
	var ttl time.Duration

	cmd := &cobra.Command{
		Use:   "create <waitgroup-name>",
		Short: "Create a waitgroup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			var opts []konductor.Option
			if ttl > 0 {
				opts = append(opts, konductor.WithTTL(ttl))
			}
			if err := waitgroup.Create(client, ctx, name, opts...); err != nil {
				logger.Error("Failed to create waitgroup", zap.Error(err))
				return err
			}

			logger.Info("Created waitgroup", zap.String("waitgroup", name))
			return nil
		},
	}

	cmd.Flags().DurationVar(&ttl, "ttl", 0, "Optional TTL for cleanup")

	return cmd
}

func newWaitGroupDeleteCmd(client *konductor.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <waitgroup-name>",
		Short: "Delete a waitgroup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			if err := waitgroup.Delete(client, ctx, name); err != nil {
				logger.Error("Failed to delete waitgroup", zap.Error(err))
				return err
			}

			logger.Info("Deleted waitgroup", zap.String("waitgroup", name))
			return nil
		},
	}

	return cmd
}
