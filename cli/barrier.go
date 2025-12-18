package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	konductor "github.com/LogicIQ/konductor/sdk/go/client"
	"github.com/LogicIQ/konductor/sdk/go/barrier"
)

func newBarrierCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "barrier",
		Short: "Manage barriers",
		Long:  "Wait for barriers and signal arrivals",
	}

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
			if err := barrier.WaitBarrier(client, ctx, barrierName, opts...); err != nil {
				return err
			}

			// Get barrier status to display info
			barrierObj, _ := barrier.GetBarrier(client, ctx, barrierName)
			if barrierObj != nil {
				fmt.Printf("✓ Barrier '%s' is open! (arrived: %d/%d)\n", 
					barrierName, barrierObj.Status.Arrived, barrierObj.Spec.Expected)
			}

			return nil
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Timeout for waiting (e.g., 30s, 5m)")

	return cmd
}

func newBarrierArriveCmd() *cobra.Command {
	var holder string

	cmd := &cobra.Command{
		Use:   "arrive <barrier-name>",
		Short: "Signal arrival at a barrier",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			barrierName := args[0]
			ctx := context.Background()

			// Create SDK client
			client := konductor.NewFromClient(k8sClient, namespace)

			// Build options
			var opts []konductor.Option
			if holder != "" {
				opts = append(opts, konductor.WithHolder(holder))
			}

			// Arrive at barrier using SDK
			if err := barrier.ArriveBarrier(client, ctx, barrierName, opts...); err != nil {
				return fmt.Errorf("failed to arrive at barrier: %w", err)
			}

			fmt.Printf("✓ Signaled arrival at barrier '%s'\n", barrierName)

			// Get barrier status to display info
			barrierObj, err := barrier.GetBarrier(client, ctx, barrierName)
			if err == nil {
				fmt.Printf("📊 Barrier status: %d/%d arrived, phase: %s\n", 
					barrierObj.Status.Arrived, barrierObj.Spec.Expected, barrierObj.Status.Phase)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&holder, "holder", "", "Arrival holder identifier (defaults to hostname)")

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
			barriers, err := barrier.ListBarriers(client, ctx)
			if err != nil {
				return fmt.Errorf("failed to list barriers: %w", err)
			}

			if len(barriers) == 0 {
				fmt.Println("No barriers found")
				return nil
			}

			fmt.Printf("%-20s %-10s %-10s %-10s %-15s\n", "NAME", "EXPECTED", "ARRIVED", "PHASE", "OPENED")
			for _, b := range barriers {
				opened := "N/A"
				if b.Status.OpenedAt != nil {
					opened = b.Status.OpenedAt.Format("15:04:05")
				}

				fmt.Printf("%-20s %-10d %-10d %-10s %-15s\n",
					b.Name,
					b.Spec.Expected,
					b.Status.Arrived,
					b.Status.Phase,
					opened,
				)
			}

			return nil
		},
	}

	return cmd
}