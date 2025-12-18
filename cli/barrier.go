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

func newBarrierCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "barrier",
		Short: "Manage barriers",
		Long:  "Wait for barriers and signal arrivals",
	}

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

			startTime := time.Now()
			for {
				var barrier syncv1.Barrier
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      barrierName,
					Namespace: namespace,
				}, &barrier); err != nil {
					return fmt.Errorf("failed to get barrier: %w", err)
				}

				switch barrier.Status.Phase {
				case syncv1.BarrierPhaseOpen:
					fmt.Printf("✓ Barrier '%s' is open! (arrived: %d/%d)\n", 
						barrierName, barrier.Status.Arrived, barrier.Spec.Expected)
					return nil
				case syncv1.BarrierPhaseFailed:
					return fmt.Errorf("barrier '%s' failed (timeout or error)", barrierName)
				case syncv1.BarrierPhaseWaiting:
					if timeout > 0 && time.Since(startTime) > timeout {
						return fmt.Errorf("timeout waiting for barrier '%s'", barrierName)
					}
					
					fmt.Printf("⏳ Waiting for barrier '%s' (arrived: %d/%d)...\n", 
						barrierName, barrier.Status.Arrived, barrier.Spec.Expected)
					time.Sleep(5 * time.Second)
				}
			}
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Timeout for waiting (e.g., 30s, 5m)")

	return cmd
}

func newBarrierArriveCmd() *cobra.Command {
	var holder string

	cmd := &cobra.Command{
		Use:   "arrive <barrier-name>",
		Short: "Signal arrival at a barrier",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			barrierName := args[0]
			ctx := context.Background()

			if holder == "" {
				holder = os.Getenv("HOSTNAME")
				if holder == "" {
					holder = fmt.Sprintf("koncli-%d", time.Now().Unix())
				}
			}


			arrival := &syncv1.Arrival{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%s", barrierName, holder),
					Namespace: namespace,
					Labels: map[string]string{
						"barrier": barrierName,
					},
				},
				Spec: syncv1.ArrivalSpec{
					Barrier: barrierName,
					Holder:  holder,
				},
			}

			if err := k8sClient.Create(ctx, arrival); err != nil {
				return fmt.Errorf("failed to create arrival: %w", err)
			}

			fmt.Printf("✓ Signaled arrival at barrier '%s' (holder: %s)\n", barrierName, holder)


			var barrier syncv1.Barrier
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      barrierName,
				Namespace: namespace,
			}, &barrier); err == nil {
				fmt.Printf("📊 Barrier status: %d/%d arrived, phase: %s\n", 
					barrier.Status.Arrived, barrier.Spec.Expected, barrier.Status.Phase)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&holder, "holder", "", "Arrival holder identifier (defaults to hostname)")

	return cmd
}

func newBarrierListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all barriers",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			var barriers syncv1.BarrierList
			if err := k8sClient.List(ctx, &barriers, client.InNamespace(namespace)); err != nil {
				return fmt.Errorf("failed to list barriers: %w", err)
			}

			if len(barriers.Items) == 0 {
				fmt.Println("No barriers found")
				return nil
			}

			fmt.Printf("%-20s %-10s %-10s %-10s %-15s\n", "NAME", "EXPECTED", "ARRIVED", "PHASE", "OPENED")
			for _, barrier := range barriers.Items {
				opened := "N/A"
				if barrier.Status.OpenedAt != nil {
					opened = barrier.Status.OpenedAt.Format("15:04:05")
				}

				fmt.Printf("%-20s %-10d %-10d %-10s %-15s\n",
					barrier.Name,
					barrier.Spec.Expected,
					barrier.Status.Arrived,
					barrier.Status.Phase,
					opened,
				)
			}

			return nil
		},
	}

	return cmd
}