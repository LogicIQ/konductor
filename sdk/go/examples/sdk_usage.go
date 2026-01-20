package examples

import (
	"context"
	"fmt"
	"log"
	"time"

	konductor "github.com/LogicIQ/konductor/sdk/go"
)

func SDKUsageExample() {
	// Create client
	client, err := konductor.New(&konductor.Config{
		Namespace: "default",
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Demonstrate all CRUD operations for each primitive
	demonstrateSemaphore(client, ctx)
	demonstrateBarrier(client, ctx)
	demonstrateGate(client, ctx)
	demonstrateLease(client, ctx)
}

func demonstrateSemaphore(client *konductor.Client, ctx context.Context) {
	fmt.Println("=== Semaphore Operations ===")

	// Create semaphore
	err := konductor.SemaphoreCreate(client, ctx, "demo-semaphore", 3,
		konductor.WithTTL(10*time.Minute))
	if err != nil {
		log.Printf("Create semaphore error: %v", err)
		return
	}
	fmt.Println("✓ Created semaphore")

	// List semaphores
	semaphores, err := konductor.SemaphoreList(client, ctx)
	if err != nil {
		log.Printf("List semaphores error: %v", err)
		return
	}
	fmt.Printf("✓ Listed %d semaphores\n", len(semaphores))

	// Get specific semaphore
	semaphore, err := konductor.SemaphoreGet(client, ctx, "demo-semaphore")
	if err != nil {
		log.Printf("Get semaphore error: %v", err)
		return
	}
	fmt.Printf("✓ Got semaphore: %d permits\n", semaphore.Spec.Permits)

	// Acquire permit
	permit, err := konductor.SemaphoreAcquire(client, ctx, "demo-semaphore",
		konductor.WithHolder("demo-holder"),
		konductor.WithTTL(5*time.Minute))
	if err != nil {
		log.Printf("Acquire permit error: %v", err)
		return
	}
	fmt.Printf("✓ Acquired permit (holder: %s)\n", permit.Holder())

	// Use semaphore with automatic cleanup
	err = konductor.SemaphoreWith(client, ctx, "demo-semaphore", func() error {
		fmt.Println("✓ Executing protected code with semaphore")
		time.Sleep(1 * time.Second)
		return nil
	}, konductor.WithHolder("auto-holder"))
	if err != nil {
		log.Printf("Semaphore with error: %v", err)
	}

	// Release permit
	err = permit.Release(ctx)
	if err != nil {
		log.Printf("Release permit error: %v", err)
	} else {
		fmt.Println("✓ Released permit")
	}

	// Update semaphore (increase permits)
	semaphore.Spec.Permits = 5
	err = konductor.SemaphoreUpdate(client, ctx, semaphore)
	if err != nil {
		log.Printf("Update semaphore error: %v", err)
	} else {
		fmt.Println("✓ Updated semaphore permits")
	}

	// Delete semaphore
	err = konductor.SemaphoreDelete(client, ctx, "demo-semaphore")
	if err != nil {
		log.Printf("Delete semaphore error: %v", err)
	} else {
		fmt.Println("✓ Deleted semaphore")
	}

	fmt.Println()
}

func demonstrateBarrier(client *konductor.Client, ctx context.Context) {
	fmt.Println("=== Barrier Operations ===")

	// Create barrier
	err := konductor.BarrierCreate(client, ctx, "demo-barrier", 2)
	if err != nil {
		log.Printf("Create barrier error: %v", err)
		return
	}
	fmt.Println("✓ Created barrier expecting 2 arrivals")

	// List barriers
	barriers, err := konductor.BarrierList(client, ctx)
	if err != nil {
		log.Printf("List barriers error: %v", err)
		return
	}
	fmt.Printf("✓ Listed %d barriers\n", len(barriers))

	// Get specific barrier
	barrier, err := konductor.BarrierGet(client, ctx, "demo-barrier")
	if err != nil {
		log.Printf("Get barrier error: %v", err)
		return
	}
	fmt.Printf("✓ Got barrier: %d/%d arrived\n", barrier.Status.Arrived, barrier.Spec.Expected)

	// Signal arrival (first)
	err = konductor.BarrierArrive(client, ctx, "demo-barrier",
		konductor.WithHolder("worker-1"))
	if err != nil {
		log.Printf("Arrive at barrier error: %v", err)
		return
	}
	fmt.Println("✓ Worker-1 arrived at barrier")

	// Signal arrival (second) - this should open the barrier
	err = konductor.BarrierArrive(client, ctx, "demo-barrier",
		konductor.WithHolder("worker-2"))
	if err != nil {
		log.Printf("Arrive at barrier error: %v", err)
		return
	}
	fmt.Println("✓ Worker-2 arrived at barrier")

	// Wait for barrier (should be immediate since we have 2/2 arrivals)
	err = konductor.BarrierWait(client, ctx, "demo-barrier",
		konductor.WithTimeout(5*time.Second))
	if err != nil {
		log.Printf("Wait for barrier error: %v", err)
	} else {
		fmt.Println("✓ Barrier opened!")
	}

	// Use barrier with function execution
	err = konductor.BarrierWith(client, ctx, "demo-barrier", func() error {
		fmt.Println("✓ Executed function and signaled arrival")
		return nil
	}, konductor.WithHolder("func-worker"))
	if err != nil {
		log.Printf("Barrier with error: %v", err)
	}

	// Update barrier (change expected count)
	barrier.Spec.Expected = 3
	err = konductor.BarrierUpdate(client, ctx, barrier)
	if err != nil {
		log.Printf("Update barrier error: %v", err)
	} else {
		fmt.Println("✓ Updated barrier expected count")
	}

	// Delete barrier
	err = konductor.BarrierDelete(client, ctx, "demo-barrier")
	if err != nil {
		log.Printf("Delete barrier error: %v", err)
	} else {
		fmt.Println("✓ Deleted barrier")
	}

	fmt.Println()
}

func demonstrateGate(client *konductor.Client, ctx context.Context) {
	fmt.Println("=== Gate Operations ===")

	// Create gate
	err := konductor.GateCreate(client, ctx, "demo-gate")
	if err != nil {
		log.Printf("Create gate error: %v", err)
		return
	}
	fmt.Println("✓ Created gate")

	// List gates
	gates, err := konductor.GateList(client, ctx)
	if err != nil {
		log.Printf("List gates error: %v", err)
		return
	}
	fmt.Printf("✓ Listed %d gates\n", len(gates))

	// Get specific gate
	gate, err := konductor.GateGet(client, ctx, "demo-gate")
	if err != nil {
		log.Printf("Get gate error: %v", err)
		return
	}
	fmt.Printf("✓ Got gate: phase %s\n", gate.Status.Phase)

	// Check if gate is open
	isOpen, err := konductor.GateCheck(client, ctx, "demo-gate")
	if err != nil {
		log.Printf("Check gate error: %v", err)
	} else {
		fmt.Printf("✓ Gate is open: %t\n", isOpen)
	}

	// Manually open gate
	err = konductor.GateOpen(client, ctx, "demo-gate")
	if err != nil {
		log.Printf("Open gate error: %v", err)
	} else {
		fmt.Println("✓ Manually opened gate")
	}

	// Wait for gate (should be immediate since we opened it)
	err = konductor.GateWait(client, ctx, "demo-gate",
		konductor.WithTimeout(5*time.Second))
	if err != nil {
		log.Printf("Wait for gate error: %v", err)
	} else {
		fmt.Println("✓ Gate wait completed")
	}

	// Use gate with function execution
	err = konductor.GateWith(client, ctx, "demo-gate", func() error {
		fmt.Println("✓ Executed function after gate opened")
		return nil
	})
	if err != nil {
		log.Printf("Gate with error: %v", err)
	}

	// Close gate
	err = konductor.GateClose(client, ctx, "demo-gate")
	if err != nil {
		log.Printf("Close gate error: %v", err)
	} else {
		fmt.Println("✓ Manually closed gate")
	}

	// Update gate (add conditions - this would typically be done by controllers)
	// gate.Spec.Conditions = append(gate.Spec.Conditions, ...)
	err = konductor.GateUpdate(client, ctx, gate)
	if err != nil {
		log.Printf("Update gate error: %v", err)
	} else {
		fmt.Println("✓ Updated gate")
	}

	// Delete gate
	err = konductor.GateDelete(client, ctx, "demo-gate")
	if err != nil {
		log.Printf("Delete gate error: %v", err)
	} else {
		fmt.Println("✓ Deleted gate")
	}

	fmt.Println()
}

func demonstrateLease(client *konductor.Client, ctx context.Context) {
	fmt.Println("=== Lease Operations ===")

	// Create lease
	err := konductor.LeaseCreate(client, ctx, "demo-lease",
		konductor.WithTTL(10*time.Minute))
	if err != nil {
		log.Printf("Create lease error: %v", err)
		return
	}
	fmt.Println("✓ Created lease")

	// List leases
	leases, err := konductor.LeaseList(client, ctx)
	if err != nil {
		log.Printf("List leases error: %v", err)
		return
	}
	fmt.Printf("✓ Listed %d leases\n", len(leases))

	// Get specific lease
	lease, err := konductor.LeaseGet(client, ctx, "demo-lease")
	if err != nil {
		log.Printf("Get lease error: %v", err)
		return
	}
	fmt.Printf("✓ Got lease: phase %s\n", lease.Status.Phase)

	// Check if lease is available
	available, err := konductor.LeaseIsAvailable(client, ctx, "demo-lease")
	if err != nil {
		log.Printf("Check lease availability error: %v", err)
	} else {
		fmt.Printf("✓ Lease available: %t\n", available)
	}

	// Try to acquire lease (non-blocking)
	leaseHandle, err := konductor.LeaseTryAcquire(client, ctx, "demo-lease",
		konductor.WithHolder("demo-holder"))
	if err != nil {
		log.Printf("Try acquire lease error: %v", err)
	} else {
		fmt.Printf("✓ Try-acquired lease (holder: %s)\n", leaseHandle.Holder())

		// Release the lease
		err = leaseHandle.Release(ctx)
		if err != nil {
			log.Printf("Release lease error: %v", err)
		} else {
			fmt.Println("✓ Released lease")
		}
	}

	// Acquire lease with blocking
	leaseHandle, err = konductor.LeaseAcquire(client, ctx, "demo-lease",
		konductor.WithHolder("blocking-holder"),
		konductor.WithPriority(10),
		konductor.WithTimeout(5*time.Second))
	if err != nil {
		log.Printf("Acquire lease error: %v", err)
	} else {
		fmt.Printf("✓ Acquired lease (holder: %s)\n", leaseHandle.Holder())

		// Use lease with automatic cleanup
		err = konductor.LeaseWith(client, ctx, "demo-lease", func() error {
			fmt.Println("✓ Executing protected code with lease")
			time.Sleep(1 * time.Second)
			return nil
		}, konductor.WithHolder("auto-holder"))
		if err != nil {
			log.Printf("Lease with error: %v", err)
		}

		// Release the lease
		err = leaseHandle.Release(ctx)
		if err != nil {
			log.Printf("Release lease error: %v", err)
		} else {
			fmt.Println("✓ Released lease")
		}
	}

	// Update lease (change TTL)
	lease.Spec.TTL.Duration = 15 * time.Minute
	err = konductor.LeaseUpdate(client, ctx, lease)
	if err != nil {
		log.Printf("Update lease error: %v", err)
	} else {
		fmt.Println("✓ Updated lease TTL")
	}

	// Delete lease
	err = konductor.LeaseDelete(client, ctx, "demo-lease")
	if err != nil {
		log.Printf("Delete lease error: %v", err)
	} else {
		fmt.Println("✓ Deleted lease")
	}

	fmt.Println()
}

// Example of error handling and cleanup patterns
func robustSemaphoreUsage(client *konductor.Client, ctx context.Context) error {
	// Create semaphore if it doesn't exist
	_, err := konductor.SemaphoreGet(client, ctx, "api-limit")
	if err != nil {
		// Semaphore doesn't exist, create it
		err = konductor.SemaphoreCreate(client, ctx, "api-limit", 10)
		if err != nil {
			return fmt.Errorf("failed to create semaphore: %w", err)
		}
	}

	// Acquire permit with timeout
	permit, err := konductor.SemaphoreAcquire(client, ctx, "api-limit",
		konductor.WithTimeout(30*time.Second),
		konductor.WithTTL(5*time.Minute))
	if err != nil {
		return fmt.Errorf("failed to acquire permit: %w", err)
	}

	// Ensure cleanup on any exit path
	defer func() {
		if releaseErr := permit.Release(ctx); releaseErr != nil {
			log.Printf("Failed to release permit: %v", releaseErr)
		}
	}()

	// Your protected code here
	fmt.Println("Executing rate-limited operation")
	time.Sleep(2 * time.Second)

	return nil
}

// Example of coordinated pipeline execution
func pipelineExample(client *konductor.Client, ctx context.Context) error {
	// Create barriers for pipeline stages
	stages := []string{"stage1-complete", "stage2-complete", "stage3-complete"}
	for _, stage := range stages {
		err := konductor.BarrierCreate(client, ctx, stage, 3) // 3 workers per stage
		if err != nil {
			return fmt.Errorf("failed to create barrier %s: %w", stage, err)
		}
	}

	// Simulate worker execution
	workerID := "worker-1"

	// Stage 1
	fmt.Printf("Worker %s: Executing stage 1\n", workerID)
	time.Sleep(1 * time.Second)

	err := konductor.BarrierArrive(client, ctx, "stage1-complete",
		konductor.WithHolder(workerID))
	if err != nil {
		return fmt.Errorf("failed to signal stage 1 completion: %w", err)
	}

	// Wait for all workers to complete stage 1
	err = konductor.BarrierWait(client, ctx, "stage1-complete",
		konductor.WithTimeout(30*time.Second))
	if err != nil {
		return fmt.Errorf("timeout waiting for stage 1: %w", err)
	}

	fmt.Printf("Worker %s: Stage 1 complete, starting stage 2\n", workerID)

	// Continue with subsequent stages...

	// Cleanup
	for _, stage := range stages {
		konductor.BarrierDelete(client, ctx, stage)
	}

	return nil
}
