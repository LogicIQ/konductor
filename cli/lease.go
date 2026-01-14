package main

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	konductor "github.com/LogicIQ/konductor/sdk/go/client"
	"github.com/LogicIQ/konductor/sdk/go/lease"
)

func newLeaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lease",
		Short: "Manage leases",
		Long:  "Acquire, release, and manage leases for singleton execution",
	}

	cmd.AddCommand(newLeaseCreateCmd())
	cmd.AddCommand(newLeaseDeleteCmd())
	cmd.AddCommand(newLeaseAcquireCmd())
	cmd.AddCommand(newLeaseReleaseCmd())
	cmd.AddCommand(newLeaseListCmd())

	return cmd
}

func createLeaseClient() *konductor.Client {
	return konductor.NewFromClient(k8sClient, namespace)
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

			client := createLeaseClient()

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
			leaseObj, err := lease.Acquire(client, ctx, leaseName, opts...)
			if err != nil {
				return err
			}

			logger.Info("Acquired lease", zap.String("lease", leaseName), zap.String("holder", leaseObj.Holder()))
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
					return errors.New("holder must be specified or HOSTNAME must be set")
				}
			}

			client := createLeaseClient()

			// Release lease using SDK
			if err := client.ReleaseLease(ctx, leaseName, holder); err != nil {
				return err
			}

			logger.Info("Released lease", zap.String("lease", leaseName), zap.String("holder", holder))
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

			client := createLeaseClient()

			// List leases using SDK
			leases, err := lease.List(client, ctx)
			if err != nil {
				return err
			}

			if len(leases) == 0 {
				logger.Info("No leases found")
				return nil
			}

			for _, l := range leases {
				holder := l.Status.Holder
				if holder == "" {
					holder = "N/A"
				}

				acquired := "N/A"
				if l.Status.AcquiredAt != nil {
					acquired = l.Status.AcquiredAt.Format("15:04:05")
				}

				logger.Info("Lease",
					zap.String("name", l.Name),
					zap.String("holder", holder),
					zap.String("phase", string(l.Status.Phase)),
					zap.String("acquired", acquired),
					zap.Int32("renewals", l.Status.RenewCount),
				)
			}

			return nil
		},
	}

	return cmd
}

func newLeaseCreateCmd() *cobra.Command {
	var ttl time.Duration

	cmd := &cobra.Command{
		Use:   "create <lease-name>",
		Short: "Create a lease",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			leaseName := args[0]
			ctx := context.Background()

			client := createLeaseClient()

			var opts []konductor.Option
			if ttl > 0 {
				opts = append(opts, konductor.WithTTL(ttl))
			}
			if err := lease.Create(client, ctx, leaseName, opts...); err != nil {
				return err
			}

			logger.Info("Created lease", zap.String("lease", leaseName))
			return nil
		},
	}

	cmd.Flags().DurationVar(&ttl, "ttl", 10*time.Minute, "Default TTL for lease")

	return cmd
}

func newLeaseDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <lease-name>",
		Short: "Delete a lease",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			leaseName := args[0]
			ctx := context.Background()

			client := createLeaseClient()

			if err := lease.Delete(client, ctx, leaseName); err != nil {
				return err
			}

			logger.Info("Deleted lease", zap.String("lease", leaseName))
			return nil
		},
	}

	return cmd
}
