// Package main demonstrates advanced usage of the protodefault library.
//
// This example shows how to work with:
// - Nested messages
// - Enum fields
// - Repeated fields (lists)
// - Map fields (dictionaries)
// - Well-known types (google.protobuf.Duration)
package main

import (
	"fmt"
	"log"

	"github.com/yurasolovjov/protodefault"
	advancedpb "github.com/yurasolovjov/protodefault/example/advanced/proto"
)

func main() {
	fmt.Println("=== Example 1: Completely empty configuration ===")
	emptyServiceConfig()

	fmt.Println("\n=== Example 2: Partially filled configuration ===")
	partialServiceConfig()

	fmt.Println("\n=== Example 3: Overriding nested messages ===")
	nestedOverride()
}

// emptyServiceConfig demonstrates applying defaults to a complex structure.
func emptyServiceConfig() {
	cfg := &advancedpb.ServiceConfig{}

	fmt.Println("Before applying defaults:")
	fmt.Printf("  Name: %q\n", cfg.Name)
	fmt.Printf("  LogLevel: %v\n", cfg.LogLevel)
	fmt.Printf("  Database: %v\n", cfg.Database)
	fmt.Printf("  Cache: %v\n", cfg.Cache)
	fmt.Printf("  AllowedOrigins: %v\n", cfg.AllowedOrigins)
	fmt.Printf("  RetryIntervals: %v\n", cfg.RetryIntervals)
	fmt.Printf("  RateLimits: %v\n", cfg.RateLimits)
	fmt.Printf("  RequestTimeout: %v\n", cfg.RequestTimeout)

	if err := protodefault.Apply(cfg); err != nil {
		log.Fatalf("Error applying defaults: %v", err)
	}

	fmt.Println("\nAfter applying defaults:")
	printServiceConfig(cfg)
}

// partialServiceConfig demonstrates partial override.
func partialServiceConfig() {
	cfg := &advancedpb.ServiceConfig{
		Name:     "production-api",
		LogLevel: advancedpb.LogLevel_LOG_LEVEL_WARN,
		// User specifies custom origins
		AllowedOrigins: []string{"https://app.example.com"},
		// Other fields will get defaults
	}

	fmt.Println("Before applying defaults:")
	fmt.Printf("  Name: %q\n", cfg.Name)
	fmt.Printf("  LogLevel: %v\n", cfg.LogLevel)
	fmt.Printf("  AllowedOrigins: %v\n", cfg.AllowedOrigins)
	fmt.Printf("  Database: %v\n", cfg.Database)

	if err := protodefault.Apply(cfg); err != nil {
		log.Fatalf("Error applying defaults: %v", err)
	}

	fmt.Println("\nAfter applying defaults:")
	printServiceConfig(cfg)
	fmt.Println("\nNote:")
	fmt.Println("  - Name and LogLevel preserved user-provided values")
	fmt.Println("  - AllowedOrigins was NOT supplemented with defaults (list is not empty)")
	fmt.Println("  - Database and Cache were created and filled with defaults")
}

// nestedOverride demonstrates partial override of nested messages.
func nestedOverride() {
	cfg := &advancedpb.ServiceConfig{
		// User specifies only part of database settings
		Database: &advancedpb.DatabaseConfig{
			Host:     "db.production.internal",
			Port:     5433,
			Database: "production_db",
			// max_open_conns and max_idle_conns will get defaults
		},
	}

	fmt.Println("Before applying defaults:")
	fmt.Printf("  Database.Host: %q\n", cfg.Database.Host)
	fmt.Printf("  Database.Port: %d\n", cfg.Database.Port)
	fmt.Printf("  Database.MaxOpenConns: %d\n", cfg.Database.MaxOpenConns)
	fmt.Printf("  Database.MaxIdleConns: %d\n", cfg.Database.MaxIdleConns)

	if err := protodefault.Apply(cfg); err != nil {
		log.Fatalf("Error applying defaults: %v", err)
	}

	fmt.Println("\nAfter applying defaults:")
	fmt.Printf("  Database.Host: %q (user-provided)\n", cfg.Database.Host)
	fmt.Printf("  Database.Port: %d (user-provided)\n", cfg.Database.Port)
	fmt.Printf("  Database.Database: %q (user-provided)\n", cfg.Database.Database)
	fmt.Printf("  Database.MaxOpenConns: %d (default)\n", cfg.Database.MaxOpenConns)
	fmt.Printf("  Database.MaxIdleConns: %d (default)\n", cfg.Database.MaxIdleConns)
}

func printServiceConfig(cfg *advancedpb.ServiceConfig) {
	fmt.Printf("  Name: %q\n", cfg.Name)
	fmt.Printf("  LogLevel: %v\n", cfg.LogLevel)

	if cfg.Database != nil {
		fmt.Printf("  Database:\n")
		fmt.Printf("    Host: %q\n", cfg.Database.Host)
		fmt.Printf("    Port: %d\n", cfg.Database.Port)
		fmt.Printf("    Database: %q\n", cfg.Database.Database)
		fmt.Printf("    MaxOpenConns: %d\n", cfg.Database.MaxOpenConns)
		fmt.Printf("    MaxIdleConns: %d\n", cfg.Database.MaxIdleConns)
	}

	if cfg.Cache != nil {
		fmt.Printf("  Cache:\n")
		fmt.Printf("    Enabled: %v\n", cfg.Cache.Enabled)
		fmt.Printf("    TTL: %v\n", cfg.Cache.Ttl.AsDuration())
		fmt.Printf("    MaxSizeMb: %d\n", cfg.Cache.MaxSizeMb)
	}

	fmt.Printf("  AllowedOrigins: %v\n", cfg.AllowedOrigins)
	fmt.Printf("  RetryIntervals: %v\n", cfg.RetryIntervals)
	fmt.Printf("  RateLimits: %v\n", cfg.RateLimits)

	if cfg.RequestTimeout != nil {
		fmt.Printf("  RequestTimeout: %v\n", cfg.RequestTimeout.AsDuration())
	}
	if cfg.ShutdownTimeout != nil {
		fmt.Printf("  ShutdownTimeout: %v\n", cfg.ShutdownTimeout.AsDuration())
	}
}
