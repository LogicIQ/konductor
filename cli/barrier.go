package main

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
	"github.com/LogicIQ/konductor/sdk/go/barrier"
	konductor "github.com/LogicIQ/konductor/sdk/go/client"
)

func newBarrierCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "barrier",
		Short: "Manage barriers",
		Long:  "Wait for barriers and signal arrivals",
	}

	cmd.AddCommand(newBarrierCreateCmd())
	cmd.AddCommand(newBarrierDeleteCmd())
	cmd.AddCommand(newBarrierWaitCmd())
	cmd.AddCommand(newBarrierArriveCmd())
	cmd.AddCommand(newBarrierListCmd())

	return cmd
}

func newBarrierWaitCmd() *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "wait <barrier-name>",
		Short: "Wait for a barrier to open",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			barrierName := args[0]
			ctx := context.Background()

			// Create SDK client
			client := konductor.NewFromClient(k8sClient, namespace)

			// Build options
			var opts []konductor.Option
			if timeout > 0 {
				opts = append(opts, konductor.WithTimeout(timeout))
			}

			// Wait for barrier using SDK
			if err := barrier.Wait(client, ctx, barrierName, opts...); err != nil {
				return err
			}

			// Get barrier status to display info
			barrierObj, err := barrier.Get(client, ctx, barrierName)
			if err == nil && barrierObj != nil {
				logger.Info("Barrier is open",
					zap.String("barrier", barrierName),
					zap.Int32("arrived", barrierObj.Status.Arrived),
					zap.Int32("expected", barrierObj.Spec.Expected),
				)
			}

			return nil
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Timeout for waiting (e.g., 30s, 5m)")

	return cmd
}

func newBarrierArriveCmd() *cobra.Command {
	var (
		holder        string
		waitForUpdate bool
	)

	cmd := &cobra.Command{
		Use:   "arrive <barrier-name> [holder]",
		Short: "Signal arrival at a barrier",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			barrierName := args[0]
			ctx := context.Background()

			// Create SDK client
			client := konductor.NewFromClient(k8sClient, namespace)

			// Build options
			var opts []konductor.Option
			if len(args) > 1 {
				opts = append(opts, konductor.WithHolder(args[1]))
			} else if holder != "" {
				opts = append(opts, konductor.WithHolder(holder))
			}

			// Arrive at barrier using SDK
			if err := barrier.Arrive(client, ctx, barrierName, opts...); err != nil {
				return err
			}

			// Wait for controller to process if requested
			if waitForUpdate {
				barrierObj := &syncv1.Barrier{}
				barrierObj.Name = barrierName
				barrierObj.Namespace = client.Namespace()

				if err := client.WaitForCondition(ctx, barrierObj, func(obj interface{}) bool {
					b := obj.(*syncv1.Barrier)
					return b.Status.Arrived > 0 || b.Status.Phase == syncv1.BarrierPhaseOpen
				}, &konductor.WaitConfig{Timeout: 5 * time.Second}); err != nil {
					logger.Warn("Failed to wait for condition", zap.Error(err))
				}
			}

			logger.Info("Signaled arrival at barrier", zap.String("barrier", barrierName))

			// Get barrier status to display info
			barrierObj, err := barrier.Get(client, ctx, barrierName)
			if err != nil {
				return err
			}
			logger.Info("Barrier status",
				zap.Int32("arrived", barrierObj.Status.Arrived),
				zap.Int32("expected", barrierObj.Spec.Expected),
				zap.String("phase", string(barrierObj.Status.Phase)),
			)

			return nil
		},
	}

	cmd.Flags().StringVar(&holder, "holder", "", "Arrival holder identifier (defaults to hostname)")
	cmd.Flags().BoolVar(&waitForUpdate, "wait-for-update", false, "Wait for controller to process the change")

	return cmd
}

func newBarrierListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all barriers",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Create SDK client
			client := konductor.NewFromClient(k8sClient, namespace)

			// List barriers using SDK
			barriers, err := barrier.List(client, ctx)
			if err != nil {
				return err
			}

			if len(barriers) == 0 {
				logger.Info("No barriers found")
				return nil
			}

			for _, b := range barriers {
				opened := "N/A"
				if b.Status.OpenedAt != nil {
					opened = b.Status.OpenedAt.Format("15:04:05")
				}

				logger.Info("Barrier",
					zap.String("name", b.Name),
					zap.Int32("expected", b.Spec.Expected),
					zap.Int32("arrived", b.Status.Arrived),
					zap.String("phase", string(b.Status.Phase)),
					zap.String("opened", opened),
				)
			}

			return nil
		},
	}

	return cmd
}

func newBarrierCreateCmd() *cobra.Command {
	var (
		expected int32
		timeout  time.Duration
		quorum   int32
	)

	cmd := &cobra.Command{
		Use:   "create <barrier-name>",
		Short: "Create a barrier",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			barrierName := args[0]
			ctx := context.Background()

			client := konductor.NewFromClient(k8sClient, namespace)

			var opts []konductor.Option
			if timeout > 0 {
				opts = append(opts, konductor.WithTimeout(timeout))
			}
			if quorum > 0 {
				opts = append(opts, konductor.WithQuorum(quorum))
			}

			if err := barrier.Create(client, ctx, barrierName, expected, opts...); err != nil {
				return err
			}

			logger.Info("Created barrier",
				zap.String("barrier", barrierName),
				zap.Int32("expected", expected),
			)
			return nil
		},
	}

	cmd.Flags().Int32Var(&expected, "expected", 1, "Expected number of arrivals")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Timeout for barrier")
	cmd.Flags().Int32Var(&quorum, "quorum", 0, "Minimum arrivals to open")

	return cmd
}

func newBarrierDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <barrier-name>",
		Short: "Delete a barrier",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			barrierName := args[0]
			ctx := context.Background()

			client := konductor.NewFromClient(k8sClient, namespace)

			if err := barrier.Delete(client, ctx, barrierName); err != nil {
				return err
			}

			logger.Info("Deleted barrier", zap.String("barrier", barrierName))
			return nil
		},
	}

	return cmd
}
