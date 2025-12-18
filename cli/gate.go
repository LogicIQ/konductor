package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
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

			startTime := time.Now()
			for {
				var gate syncv1.Gate
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      gateName,
					Namespace: namespace,
				}, &gate); err != nil {
					return fmt.Errorf("failed to get gate: %w", err)
				}

				switch gate.Status.Phase {
				case syncv1.GatePhaseOpen:
					fmt.Printf("✓ Gate '%s' is open! All conditions met.\n", gateName)
					return nil
				case syncv1.GatePhaseFailed:
					fmt.Printf("❌ Gate '%s' failed (timeout or error)\n", gateName)
					printGateConditions(gate)
					return fmt.Errorf("gate '%s' failed", gateName)
				case syncv1.GatePhaseWaiting:
					if timeout > 0 && time.Since(startTime) > timeout {
						fmt.Printf("⏰ Timeout waiting for gate '%s'\n", gateName)
						printGateConditions(gate)
						return fmt.Errorf("timeout waiting for gate '%s'", gateName)
					}
					
					fmt.Printf("⏳ Waiting for gate '%s'...\n", gateName)
					printGateConditions(gate)
					time.Sleep(10 * time.Second)
				}
			}
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

			var gates syncv1.GateList
			if err := k8sClient.List(ctx, &gates, client.InNamespace(namespace)); err != nil {
				return fmt.Errorf("failed to list gates: %w", err)
			}

			if len(gates.Items) == 0 {
				fmt.Println("No gates found")
				return nil
			}

			fmt.Printf("%-20s %-10s %-10s %-15s\n", "NAME", "CONDITIONS", "PHASE", "OPENED")
			for _, gate := range gates.Items {
				opened := "N/A"
				if gate.Status.OpenedAt != nil {
					opened = gate.Status.OpenedAt.Format("15:04:05")
				}

				conditionCount := len(gate.Spec.Conditions)
				metCount := 0
				for _, status := range gate.Status.ConditionStatuses {
					if status.Met {
						metCount++
					}
				}

				fmt.Printf("%-20s %-10s %-10s %-15s\n",
					gate.Name,
					fmt.Sprintf("%d/%d", metCount, conditionCount),
					gate.Status.Phase,
					opened,
				)

				// Show condition details
				for _, status := range gate.Status.ConditionStatuses {
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