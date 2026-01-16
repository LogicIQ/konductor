package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	konductor "github.com/LogicIQ/konductor/sdk/go/client"
	"github.com/LogicIQ/konductor/sdk/go/once"
)

func newOnceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "once",
		Short: "Manage once primitives",
		Long:  "Ensure actions execute exactly once across multiple pods",
	}

	cmd.AddCommand(newOnceCreateCmd())
	cmd.AddCommand(newOnceDeleteCmd())
	cmd.AddCommand(newOnceCheckCmd())
	cmd.AddCommand(newOnceListCmd())

	return cmd
}

func createOnceClient() (*konductor.Client, error) {
	if k8sClient == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}
	return konductor.NewFromClient(k8sClient, namespace), nil
}

func newOnceCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check <once-name>",
		Short: "Check if once has been executed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			client, err := createOnceClient()
			if err != nil {
				return err
			}

			executed, err := once.IsExecuted(client, ctx, name)
			if err != nil {
				return err
			}

			if executed {
				logger.Info("Once has been executed", zap.String("once", name))
			} else {
				logger.Info("Once has not been executed", zap.String("once", name))
			}

			return nil
		},
	}

	return cmd
}

func newOnceListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all onces",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			client, err := createOnceClient()
			if err != nil {
				return err
			}

			onces, err := once.List(client, ctx)
			if err != nil {
				return err
			}

			if len(onces) == 0 {
				logger.Info("No onces found")
				return nil
			}

			for _, o := range onces {
				executor := o.Status.Executor
				if executor == "" {
					executor = "N/A"
				}

				executedAt := "N/A"
				if o.Status.ExecutedAt != nil {
					executedAt = o.Status.ExecutedAt.Format("15:04:05")
				}

				logger.Info("Once",
					zap.String("name", o.Name),
					zap.Bool("executed", o.Status.Executed),
					zap.String("executor", executor),
					zap.String("phase", string(o.Status.Phase)),
					zap.String("executedAt", executedAt),
				)
			}

			return nil
		},
	}

	return cmd
}

func newOnceCreateCmd() *cobra.Command {
	var ttl time.Duration

	cmd := &cobra.Command{
		Use:   "create <once-name>",
		Short: "Create a once",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			client, err := createOnceClient()
			if err != nil {
				return err
			}

			var opts []konductor.Option
			if ttl > 0 {
				opts = append(opts, konductor.WithTTL(ttl))
			}
			if err := once.Create(client, ctx, name, opts...); err != nil {
				return err
			}

			logger.Info("Created once", zap.String("once", name))
			return nil
		},
	}

	cmd.Flags().DurationVar(&ttl, "ttl", 0, "Optional TTL for cleanup")

	return cmd
}

func newOnceDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <once-name>",
		Short: "Delete a once",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			client, err := createOnceClient()
			if err != nil {
				return err
			}

			if err := once.Delete(client, ctx, name); err != nil {
				return err
			}

			logger.Info("Deleted once", zap.String("once", name))
			return nil
		},
	}

	return cmd
}
