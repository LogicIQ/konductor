package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	konductor "github.com/LogicIQ/konductor/sdk/go/client"
	"github.com/LogicIQ/konductor/sdk/go/lease"
)

func newLeaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lease",
		Short: "Manage leases",
		Long:  "Acquire, release, and manage leases for singleton execution",
	}

	cmd.AddCommand(newLeaseAcquireCmd())
	cmd.AddCommand(newLeaseReleaseCmd())
	cmd.AddCommand(newLeaseListCmd())

	return cmd
}

func newLeaseAcquireCmd() *cobra.Command {
	var (
		wait     bool
		timeout  time.Duration
		priority int32
		holder   string
	)

	cmd := &cobra.Command{
		Use:   "acquire <lease-name>",
		Short: "Acquire a lease",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			leaseName := args[0]
			ctx := context.Background()

			// Create SDK client
			client := konductor.NewFromClient(k8sClient, namespace)

			// Build options
			var opts []konductor.Option
			if holder != "" {
				opts = append(opts, konductor.WithHolder(holder))
			}
			if priority > 0 {
				opts = append(opts, konductor.WithPriority(priority))
			}
			if timeout > 0 {
				opts = append(opts, konductor.WithTimeout(timeout))
			}

			// Acquire lease using SDK
			leaseObj, err := lease.AcquireLease(client, ctx, leaseName, opts...)
			if err != nil {
				if !wait {
					return fmt.Errorf("failed to acquire lease: %w", err)
				}
				return fmt.Errorf("failed to acquire lease: %w", err)
			}

			fmt.Printf("✓ Acquired lease '%s' (holder: %s)\n", leaseName, leaseObj.Holder())
			return nil
		},
	}

	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for lease to become available")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Timeout for waiting (e.g., 30s, 5m)")
	cmd.Flags().Int32Var(&priority, "priority", 0, "Priority for lease acquisition (higher wins)")
	cmd.Flags().StringVar(&holder, "holder", "", "Lease holder identifier (defaults to hostname)")

	return cmd
}

func newLeaseReleaseCmd() *cobra.Command {
	var holder string

	cmd := &cobra.Command{
		Use:   "release <lease-name>",
		Short: "Release a lease",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			leaseName := args[0]
			ctx := context.Background()

			if holder == "" {
				holder = os.Getenv("HOSTNAME")
				if holder == "" {
					return fmt.Errorf("holder must be specified or HOSTNAME must be set")
				}
			}

			// Create SDK client
			client := konductor.NewFromClient(k8sClient, namespace)

			// Release lease using SDK
			if err := client.ReleaseLease(ctx, leaseName, holder); err != nil {
				return fmt.Errorf("failed to release lease: %w", err)
			}

			fmt.Printf("✓ Released lease '%s' (holder: %s)\n", leaseName, holder)
			return nil
		},
	}

	cmd.Flags().StringVar(&holder, "holder", "", "Lease holder identifier (defaults to hostname)")

	return cmd
}

func newLeaseListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all leases",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Create SDK client
			client := konductor.NewFromClient(k8sClient, namespace)

			// List leases using SDK
			leases, err := lease.ListLeases(client, ctx)
			if err != nil {
				return fmt.Errorf("failed to list leases: %w", err)
			}

			if len(leases) == 0 {
				fmt.Println("No leases found")
				return nil
			}

			fmt.Printf("%-20s %-20s %-10s %-15s %-10s\n", "NAME", "HOLDER", "PHASE", "ACQUIRED", "RENEWALS")
			for _, l := range leases {
				holder := l.Status.Holder
				if holder == "" {
					holder = "N/A"
				}

				acquired := "N/A"
				if l.Status.AcquiredAt != nil {
					acquired = l.Status.AcquiredAt.Format("15:04:05")
				}

				fmt.Printf("%-20s %-20s %-10s %-15s %-10d\n",
					l.Name,
					holder,
					l.Status.Phase,
					acquired,
					l.Status.RenewCount,
				)
			}

			return nil
		},
	}

	return cmd
}