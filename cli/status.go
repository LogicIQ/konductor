package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
	konductor "github.com/LogicIQ/konductor/sdk/go/client"
	"github.com/LogicIQ/konductor/sdk/go/semaphore"
	"github.com/LogicIQ/konductor/sdk/go/barrier"
	"github.com/LogicIQ/konductor/sdk/go/lease"
	"github.com/LogicIQ/konductor/sdk/go/gate"
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

			// Create SDK client
			client := konductor.NewFromClient(k8sClient, namespace)

			// Get semaphore using SDK
			sem, err := semaphore.GetSemaphore(client, ctx, name)
			if err != nil {
				return fmt.Errorf("failed to get semaphore: %w", err)
			}

			fmt.Printf("🚦 Semaphore: %s\n", sem.Name)
			fmt.Printf("   Namespace: %s\n", sem.Namespace)
			fmt.Printf("   Permits: %d total, %d in use, %d available\n", 
				sem.Spec.Permits, sem.Status.InUse, sem.Status.Available)
			fmt.Printf("   Phase: %s\n", sem.Status.Phase)


			// List permits using SDK
			if permits, err := client.ListPermits(ctx, name); err == nil {
				if len(permits) > 0 {
					fmt.Println("\n📋 Active Permits:")
					for _, permit := range permits {
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

			// Create SDK client
			client := konductor.NewFromClient(k8sClient, namespace)

			// Get barrier using SDK
			bar, err := barrier.GetBarrier(client, ctx, name)
			if err != nil {
				return fmt.Errorf("failed to get barrier: %w", err)
			}

			fmt.Printf("🚧 Barrier: %s\n", bar.Name)
			fmt.Printf("   Namespace: %s\n", bar.Namespace)
			fmt.Printf("   Expected: %d arrivals\n", bar.Spec.Expected)
			fmt.Printf("   Arrived: %d\n", bar.Status.Arrived)
			fmt.Printf("   Phase: %s\n", bar.Status.Phase)

			if bar.Spec.Quorum != nil {
				fmt.Printf("   Quorum: %d (minimum to open)\n", *bar.Spec.Quorum)
			}

			if bar.Status.OpenedAt != nil {
				fmt.Printf("   Opened: %s\n", bar.Status.OpenedAt.Format("2006-01-02 15:04:05"))
			}

			if len(bar.Status.Arrivals) > 0 {
				fmt.Println("\n📋 Arrivals:")
				for _, arrival := range bar.Status.Arrivals {
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

			// Create SDK client
			client := konductor.NewFromClient(k8sClient, namespace)

			// Get lease using SDK
			l, err := lease.GetLease(client, ctx, name)
			if err != nil {
				return fmt.Errorf("failed to get lease: %w", err)
			}

			fmt.Printf("🔒 Lease: %s\n", l.Name)
			fmt.Printf("   Namespace: %s\n", l.Namespace)
			fmt.Printf("   TTL: %s\n", l.Spec.TTL.Duration)
			fmt.Printf("   Phase: %s\n", l.Status.Phase)

			if l.Status.Holder != "" {
				fmt.Printf("   Holder: %s\n", l.Status.Holder)
				if l.Status.AcquiredAt != nil {
					fmt.Printf("   Acquired: %s\n", l.Status.AcquiredAt.Format("2006-01-02 15:04:05"))
				}
				if l.Status.ExpiresAt != nil {
					fmt.Printf("   Expires: %s\n", l.Status.ExpiresAt.Format("2006-01-02 15:04:05"))
				}
				fmt.Printf("   Renewals: %d\n", l.Status.RenewCount)
			}


			// List lease requests using SDK
			if requests, err := client.ListLeaseRequests(ctx, name); err == nil {
				pendingRequests := []syncv1.LeaseRequest{}
				for _, req := range requests {
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

			// Create SDK client
			client := konductor.NewFromClient(k8sClient, namespace)

			// Get gate using SDK
			g, err := gate.GetGate(client, ctx, name)
			if err != nil {
				return fmt.Errorf("failed to get gate: %w", err)
			}

			fmt.Printf("🚪 Gate: %s\n", g.Name)
			fmt.Printf("   Namespace: %s\n", g.Namespace)
			fmt.Printf("   Phase: %s\n", g.Status.Phase)

			if g.Status.OpenedAt != nil {
				fmt.Printf("   Opened: %s\n", g.Status.OpenedAt.Format("2006-01-02 15:04:05"))
			}

			fmt.Println("\n📋 Conditions:")
			for i, condition := range g.Spec.Conditions {
				status := "❌ Not Met"
				message := "Checking..."

				if i < len(g.Status.ConditionStatuses) {
					condStatus := g.Status.ConditionStatuses[i]
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

			// Create SDK client
			client := konductor.NewFromClient(k8sClient, namespace)

			fmt.Println("🎯 Konductor Status Overview")
			fmt.Println("============================")

			// List semaphores using SDK
			if semaphores, err := semaphore.ListSemaphores(client, ctx); err == nil {
				fmt.Printf("\n🚦 Semaphores (%d):\n", len(semaphores))
				if len(semaphores) == 0 {
					fmt.Println("   None found")
				} else {
					for _, sem := range semaphores {
						fmt.Printf("   • %s: %d/%d permits, %s\n", 
							sem.Name, sem.Status.InUse, sem.Spec.Permits, sem.Status.Phase)
					}
				}
			}


			// List barriers using SDK
			if barriers, err := barrier.ListBarriers(client, ctx); err == nil {
				fmt.Printf("\n🚧 Barriers (%d):\n", len(barriers))
				if len(barriers) == 0 {
					fmt.Println("   None found")
				} else {
					for _, b := range barriers {
						fmt.Printf("   • %s: %d/%d arrived, %s\n", 
							b.Name, b.Status.Arrived, b.Spec.Expected, b.Status.Phase)
					}
				}
			}


			// List leases using SDK
			if leases, err := lease.ListLeases(client, ctx); err == nil {
				fmt.Printf("\n🔒 Leases (%d):\n", len(leases))
				if len(leases) == 0 {
					fmt.Println("   None found")
				} else {
					for _, l := range leases {
						holder := "Available"
						if l.Status.Holder != "" {
							holder = l.Status.Holder
						}
						fmt.Printf("   • %s: %s, %s\n", l.Name, holder, l.Status.Phase)
					}
				}
			}


			// List gates using SDK
			if gates, err := gate.ListGates(client, ctx); err == nil {
				fmt.Printf("\n🚪 Gates (%d):\n", len(gates))
				if len(gates) == 0 {
					fmt.Println("   None found")
				} else {
					for _, g := range gates {
						metCount := 0
						for _, status := range g.Status.ConditionStatuses {
							if status.Met {
								metCount++
							}
						}
						fmt.Printf("   • %s: %d/%d conditions met, %s\n", 
							g.Name, metCount, len(g.Spec.Conditions), g.Status.Phase)
					}
				}
			}

			return nil
		},
	}

	return cmd
}