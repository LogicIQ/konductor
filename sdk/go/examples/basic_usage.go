package examples

import (
	"context"
	"fmt"
	"log"
	"time"

	konductor "github.com/LogicIQ/konductor/sdk/go"
	"github.com/LogicIQ/konductor/sdk/go/barrier"
	"github.com/LogicIQ/konductor/sdk/go/gate"
	"github.com/LogicIQ/konductor/sdk/go/lease"
	"github.com/LogicIQ/konductor/sdk/go/semaphore"
)

func BasicUsageExample() {
	// Create konductor client
	client, err := konductor.New(&konductor.Config{
		Namespace: "default",
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Example 1: Semaphore usage
	fmt.Println("=== Semaphore Example ===")
	if err := semaphoreExample(ctx, client); err != nil {
		log.Printf("Semaphore example failed: %v", err)
	}

	// Example 2: Barrier usage
	fmt.Println("\n=== Barrier Example ===")
	if err := barrierExample(ctx, client); err != nil {
		log.Printf("Barrier example failed: %v", err)
	}

	// Example 3: Lease usage
	fmt.Println("\n=== Lease Example ===")
	if err := leaseExample(ctx, client); err != nil {
		log.Printf("Lease example failed: %v", err)
	}

	// Example 4: Gate usage
	fmt.Println("\n=== Gate Example ===")
	if err := gateExample(ctx, client); err != nil {
		log.Printf("Gate example failed: %v", err)
	}
}

func semaphoreExample(ctx context.Context, client *konductor.Client) error {
	// Acquire a semaphore permit
	permit, err := semaphore.Acquire(client, ctx, "api-quota",
		konductor.WithTTL(5*time.Minute),
		konductor.WithTimeout(30*time.Second),
		konductor.WithHolder("example-app"),
	)
	if err != nil {
		return fmt.Errorf("failed to acquire semaphore: %w", err)
	}
	defer permit.Release(ctx)

	fmt.Printf("Acquired semaphore permit (holder: %s)\n", permit.Holder())

	// Simulate work
	fmt.Println("Doing work with rate limiting...")
	time.Sleep(2 * time.Second)

	fmt.Println("Work completed, releasing permit")
	return nil
}

func barrierExample(ctx context.Context, client *konductor.Client) error {
	// Wait for a barrier to open
	fmt.Println("Waiting for barrier 'stage-1'...")
	if err := barrier.Wait(client, ctx, "stage-1",
		konductor.WithTimeout(30*time.Second),
	); err != nil {
		return fmt.Errorf("failed to wait for barrier: %w", err)
	}

	fmt.Println("Barrier 'stage-1' is open!")

	// Do some work
	fmt.Println("Processing stage 1...")
	time.Sleep(1 * time.Second)

	// Signal arrival at next barrier
	if err := barrier.Arrive(client, ctx, "stage-2",
		konductor.WithHolder("example-app"),
	); err != nil {
		return fmt.Errorf("failed to arrive at barrier: %w", err)
	}

	fmt.Println("Signaled arrival at barrier 'stage-2'")
	return nil
}

func leaseExample(ctx context.Context, client *konductor.Client) error {
	// Try to acquire a lease
	leaseHandle, err := lease.Acquire(client, ctx, "singleton-task",
		konductor.WithPriority(5),
		konductor.WithTimeout(10*time.Second),
		konductor.WithHolder("example-app"),
	)
	if err != nil {
		return fmt.Errorf("failed to acquire lease: %w", err)
	}
	defer leaseHandle.Release(ctx)

	fmt.Printf("Acquired lease (holder: %s)\n", leaseHandle.Holder())

	// Do singleton work
	fmt.Println("Performing singleton task...")
	time.Sleep(2 * time.Second)

	fmt.Println("Singleton task completed, releasing lease")
	return nil
}

func gateExample(ctx context.Context, client *konductor.Client) error {
	// Wait for gate to open (all conditions met)
	fmt.Println("Waiting for gate 'processing-gate'...")
	if err := gate.Wait(client, ctx, "processing-gate",
		konductor.WithTimeout(30*time.Second),
	); err != nil {
		return fmt.Errorf("failed to wait for gate: %w", err)
	}

	fmt.Println("Gate 'processing-gate' is open!")

	// Check gate conditions
	conditions, err := gate.GetConditions(client, ctx, "processing-gate")
	if err != nil {
		return fmt.Errorf("failed to get gate conditions: %w", err)
	}

	fmt.Println("Gate conditions:")
	for _, condition := range conditions {
		status := "[NOT MET]"
		if condition.Met {
			status = "[MET]"
		}
		fmt.Printf("  %s %s/%s: %s\n", status, condition.Type, condition.Name, condition.Message)
	}

	return nil
}
