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

			if holder == "" {
				holder = os.Getenv("HOSTNAME")
				if holder == "" {
					holder = fmt.Sprintf("koncli-%d", time.Now().Unix())
				}
			}


			request := &syncv1.LeaseRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%s", leaseName, holder),
					Namespace: namespace,
					Labels: map[string]string{
						"lease": leaseName,
					},
				},
				Spec: syncv1.LeaseRequestSpec{
					Lease:  leaseName,
					Holder: holder,
				},
			}

			if priority > 0 {
				request.Spec.Priority = &priority
			}

			if err := k8sClient.Create(ctx, request); err != nil {
				return fmt.Errorf("failed to create lease request: %w", err)
			}

			startTime := time.Now()
			for {

				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      request.Name,
					Namespace: namespace,
				}, request); err != nil {
					return fmt.Errorf("failed to get lease request: %w", err)
				}

				switch request.Status.Phase {
				case syncv1.LeaseRequestPhaseGranted:
					fmt.Printf("✓ Acquired lease '%s' (holder: %s)\n", leaseName, holder)
					return nil
				case syncv1.LeaseRequestPhaseDenied:
					return fmt.Errorf("lease request denied for '%s'", leaseName)
				case syncv1.LeaseRequestPhasePending:
					if !wait {
						return fmt.Errorf("lease '%s' is not available", leaseName)
					}

					if timeout > 0 && time.Since(startTime) > timeout {
						return fmt.Errorf("timeout waiting for lease '%s'", leaseName)
					}

					fmt.Printf("⏳ Waiting for lease '%s'...\n", leaseName)
					time.Sleep(5 * time.Second)
				}
			}
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


			requestName := fmt.Sprintf("%s-%s", leaseName, holder)
			request := &syncv1.LeaseRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      requestName,
					Namespace: namespace,
				},
			}

			if err := k8sClient.Delete(ctx, request); err != nil {
				return fmt.Errorf("failed to delete lease request: %w", err)
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

			var leases syncv1.LeaseList
			if err := k8sClient.List(ctx, &leases, client.InNamespace(namespace)); err != nil {
				return fmt.Errorf("failed to list leases: %w", err)
			}

			if len(leases.Items) == 0 {
				fmt.Println("No leases found")
				return nil
			}

			fmt.Printf("%-20s %-20s %-10s %-15s %-10s\n", "NAME", "HOLDER", "PHASE", "ACQUIRED", "RENEWALS")
			for _, lease := range leases.Items {
				holder := lease.Status.Holder
				if holder == "" {
					holder = "N/A"
				}

				acquired := "N/A"
				if lease.Status.AcquiredAt != nil {
					acquired = lease.Status.AcquiredAt.Format("15:04:05")
				}

				fmt.Printf("%-20s %-20s %-10s %-15s %-10d\n",
					lease.Name,
					holder,
					lease.Status.Phase,
					acquired,
					lease.Status.RenewCount,
				)
			}

			return nil
		},
	}

	return cmd
}