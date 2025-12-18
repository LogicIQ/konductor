package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

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

func newGateWaitCmd() *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "wait <gate-name>",
		Short: "Wait for a gate to open",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gateName := args[0]
			ctx := context.Background()

			// Create SDK client
			client := konductor.NewFromClient(k8sClient, namespace)

			// Build options
			var opts []konductor.Option
			if timeout > 0 {
				opts = append(opts, konductor.WithTimeout(timeout))
			}

			// Wait for gate using SDK
			if err := gate.Wait(client, ctx, gateName, opts...); err != nil {
				return err
			}

			fmt.Printf("✓ Gate '%s' is open! All conditions met.\n", gateName)
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

			// Create SDK client
			client := konductor.NewFromClient(k8sClient, namespace)

			// List gates using SDK
			gates, err := gate.List(client, ctx)
			if err != nil {
				return fmt.Errorf("failed to list gates: %w", err)
			}

			if len(gates) == 0 {
				fmt.Println("No gates found")
				return nil
			}

			fmt.Printf("%-20s %-10s %-10s %-15s\n", "NAME", "CONDITIONS", "PHASE", "OPENED")
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

				fmt.Printf("%-20s %-10s %-10s %-15s\n",
					g.Name,
					fmt.Sprintf("%d/%d", metCount, conditionCount),
					g.Status.Phase,
					opened,
				)

				// Show condition details
				for _, status := range g.Status.ConditionStatuses {
					icon := "❌"
					if status.Met {
						icon = "✅"
					}
					fmt.Printf("  %s %s/%s: %s\n", icon, status.Type, status.Name, status.Message)
				}
				fmt.Println()
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

			client := konductor.NewFromClient(k8sClient, namespace)

			if err := gate.Create(client, ctx, gateName); err != nil {
				return fmt.Errorf("failed to create gate: %w", err)
			}

			fmt.Printf("✓ Created gate '%s'\n", gateName)
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

			client := konductor.NewFromClient(k8sClient, namespace)

			if err := gate.Delete(client, ctx, gateName); err != nil {
				return fmt.Errorf("failed to delete gate: %w", err)
			}

			fmt.Printf("✓ Deleted gate '%s'\n", gateName)
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

			client := konductor.NewFromClient(k8sClient, namespace)

			if err := gate.Open(client, ctx, gateName); err != nil {
				return fmt.Errorf("failed to open gate: %w", err)
			}

			fmt.Printf("✓ Opened gate '%s'\n", gateName)
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

			client := konductor.NewFromClient(k8sClient, namespace)

			if err := gate.Close(client, ctx, gateName); err != nil {
				return fmt.Errorf("failed to close gate: %w", err)
			}

			fmt.Printf("✓ Closed gate '%s'\n", gateName)
			return nil
		},
	}

	return cmd
}