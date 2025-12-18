package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	konductor "github.com/LogicIQ/konductor/sdk/go/client"
	"github.com/LogicIQ/konductor/sdk/go/semaphore"
)

func newSemaphoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "semaphore",
		Short: "Manage semaphores",
		Long:  "Acquire, release, and manage semaphore permits",
	}

	cmd.AddCommand(newSemaphoreAcquireCmd())
	cmd.AddCommand(newSemaphoreReleaseCmd())
	cmd.AddCommand(newSemaphoreListCmd())

	return cmd
}

func newSemaphoreAcquireCmd() *cobra.Command {
	var (
		wait    bool
		timeout time.Duration
		ttl     time.Duration
		holder  string
	)

	cmd := &cobra.Command{
		Use:   "acquire <semaphore-name>",
		Short: "Acquire a semaphore permit",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			semaphoreName := args[0]
			ctx := context.Background()

			// Create SDK client
			client := konductor.NewFromClient(k8sClient, namespace)

			// Build options
			var opts []konductor.Option
			if holder != "" {
				opts = append(opts, konductor.WithHolder(holder))
			}
			if ttl > 0 {
				opts = append(opts, konductor.WithTTL(ttl))
			}
			if timeout > 0 {
				opts = append(opts, konductor.WithTimeout(timeout))
			}

			// Acquire semaphore using SDK
			permit, err := semaphore.AcquireSemaphore(client, ctx, semaphoreName, opts...)
			if err != nil {
				if !wait {
					return fmt.Errorf("failed to acquire semaphore: %w", err)
				}
				// For wait mode, we need to implement retry logic here
				return fmt.Errorf("failed to acquire semaphore: %w", err)
			}

			fmt.Printf("✓ Acquired permit for semaphore '%s' (holder: %s)\n", semaphoreName, permit.Holder())
			return nil
		},
	}

	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for permit to become available")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Timeout for waiting (e.g., 30s, 5m)")
	cmd.Flags().DurationVar(&ttl, "ttl", 10*time.Minute, "Time-to-live for the permit")
	cmd.Flags().StringVar(&holder, "holder", "", "Permit holder identifier (defaults to hostname)")

	return cmd
}

func newSemaphoreReleaseCmd() *cobra.Command {
	var holder string

	cmd := &cobra.Command{
		Use:   "release <semaphore-name>",
		Short: "Release a semaphore permit",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			semaphoreName := args[0]
			ctx := context.Background()

			if holder == "" {
				holder = os.Getenv("HOSTNAME")
				if holder == "" {
					return fmt.Errorf("holder must be specified or HOSTNAME must be set")
				}
			}

			// Create SDK client
			client := konductor.NewFromClient(k8sClient, namespace)

			// Release permit using SDK
			if err := client.ReleaseSemaphorePermit(ctx, semaphoreName, holder); err != nil {
				return fmt.Errorf("failed to release permit: %w", err)
			}

			fmt.Printf("✓ Released permit for semaphore '%s' (holder: %s)\n", semaphoreName, holder)
			return nil
		},
	}

	cmd.Flags().StringVar(&holder, "holder", "", "Permit holder identifier (defaults to hostname)")

	return cmd
}

func newSemaphoreListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all semaphores",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Create SDK client
			client := konductor.NewFromClient(k8sClient, namespace)

			// List semaphores using SDK
			semaphores, err := semaphore.ListSemaphores(client, ctx)
			if err != nil {
				return fmt.Errorf("failed to list semaphores: %w", err)
			}

			if len(semaphores) == 0 {
				fmt.Println("No semaphores found")
				return nil
			}

			fmt.Printf("%-20s %-10s %-10s %-10s %-10s\n", "NAME", "PERMITS", "IN-USE", "AVAILABLE", "PHASE")
			for _, sem := range semaphores {
				fmt.Printf("%-20s %-10d %-10d %-10d %-10s\n",
					sem.Name,
					sem.Spec.Permits,
					sem.Status.InUse,
					sem.Status.Available,
					sem.Status.Phase,
				)
			}

			return nil
		},
	}

	return cmd
}