package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
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


			if holder == "" {
				holder = os.Getenv("HOSTNAME")
				if holder == "" {
					holder = fmt.Sprintf("koncli-%d", time.Now().Unix())
				}
			}


			var semaphore syncv1.Semaphore
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      semaphoreName,
				Namespace: namespace,
			}, &semaphore); err != nil {
				return fmt.Errorf("failed to get semaphore: %w", err)
			}

			startTime := time.Now()
			for {

				if semaphore.Status.Available > 0 {

					permit := &syncv1.Permit{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("%s-%s", semaphoreName, holder),
							Namespace: namespace,
							Labels: map[string]string{
								"semaphore": semaphoreName,
							},
						},
						Spec: syncv1.PermitSpec{
							Semaphore: semaphoreName,
							Holder:    holder,
						},
					}

					if ttl > 0 {
						permit.Spec.TTL = &metav1.Duration{Duration: ttl}
					}

					if err := k8sClient.Create(ctx, permit); err != nil {
						return fmt.Errorf("failed to create permit: %w", err)
					}

					fmt.Printf("✓ Acquired permit for semaphore '%s' (holder: %s)\n", semaphoreName, holder)
					return nil
				}

				if !wait {
					return fmt.Errorf("no permits available for semaphore '%s'", semaphoreName)
				}

				if timeout > 0 && time.Since(startTime) > timeout {
					return fmt.Errorf("timeout waiting for semaphore '%s'", semaphoreName)
				}

				fmt.Printf("⏳ Waiting for permit (available: %d/%d)...\n", 
					semaphore.Status.Available, semaphore.Spec.Permits)
				time.Sleep(5 * time.Second)


				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      semaphoreName,
					Namespace: namespace,
				}, &semaphore); err != nil {
					return fmt.Errorf("failed to refresh semaphore: %w", err)
				}
			}
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

			permitName := fmt.Sprintf("%s-%s", semaphoreName, holder)
			permit := &syncv1.Permit{
				ObjectMeta: metav1.ObjectMeta{
					Name:      permitName,
					Namespace: namespace,
				},
			}

			if err := k8sClient.Delete(ctx, permit); err != nil {
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

			var semaphores syncv1.SemaphoreList
			if err := k8sClient.List(ctx, &semaphores, client.InNamespace(namespace)); err != nil {
				return fmt.Errorf("failed to list semaphores: %w", err)
			}

			if len(semaphores.Items) == 0 {
				fmt.Println("No semaphores found")
				return nil
			}

			fmt.Printf("%-20s %-10s %-10s %-10s %-10s\n", "NAME", "PERMITS", "IN-USE", "AVAILABLE", "PHASE")
			for _, sem := range semaphores.Items {
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