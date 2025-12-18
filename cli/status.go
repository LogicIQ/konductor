package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of coordination primitives",
		Long:  "Display detailed status information for semaphores, barriers, leases, and gates",
	}

	cmd.AddCommand(newStatusSemaphoreCmd())
	cmd.AddCommand(newStatusBarrierCmd())
	cmd.AddCommand(newStatusLeaseCmd())
	cmd.AddCommand(newStatusGateCmd())
	cmd.AddCommand(newStatusAllCmd())

	return cmd
}

func newStatusSemaphoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "semaphore <name>",
		Short: "Show semaphore status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			var semaphore syncv1.Semaphore
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			}, &semaphore); err != nil {
				return fmt.Errorf("failed to get semaphore: %w", err)
			}

			fmt.Printf("🚦 Semaphore: %s\n", semaphore.Name)
			fmt.Printf("   Namespace: %s\n", semaphore.Namespace)
			fmt.Printf("   Permits: %d total, %d in use, %d available\n", 
				semaphore.Spec.Permits, semaphore.Status.InUse, semaphore.Status.Available)
			fmt.Printf("   Phase: %s\n", semaphore.Status.Phase)


			var permits syncv1.PermitList
			if err := k8sClient.List(ctx, &permits, client.InNamespace(namespace), 
				client.MatchingLabels{"semaphore": name}); err == nil {
				
				if len(permits.Items) > 0 {
					fmt.Println("\n📋 Active Permits:")
					for _, permit := range permits.Items {
						status := "Active"
						if permit.Status.ExpiresAt != nil {
							status = fmt.Sprintf("Expires: %s", permit.Status.ExpiresAt.Format("15:04:05"))
						}
						fmt.Printf("   • %s (%s)\n", permit.Spec.Holder, status)
					}
				}
			}

			return nil
		},
	}

	return cmd
}

func newStatusBarrierCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "barrier <name>",
		Short: "Show barrier status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			var barrier syncv1.Barrier
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			}, &barrier); err != nil {
				return fmt.Errorf("failed to get barrier: %w", err)
			}

			fmt.Printf("🚧 Barrier: %s\n", barrier.Name)
			fmt.Printf("   Namespace: %s\n", barrier.Namespace)
			fmt.Printf("   Expected: %d arrivals\n", barrier.Spec.Expected)
			fmt.Printf("   Arrived: %d\n", barrier.Status.Arrived)
			fmt.Printf("   Phase: %s\n", barrier.Status.Phase)

			if barrier.Spec.Quorum != nil {
				fmt.Printf("   Quorum: %d (minimum to open)\n", *barrier.Spec.Quorum)
			}

			if barrier.Status.OpenedAt != nil {
				fmt.Printf("   Opened: %s\n", barrier.Status.OpenedAt.Format("2006-01-02 15:04:05"))
			}

			if len(barrier.Status.Arrivals) > 0 {
				fmt.Println("\n📋 Arrivals:")
				for _, arrival := range barrier.Status.Arrivals {
					fmt.Printf("   • %s\n", arrival)
				}
			}

			return nil
		},
	}

	return cmd
}

func newStatusLeaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lease <name>",
		Short: "Show lease status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			var lease syncv1.Lease
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			}, &lease); err != nil {
				return fmt.Errorf("failed to get lease: %w", err)
			}

			fmt.Printf("🔒 Lease: %s\n", lease.Name)
			fmt.Printf("   Namespace: %s\n", lease.Namespace)
			fmt.Printf("   TTL: %s\n", lease.Spec.TTL.Duration)
			fmt.Printf("   Phase: %s\n", lease.Status.Phase)

			if lease.Status.Holder != "" {
				fmt.Printf("   Holder: %s\n", lease.Status.Holder)
				if lease.Status.AcquiredAt != nil {
					fmt.Printf("   Acquired: %s\n", lease.Status.AcquiredAt.Format("2006-01-02 15:04:05"))
				}
				if lease.Status.ExpiresAt != nil {
					fmt.Printf("   Expires: %s\n", lease.Status.ExpiresAt.Format("2006-01-02 15:04:05"))
				}
				fmt.Printf("   Renewals: %d\n", lease.Status.RenewCount)
			}


			var requests syncv1.LeaseRequestList
			if err := k8sClient.List(ctx, &requests, client.InNamespace(namespace), 
				client.MatchingLabels{"lease": name}); err == nil {
				
				pendingRequests := []syncv1.LeaseRequest{}
				for _, req := range requests.Items {
					if req.Status.Phase == syncv1.LeaseRequestPhasePending {
						pendingRequests = append(pendingRequests, req)
					}
				}

				if len(pendingRequests) > 0 {
					fmt.Println("\n📋 Pending Requests:")
					for _, req := range pendingRequests {
						priority := "0"
						if req.Spec.Priority != nil {
							priority = fmt.Sprintf("%d", *req.Spec.Priority)
						}
						fmt.Printf("   • %s (priority: %s)\n", req.Spec.Holder, priority)
					}
				}
			}

			return nil
		},
	}

	return cmd
}

func newStatusGateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gate <name>",
		Short: "Show gate status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			var gate syncv1.Gate
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			}, &gate); err != nil {
				return fmt.Errorf("failed to get gate: %w", err)
			}

			fmt.Printf("🚪 Gate: %s\n", gate.Name)
			fmt.Printf("   Namespace: %s\n", gate.Namespace)
			fmt.Printf("   Phase: %s\n", gate.Status.Phase)

			if gate.Status.OpenedAt != nil {
				fmt.Printf("   Opened: %s\n", gate.Status.OpenedAt.Format("2006-01-02 15:04:05"))
			}

			fmt.Println("\n📋 Conditions:")
			for i, condition := range gate.Spec.Conditions {
				status := "❌ Not Met"
				message := "Checking..."

				if i < len(gate.Status.ConditionStatuses) {
					condStatus := gate.Status.ConditionStatuses[i]
					if condStatus.Met {
						status = "✅ Met"
					} else {
						status = "❌ Not Met"
					}
					message = condStatus.Message
				}

				fmt.Printf("   %s %s/%s: %s\n", status, condition.Type, condition.Name, message)
				if condition.State != "" {
					fmt.Printf("      Required state: %s\n", condition.State)
				}
				if condition.Value != nil {
					fmt.Printf("      Required value: %d\n", *condition.Value)
				}
			}

			return nil
		},
	}

	return cmd
}

func newStatusAllCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all",
		Short: "Show status of all coordination primitives",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			fmt.Println("🎯 Konductor Status Overview")
			fmt.Println("============================")


			var semaphores syncv1.SemaphoreList
			if err := k8sClient.List(ctx, &semaphores, client.InNamespace(namespace)); err == nil {
				fmt.Printf("\n🚦 Semaphores (%d):\n", len(semaphores.Items))
				if len(semaphores.Items) == 0 {
					fmt.Println("   None found")
				} else {
					for _, sem := range semaphores.Items {
						fmt.Printf("   • %s: %d/%d permits, %s\n", 
							sem.Name, sem.Status.InUse, sem.Spec.Permits, sem.Status.Phase)
					}
				}
			}


			var barriers syncv1.BarrierList
			if err := k8sClient.List(ctx, &barriers, client.InNamespace(namespace)); err == nil {
				fmt.Printf("\n🚧 Barriers (%d):\n", len(barriers.Items))
				if len(barriers.Items) == 0 {
					fmt.Println("   None found")
				} else {
					for _, barrier := range barriers.Items {
						fmt.Printf("   • %s: %d/%d arrived, %s\n", 
							barrier.Name, barrier.Status.Arrived, barrier.Spec.Expected, barrier.Status.Phase)
					}
				}
			}


			var leases syncv1.LeaseList
			if err := k8sClient.List(ctx, &leases, client.InNamespace(namespace)); err == nil {
				fmt.Printf("\n🔒 Leases (%d):\n", len(leases.Items))
				if len(leases.Items) == 0 {
					fmt.Println("   None found")
				} else {
					for _, lease := range leases.Items {
						holder := "Available"
						if lease.Status.Holder != "" {
							holder = lease.Status.Holder
						}
						fmt.Printf("   • %s: %s, %s\n", lease.Name, holder, lease.Status.Phase)
					}
				}
			}


			var gates syncv1.GateList
			if err := k8sClient.List(ctx, &gates, client.InNamespace(namespace)); err == nil {
				fmt.Printf("\n🚪 Gates (%d):\n", len(gates.Items))
				if len(gates.Items) == 0 {
					fmt.Println("   None found")
				} else {
					for _, gate := range gates.Items {
						metCount := 0
						for _, status := range gate.Status.ConditionStatuses {
							if status.Met {
								metCount++
							}
						}
						fmt.Printf("   • %s: %d/%d conditions met, %s\n", 
							gate.Name, metCount, len(gate.Spec.Conditions), gate.Status.Phase)
					}
				}
			}

			return nil
		},
	}

	return cmd
}