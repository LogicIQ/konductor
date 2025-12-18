package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
	konductor "github.com/LogicIQ/konductor/sdk/go/client"
	"github.com/LogicIQ/konductor/sdk/go/gate"
)

func newGateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gate",
		Short: "Manage gates",
		Long:  "Wait for gates and check dependency conditions",
	}

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
			if err := gate.WaitGate(client, ctx, gateName, opts...); err != nil {
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
			gates, err := gate.ListGates(client, ctx)
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

func printGateConditions(gate syncv1.Gate) {
	fmt.Println("📋 Condition Status:")
	for _, status := range gate.Status.ConditionStatuses {
		icon := "❌"
		if status.Met {
			icon = "✅"
		}
		fmt.Printf("  %s %s/%s: %s\n", icon, status.Type, status.Name, status.Message)
	}
}