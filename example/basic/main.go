// Package main demonstrates basic usage of the protodefault library.
//
// This example shows how to apply default values
// to simple scalar fields of a protobuf message.
package main

import (
	"fmt"
	"log"

	"github.com/yurasolovjov/protodefault"
	basicpb "github.com/yurasolovjov/protodefault/example/basic/proto"
)

func main() {
	fmt.Println("=== Example 1: Empty configuration ===")
	emptyConfig()

	fmt.Println("\n=== Example 2: Partially filled configuration ===")
	partialConfig()

	fmt.Println("\n=== Example 3: Fully filled configuration ===")
	fullConfig()
}

// emptyConfig demonstrates applying defaults to an empty message.
func emptyConfig() {
	cfg := &basicpb.Config{}

	fmt.Println("Before applying defaults:")
	printConfig(cfg)

	if err := protodefault.Apply(cfg); err != nil {
		log.Fatalf("Error applying defaults: %v", err)
	}

	fmt.Println("\nAfter applying defaults:")
	printConfig(cfg)
}

// partialConfig demonstrates that user-provided values are not overwritten.
func partialConfig() {
	cfg := &basicpb.Config{
		Host: "192.168.1.100", // User specified custom host
		Port: 9000,            // User specified custom port
		// debug, timeout_seconds, max_connections — not set
	}

	fmt.Println("Before applying defaults:")
	printConfig(cfg)

	if err := protodefault.Apply(cfg); err != nil {
		log.Fatalf("Error applying defaults: %v", err)
	}

	fmt.Println("\nAfter applying defaults:")
	printConfig(cfg)
	fmt.Println("(Host and Port preserved user-provided values)")
}

// fullConfig demonstrates that a fully filled message is not modified.
func fullConfig() {
	cfg := &basicpb.Config{
		Host:           "production.example.com",
		Port:           443,
		Debug:          true,
		TimeoutSeconds: 60,
		MaxConnections: 500,
	}

	fmt.Println("Before applying defaults:")
	printConfig(cfg)

	if err := protodefault.Apply(cfg); err != nil {
		log.Fatalf("Error applying defaults: %v", err)
	}

	fmt.Println("\nAfter applying defaults:")
	printConfig(cfg)
	fmt.Println("(All values preserved, except those equal to zero-value)")
}

func printConfig(cfg *basicpb.Config) {
	fmt.Printf("  Host:           %q\n", cfg.Host)
	fmt.Printf("  Port:           %d\n", cfg.Port)
	fmt.Printf("  Debug:          %v\n", cfg.Debug)
	fmt.Printf("  TimeoutSeconds: %d\n", cfg.TimeoutSeconds)
	fmt.Printf("  MaxConnections: %d\n", cfg.MaxConnections)
}
