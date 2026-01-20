// Package konductor provides a Go SDK for Konductor coordination primitives
package konductor

import (
	"github.com/LogicIQ/konductor/sdk/go/barrier"
	"github.com/LogicIQ/konductor/sdk/go/client"
	"github.com/LogicIQ/konductor/sdk/go/gate"
	"github.com/LogicIQ/konductor/sdk/go/lease"
	"github.com/LogicIQ/konductor/sdk/go/mutex"
	"github.com/LogicIQ/konductor/sdk/go/semaphore"
)

// Client is the main konductor client
type Client = client.Client

// Config holds client configuration
type Config = client.Config

// Options for coordination operations
type Options = client.Options

// Option functions
var (
	WithTTL      = client.WithTTL
	WithTimeout  = client.WithTimeout
	WithPriority = client.WithPriority
	WithHolder   = client.WithHolder
	WithQuorum   = client.WithQuorum
)

// New creates a new konductor client
var New = client.New

// NewFromClient creates a konductor client from an existing Kubernetes client
var NewFromClient = client.NewFromClient

// Semaphore operations
var (
	SemaphoreCreate  = semaphore.Create
	SemaphoreDelete  = semaphore.Delete
	SemaphoreUpdate  = semaphore.Update
	SemaphoreGet     = semaphore.Get
	SemaphoreList    = semaphore.List
	SemaphoreAcquire = semaphore.Acquire
	SemaphoreWith    = semaphore.With
)

// Barrier operations
var (
	BarrierCreate = barrier.Create
	BarrierDelete = barrier.Delete
	BarrierUpdate = barrier.Update
	BarrierGet    = barrier.Get
	BarrierList   = barrier.List
	BarrierWait   = barrier.Wait
	BarrierArrive = barrier.Arrive
	BarrierWith   = barrier.With
)

// Gate operations
var (
	GateCreate = gate.Create
	GateDelete = gate.Delete
	GateUpdate = gate.Update
	GateGet    = gate.Get
	GateList   = gate.List
	GateWait   = gate.Wait
	GateCheck  = gate.Check
	GateOpen   = gate.Open
	GateClose  = gate.Close
	GateWith   = gate.With
)

// Lease operations
var (
	LeaseCreate      = lease.Create
	LeaseDelete      = lease.Delete
	LeaseUpdate      = lease.Update
	LeaseGet         = lease.Get
	LeaseList        = lease.List
	LeaseAcquire     = lease.Acquire
	LeaseTryAcquire  = lease.TryAcquire
	LeaseWith        = lease.With
	LeaseIsAvailable = lease.IsAvailable
)

// Mutex operations
var (
	MutexCreate   = mutex.Create
	MutexDelete   = mutex.Delete
	MutexUpdate   = mutex.Update
	MutexGet      = mutex.Get
	MutexList     = mutex.List
	MutexLock     = mutex.Lock
	MutexTryLock  = mutex.TryLock
	MutexUnlock   = mutex.Unlock
	MutexWith     = mutex.With
	MutexIsLocked = mutex.IsLocked
)
