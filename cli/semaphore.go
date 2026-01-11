package main

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
	konductor "github.com/LogicIQ/konductor/sdk/go/client"
	"github.com/LogicIQ/konductor/sdk/go/semaphore"
)

func newSemaphoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "semaphore",
		Short: "Manage semaphores",
		Long:  "Acquire, release, and manage semaphore permits",
	}

	cmd.AddCommand(newSemaphoreCreateCmd())
	cmd.AddCommand(newSemaphoreDeleteCmd())
	cmd.AddCommand(newSemaphoreAcquireCmd())
	cmd.AddCommand(newSemaphoreReleaseCmd())
	cmd.AddCommand(newSemaphoreListCmd())

	return cmd
}

func newSemaphoreAcquireCmd() *cobra.Command {
	var (
		wait          bool
		timeout       time.Duration
		ttl           time.Duration
		holder        string
		waitForUpdate bool
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
			permit, err := semaphore.Acquire(client, ctx, semaphoreName, opts...)
			if err != nil {
				return err
			}

			// Wait for controller to process if requested
			if waitForUpdate {
				time.Sleep(3 * time.Second)
			}

			logger.Info("Acquired permit for semaphore", zap.String("semaphore", semaphoreName), zap.String("holder", permit.Holder()))
			return nil
		},
	}

	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for permit to become available")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Timeout for waiting (e.g., 30s, 5m)")
	cmd.Flags().DurationVar(&ttl, "ttl", 10*time.Minute, "Time-to-live for the permit")
	cmd.Flags().StringVar(&holder, "holder", "", "Permit holder identifier (defaults to hostname)")
	cmd.Flags().BoolVar(&waitForUpdate, "wait-for-update", false, "Wait for controller to process the change")

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
					return errors.New("holder must be specified or HOSTNAME must be set")
				}
			}

			// Create SDK client
			client := konductor.NewFromClient(k8sClient, namespace)

			// Find and release permit by holder
			permits, err := client.ListPermits(ctx, semaphoreName)
			if err != nil {
				return err
			}

			var permitToDelete *syncv1.Permit
			for i := range permits {
				if permits[i].Spec.Holder == holder {
					permitToDelete = &permits[i]
					break
				}
			}

			if permitToDelete == nil {
				logger.Error("No permit found", zap.String("holder", holder))
				return errors.New("no permit found for holder")
			}

			if err := client.K8sClient().Delete(ctx, permitToDelete); err != nil {
				logger.Error("Failed to delete permit", zap.Error(err))
				return err
			}

			logger.Info("Released permit for semaphore", zap.String("semaphore", semaphoreName), zap.String("holder", holder))
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
			semaphores, err := semaphore.List(client, ctx)
			if err != nil {
				return err
			}

			if len(semaphores) == 0 {
				logger.Info("No semaphores found")
				return nil
			}

			for _, sem := range semaphores {
				logger.Info("Semaphore",
					zap.String("name", sem.Name),
					zap.Int32("permits", sem.Spec.Permits),
					zap.Int32("in-use", sem.Status.InUse),
					zap.Int32("available", sem.Status.Available),
					zap.String("phase", string(sem.Status.Phase)),
				)
			}

			return nil
		},
	}

	return cmd
}

func newSemaphoreCreateCmd() *cobra.Command {
	var (
		permits int32
		ttl     time.Duration
	)

	cmd := &cobra.Command{
		Use:   "create <semaphore-name>",
		Short: "Create a semaphore",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			semaphoreName := args[0]
			ctx := context.Background()

			client := konductor.NewFromClient(k8sClient, namespace)

			var opts []konductor.Option
			if ttl > 0 {
				opts = append(opts, konductor.WithTTL(ttl))
			}
			if err := semaphore.Create(client, ctx, semaphoreName, permits, opts...); err != nil {
				return err
			}

			logger.Info("Created semaphore", zap.String("semaphore", semaphoreName), zap.Int32("permits", permits))
			return nil
		},
	}

	cmd.Flags().Int32Var(&permits, "permits", 1, "Number of permits")
	cmd.Flags().DurationVar(&ttl, "ttl", 0, "Default TTL for permits")

	return cmd
}

func newSemaphoreDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <semaphore-name>",
		Short: "Delete a semaphore",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			semaphoreName := args[0]
			ctx := context.Background()

			client := konductor.NewFromClient(k8sClient, namespace)

			if err := semaphore.Delete(client, ctx, semaphoreName); err != nil {
				return err
			}

			logger.Info("Deleted semaphore", zap.String("semaphore", semaphoreName))
			return nil
		},
	}

	return cmd
}
