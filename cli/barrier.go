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
			barrierObj, _ := barrier.Get(client, ctx, barrierName)
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
	var (
		holder string
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
				return fmt.Errorf("failed to arrive at barrier: %w", err)
			}

			// Wait for controller to process if requested
			if waitForUpdate {
				time.Sleep(3 * time.Second)
			}

			fmt.Printf("✓ Signaled arrival at barrier '%s'\n", barrierName)

			// Get barrier status to display info
			barrierObj, err := barrier.Get(client, ctx, barrierName)
			if err == nil {
				fmt.Printf("📊 Barrier status: %d/%d arrived, phase: %s\n", 
					barrierObj.Status.Arrived, barrierObj.Spec.Expected, barrierObj.Status.Phase)
			}

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

			if err := barrier.Create(client, ctx, barrierName, expected); err != nil {
				return fmt.Errorf("failed to create barrier: %w", err)
			}

			fmt.Printf("✓ Created barrier '%s' expecting %d arrivals\n", barrierName, expected)
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
				return fmt.Errorf("failed to delete barrier: %w", err)
			}

			fmt.Printf("✓ Deleted barrier '%s'\n", barrierName)
			return nil
		},
	}

	return cmd
}