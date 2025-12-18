// Package konductor provides a Go SDK for Konductor coordination primitives
package konductor

import (
	"github.com/LogicIQ/konductor/sdk/go/client"
	
	// Re-export subpackages for convenience
	_ "github.com/LogicIQ/konductor/sdk/go/barrier"
	_ "github.com/LogicIQ/konductor/sdk/go/gate"
	_ "github.com/LogicIQ/konductor/sdk/go/lease"
	_ "github.com/LogicIQ/konductor/sdk/go/semaphore"
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
)

// New creates a new konductor client
var New = client.New

// NewFromClient creates a konductor client from an existing Kubernetes client
var NewFromClient = client.NewFromClient