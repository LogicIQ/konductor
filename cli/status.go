package main

import (
	"context"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
	"github.com/LogicIQ/konductor/sdk/go/barrier"
	konductor "github.com/LogicIQ/konductor/sdk/go/client"
	"github.com/LogicIQ/konductor/sdk/go/gate"
	"github.com/LogicIQ/konductor/sdk/go/lease"
	"github.com/LogicIQ/konductor/sdk/go/semaphore"
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
			sem, err := semaphore.Get(client, ctx, name)
			if err != nil {
				return err
			}

			logger.Info("Semaphore status",
				zap.String("name", sem.Name),
				zap.String("namespace", sem.Namespace),
				zap.Int32("permits_total", sem.Spec.Permits),
				zap.Int32("permits_in_use", sem.Status.InUse),
				zap.Int32("permits_available", sem.Status.Available),
				zap.String("phase", string(sem.Status.Phase)),
			)

			// List permits using SDK
			permits, err := client.ListPermits(ctx, name)
			if err != nil {
				logger.Warn("Failed to list permits", zap.Error(err))
			} else if len(permits) > 0 {
				for _, permit := range permits {
					expires := "Active"
					if permit.Status.ExpiresAt != nil {
						expires = permit.Status.ExpiresAt.Format("15:04:05")
					}
					logger.Info("Active permit",
						zap.String("holder", permit.Spec.Holder),
						zap.String("expires", expires),
					)
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
			bar, err := barrier.Get(client, ctx, name)
			if err != nil {
				return err
			}

			fields := []zap.Field{
				zap.String("name", bar.Name),
				zap.String("namespace", bar.Namespace),
				zap.Int32("expected", bar.Spec.Expected),
				zap.Int32("arrived", bar.Status.Arrived),
				zap.String("phase", string(bar.Status.Phase)),
			}

			if bar.Spec.Quorum != nil {
				fields = append(fields, zap.Int32("quorum", *bar.Spec.Quorum))
			}

			if bar.Status.OpenedAt != nil {
				fields = append(fields, zap.String("opened", bar.Status.OpenedAt.Format("2006-01-02 15:04:05")))
			}

			logger.Info("Barrier status", fields...)

			if len(bar.Status.Arrivals) > 0 {
				for _, arrival := range bar.Status.Arrivals {
					logger.Info("Arrival", zap.String("holder", arrival))
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
			l, err := lease.Get(client, ctx, name)
			if err != nil {
				return err
			}

			fields := []zap.Field{
				zap.String("name", l.Name),
				zap.String("namespace", l.Namespace),
				zap.Duration("ttl", l.Spec.TTL.Duration),
				zap.String("phase", string(l.Status.Phase)),
			}

			if l.Status.Holder != "" {
				fields = append(fields, zap.String("holder", l.Status.Holder))
				if l.Status.AcquiredAt != nil {
					fields = append(fields, zap.String("acquired", l.Status.AcquiredAt.Format("2006-01-02 15:04:05")))
				}
				if l.Status.ExpiresAt != nil {
					fields = append(fields, zap.String("expires", l.Status.ExpiresAt.Format("2006-01-02 15:04:05")))
				}
				fields = append(fields, zap.Int32("renewals", l.Status.RenewCount))
			}

			logger.Info("Lease status", fields...)

			// List lease requests using SDK
			requests, err := client.ListLeaseRequests(ctx, name)
			if err != nil {
				logger.Warn("Failed to list lease requests", zap.Error(err))
			} else {
				for _, req := range requests {
					if req.Status.Phase == syncv1.LeaseRequestPhasePending {
						priority := int32(0)
						if req.Spec.Priority != nil {
							priority = *req.Spec.Priority
						}
						logger.Info("Pending request",
							zap.String("holder", req.Spec.Holder),
							zap.Int32("priority", priority),
						)
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
			g, err := gate.Get(client, ctx, name)
			if err != nil {
				return err
			}

			fields := []zap.Field{
				zap.String("name", g.Name),
				zap.String("namespace", g.Namespace),
				zap.String("phase", string(g.Status.Phase)),
			}

			if g.Status.OpenedAt != nil {
				fields = append(fields, zap.String("opened", g.Status.OpenedAt.Format("2006-01-02 15:04:05")))
			}

			logger.Info("Gate status", fields...)

			for i, condition := range g.Spec.Conditions {
				met := false
				message := "Checking..."

				if i < len(g.Status.ConditionStatuses) {
					condStatus := g.Status.ConditionStatuses[i]
					met = condStatus.Met
					message = condStatus.Message
				}

				condFields := []zap.Field{
					zap.Bool("met", met),
					zap.String("type", condition.Type),
					zap.String("name", condition.Name),
					zap.String("message", message),
				}

				if condition.State != "" {
					condFields = append(condFields, zap.String("required_state", condition.State))
				}
				if condition.Value != nil {
					condFields = append(condFields, zap.Int32("required_value", *condition.Value))
				}

				logger.Info("Condition", condFields...)
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

			logger.Info("Konductor Status Overview")

			// List semaphores using SDK
			semaphores, err := semaphore.List(client, ctx)
			if err != nil {
				logger.Warn("Failed to list semaphores", zap.Error(err))
			} else {
				logger.Info("Semaphores", zap.Int("count", len(semaphores)))
				for _, sem := range semaphores {
					logger.Info("Semaphore",
						zap.String("name", sem.Name),
						zap.Int32("in_use", sem.Status.InUse),
						zap.Int32("total", sem.Spec.Permits),
						zap.String("phase", string(sem.Status.Phase)),
					)
				}
			}

			// List barriers using SDK
			barriers, err := barrier.List(client, ctx)
			if err != nil {
				logger.Warn("Failed to list barriers", zap.Error(err))
			} else {
				logger.Info("Barriers", zap.Int("count", len(barriers)))
				for _, b := range barriers {
					logger.Info("Barrier",
						zap.String("name", b.Name),
						zap.Int32("arrived", b.Status.Arrived),
						zap.Int32("expected", b.Spec.Expected),
						zap.String("phase", string(b.Status.Phase)),
					)
				}
			}

			// List leases using SDK
			leases, err := lease.List(client, ctx)
			if err != nil {
				logger.Warn("Failed to list leases", zap.Error(err))
			} else {
				logger.Info("Leases", zap.Int("count", len(leases)))
				for _, l := range leases {
					holder := "Available"
					if l.Status.Holder != "" {
						holder = l.Status.Holder
					}
					logger.Info("Lease",
						zap.String("name", l.Name),
						zap.String("holder", holder),
						zap.String("phase", string(l.Status.Phase)),
					)
				}
			}

			// List gates using SDK
			gates, err := gate.List(client, ctx)
			if err != nil {
				logger.Warn("Failed to list gates", zap.Error(err))
			} else {
				logger.Info("Gates", zap.Int("count", len(gates)))
				for _, g := range gates {
					metCount := 0
					for _, status := range g.Status.ConditionStatuses {
						if status.Met {
							metCount++
						}
					}
					logger.Info("Gate",
						zap.String("name", g.Name),
						zap.Int("conditions_met", metCount),
						zap.Int("conditions_total", len(g.Spec.Conditions)),
						zap.String("phase", string(g.Status.Phase)),
					)
				}
			}

			return nil
		},
	}

	return cmd
}
