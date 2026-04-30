// Package main demonstrates the integration of protodefault with protovalidate.
//
// This example shows the recommended pattern for processing protobuf messages:
//  1. Parse user input (JSON, YAML, etc.) into a protobuf message
//  2. Apply default values using protodefault
//  3. Validate the message using protovalidate
//
// This order ensures that:
//   - Required fields without defaults will fail validation if not provided
//   - Optional fields with defaults will pass validation even if not provided
//   - User-provided values are never overwritten by defaults
package main

import (
	"fmt"
	"os"

	"buf.build/go/protovalidate"

	"github.com/yurasolovjov/protodefault"
	pb "github.com/yurasolovjov/protodefault/example/with-validation/gen"
)

func main() {
	// Create a validator instance (reuse it for multiple validations)
	validator, err := protovalidate.New()
	if err != nil {
		fmt.Printf("Failed to create validator: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Example 1: Valid user with minimal required fields ===")
	fmt.Println()
	example1(validator)

	fmt.Println()
	fmt.Println("=== Example 2: Invalid user - missing required fields ===")
	fmt.Println()
	example2(validator)

	fmt.Println()
	fmt.Println("=== Example 3: Invalid user - validation constraint violated ===")
	fmt.Println()
	example3(validator)

	fmt.Println()
	fmt.Println("=== Example 4: Fully customized user ===")
	fmt.Println()
	example4(validator)
}

// example1 demonstrates a valid user with only required fields provided.
// All optional fields will receive default values.
func example1(validator protovalidate.Validator) {
	user := &pb.User{
		Username: "johndoe",
		Email:    "john@example.com",
	}

	fmt.Println("Before applying defaults:")
	printUser(user)

	// Step 1: Apply defaults
	if err := protodefault.Apply(user); err != nil {
		fmt.Printf("Error applying defaults: %v\n", err)
		return
	}

	fmt.Println("\nAfter applying defaults:")
	printUser(user)

	// Step 2: Validate
	if err := validator.Validate(user); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
		return
	}

	fmt.Println("\n✓ Validation passed!")
}

// example2 demonstrates validation failure when required fields are missing.
func example2(validator protovalidate.Validator) {
	user := &pb.User{
		// Missing username and email - these are required!
	}

	fmt.Println("Before applying defaults:")
	printUser(user)

	// Step 1: Apply defaults
	if err := protodefault.Apply(user); err != nil {
		fmt.Printf("Error applying defaults: %v\n", err)
		return
	}

	fmt.Println("\nAfter applying defaults:")
	printUser(user)

	// Step 2: Validate - this will fail because username and email are empty
	if err := validator.Validate(user); err != nil {
		fmt.Printf("\n✗ Validation failed (expected):\n%v\n", err)
		return
	}

	fmt.Println("\n✓ Validation passed!")
}

// example3 demonstrates validation failure when a constraint is violated.
func example3(validator protovalidate.Validator) {
	user := &pb.User{
		Username: "ab",            // Too short! Minimum is 3 characters
		Email:    "invalid-email", // Invalid email format
		Age:      10,              // Too young! Minimum is 13
	}

	fmt.Println("Before applying defaults:")
	printUser(user)

	// Step 1: Apply defaults
	if err := protodefault.Apply(user); err != nil {
		fmt.Printf("Error applying defaults: %v\n", err)
		return
	}

	fmt.Println("\nAfter applying defaults:")
	printUser(user)

	// Step 2: Validate - this will fail due to constraint violations
	if err := validator.Validate(user); err != nil {
		fmt.Printf("\n✗ Validation failed (expected):\n%v\n", err)
		return
	}

	fmt.Println("\n✓ Validation passed!")
}

// example4 demonstrates a fully customized user where all fields are provided.
func example4(validator protovalidate.Validator) {
	user := &pb.User{
		Username:    "admin",
		Email:       "admin@example.com",
		DisplayName: "System Administrator",
		Age:         35,
		Role:        pb.Role_ROLE_ADMIN,
		Settings: &pb.Settings{
			EmailNotifications: false,
			Language:           "de",
			ItemsPerPage:       50,
			Theme:              "dark",
		},
		Tags: []string{"admin", "verified", "premium"},
	}

	fmt.Println("Before applying defaults:")
	printUser(user)

	// Step 1: Apply defaults (nothing should change since all fields are set)
	if err := protodefault.Apply(user); err != nil {
		fmt.Printf("Error applying defaults: %v\n", err)
		return
	}

	fmt.Println("\nAfter applying defaults:")
	printUser(user)

	// Step 2: Validate
	if err := validator.Validate(user); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
		return
	}

	fmt.Println("\n✓ Validation passed!")
	fmt.Println("  Note: All user-provided values were preserved (not overwritten by defaults)")
}

func printUser(u *pb.User) {
	fmt.Printf("  Username:     %q\n", u.GetUsername())
	fmt.Printf("  Email:        %q\n", u.GetEmail())
	fmt.Printf("  DisplayName:  %q\n", u.GetDisplayName())
	fmt.Printf("  Age:          %d\n", u.GetAge())
	fmt.Printf("  Role:         %s\n", u.GetRole())
	fmt.Printf("  Tags:         %v\n", u.GetTags())
	if s := u.GetSettings(); s != nil {
		fmt.Printf("  Settings:\n")
		fmt.Printf("    EmailNotifications: %v\n", s.GetEmailNotifications())
		fmt.Printf("    Language:           %q\n", s.GetLanguage())
		fmt.Printf("    ItemsPerPage:       %d\n", s.GetItemsPerPage())
		fmt.Printf("    Theme:              %q\n", s.GetTheme())
	} else {
		fmt.Printf("  Settings:     <nil>\n")
	}
}
