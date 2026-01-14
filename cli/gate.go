package main

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	konductor "github.com/LogicIQ/konductor/sdk/go/client"
	"github.com/LogicIQ/konductor/sdk/go/gate"
)

func newGateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gate",
		Short: "Manage gates",
		Long:  "Wait for gates and check dependency conditions",
	}

	cmd.AddCommand(newGateCreateCmd())
	cmd.AddCommand(newGateDeleteCmd())
	cmd.AddCommand(newGateOpenCmd())
	cmd.AddCommand(newGateCloseCmd())
	cmd.AddCommand(newGateWaitCmd())
	cmd.AddCommand(newGateListCmd())

	return cmd
}

func createGateClient() *konductor.Client {
	return konductor.NewFromClient(k8sClient, namespace)
}

func newGateWaitCmd() *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "wait <gate-name>",
		Short: "Wait for a gate to open",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gateName := args[0]
			ctx := context.Background()

			client := createGateClient()

			// Build options
			var opts []konductor.Option
			if timeout > 0 {
				opts = append(opts, konductor.WithTimeout(timeout))
			}

			// Wait for gate using SDK
			if err := gate.Wait(client, ctx, gateName, opts...); err != nil {
				return err
			}

			logger.Info("Gate is open", zap.String("gate", gateName))
			return nil
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Timeout for waiting (e.g., 30s, 5m)")

	return cmd
}

func newGateListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all gates",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			client := createGateClient()

			// List gates using SDK
			gates, err := gate.List(client, ctx)
			if err != nil {
				return err
			}

			if len(gates) == 0 {
				logger.Info("No gates found")
				return nil
			}

			for _, g := range gates {
				opened := "N/A"
				if g.Status.OpenedAt != nil {
					opened = g.Status.OpenedAt.Format("15:04:05")
				}

				conditionCount := len(g.Spec.Conditions)
				metCount := 0
				for _, status := range g.Status.ConditionStatuses {
					if status.Met {
						metCount++
					}
				}

				logger.Info("Gate",
					zap.String("name", g.Name),
					zap.Int("conditions_met", metCount),
					zap.Int("conditions_total", conditionCount),
					zap.String("phase", string(g.Status.Phase)),
					zap.String("opened", opened),
				)

				// Show condition details
				for _, status := range g.Status.ConditionStatuses {
					logger.Info("Condition",
						zap.Bool("met", status.Met),
						zap.String("type", status.Type),
						zap.String("name", status.Name),
						zap.String("message", status.Message),
					)
				}
			}

			return nil
		},
	}

	return cmd
}

func newGateCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <gate-name>",
		Short: "Create a gate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gateName := args[0]
			ctx := context.Background()

			client := createGateClient()

			if err := gate.Create(client, ctx, gateName); err != nil {
				return err
			}

			logger.Info("Created gate", zap.String("gate", gateName))
			return nil
		},
	}

	return cmd
}

func newGateDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <gate-name>",
		Short: "Delete a gate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gateName := args[0]
			ctx := context.Background()

			client := createGateClient()

			if err := gate.Delete(client, ctx, gateName); err != nil {
				return err
			}

			logger.Info("Deleted gate", zap.String("gate", gateName))
			return nil
		},
	}

	return cmd
}

func newGateOpenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open <gate-name>",
		Short: "Open a gate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gateName := args[0]
			ctx := context.Background()

			client := createGateClient()

			if err := gate.Open(client, ctx, gateName); err != nil {
				return err
			}

			logger.Info("Opened gate", zap.String("gate", gateName))
			return nil
		},
	}

	return cmd
}

func newGateCloseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close <gate-name>",
		Short: "Close a gate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gateName := args[0]
			ctx := context.Background()

			client := createGateClient()

			if err := gate.Close(client, ctx, gateName); err != nil {
				return err
			}

			logger.Info("Closed gate", zap.String("gate", gateName))
			return nil
		},
	}

	return cmd
}
