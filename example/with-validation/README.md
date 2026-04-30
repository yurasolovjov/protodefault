# protodefault + protovalidate Integration Example

This example demonstrates how to use `protodefault` together with `protovalidate` for a complete configuration processing pipeline.

## Overview

The recommended pattern for processing protobuf messages:

1. **Parse** user input (JSON, YAML, etc.) into a protobuf message
2. **Apply defaults** using `protodefault.Apply()`
3. **Validate** the message using `protovalidate.Validate()`

This order ensures that:
- Required fields without defaults will fail validation if not provided
- Optional fields with defaults will pass validation even if not provided
- User-provided values are never overwritten by defaults

## Project Structure

```
with-validation/
├── proto/
│   ├── user.proto                    # Message with both validation rules and defaults
│   └── defaults/v1/options.proto     # Copy of options.proto (for buf to resolve imports)
├── gen/
│   └── user.pb.go                    # Generated Go code
├── main.go                           # Example usage
├── buf.yaml                          # Buf configuration
├── buf.gen.yaml                      # Buf code generation config
└── go.mod
```

## Key Concepts

### Combining Defaults and Validation

In `user.proto`, fields can have both `default_value` and validation rules:

```proto
// User's age. Defaults to 18.
// Must be between 13 and 150.
int32 age = 4 [
  (defaults.v1.default_value) = "18",
  (buf.validate.field).int32 = {
    gte: 13,
    lte: 150
  }
];
```

This means:
- If `age` is not provided, it defaults to `18`
- After defaults are applied, validation ensures `age` is between 13 and 150
- The default value `18` satisfies the validation constraint

### Required vs Optional Fields

Fields without `default_value` are effectively required:

```proto
// No default - this field is required.
string username = 1 [(buf.validate.field).string = {
  min_len: 3,
  max_len: 50
}];
```

If `username` is empty after applying defaults, validation will fail.

## Running the Example

```bash
# From the example directory
go run .

# Or from project root
go run ./example/with-validation/...
```

## Regenerating Proto Code

If you modify `user.proto`:

```bash
cd example/with-validation
buf dep update
buf generate

# Remove the generated defaults/v1 (we use the main package's version)
rm -rf gen/defaults
```

## Example Output

```
=== Example 1: Valid user with minimal required fields ===

Before applying defaults:
  Username:     "johndoe"
  Email:        "john@example.com"
  DisplayName:  ""
  Age:          0
  ...

After applying defaults:
  Username:     "johndoe"
  Email:        "john@example.com"
  DisplayName:  "Anonymous"
  Age:          18
  Role:         ROLE_USER
  Tags:         [new-user]
  Settings:
    EmailNotifications: true
    Language:           "en"
    ItemsPerPage:       25
    Theme:              "light"

✓ Validation passed!
```

## Important Notes

### Proto3 Scalar Limitation

For proto3 scalar fields without `optional`, the library cannot distinguish between "user explicitly set zero/false/empty" and "not set". This means:

- `bool` field set to `false` will be overwritten by default `true`
- `int32` field set to `0` will be overwritten by default

**Solution**: Use `optional` keyword for fields where you need to distinguish explicit zero from unset:

```proto
optional bool email_notifications = 1 [(defaults.v1.default_value) = "true"];
```

### Dependencies

This example uses:
- `buf.build/go/protovalidate` - Validation library
- `buf.build/bufbuild/protovalidate` - Validation proto definitions
- `github.com/yurasolovjov/protodefault` - Default values library
