# protodefault Usage Examples

This directory contains examples demonstrating how to use the `protodefault` library.

## Structure

```
example/
├── basic/              # Basic example with scalar types
│   ├── main.go
│   └── proto/
│       └── config.proto
├── advanced/           # Advanced example with composite types
│   ├── main.go
│   └── proto/
│       └── service.proto
└── with-validation/    # Integration with protovalidate
    ├── main.go
    ├── proto/
    │   └── user.proto
    └── README.md
```

## Running Examples

### Basic Example

Demonstrates applying defaults to simple scalar fields (string, int32, bool):

```bash
go run ./example/basic/...
```

**What it shows:**
- Applying defaults to an empty message
- Preserving user-provided values (not overwritten)
- Behavior with a fully filled message

### Advanced Example

Demonstrates working with composite types:

```bash
go run ./example/advanced/...
```

**What it shows:**
- Nested messages — automatic initialization and recursive application of defaults
- Enum fields — default by constant name
- Repeated fields — default in JSON array format
- Map fields — default in JSON object format
- Well-known types — `google.protobuf.Duration`

### With Validation Example

Demonstrates integration with [protovalidate](https://github.com/bufbuild/protovalidate):

```bash
go run ./example/with-validation/...
```

**What it shows:**
- Combining `default_value` with validation rules
- Recommended processing order: parse → apply defaults → validate
- Required fields (no default) vs optional fields (with default)
- Validation error messages for constraint violations

See [with-validation/README.md](with-validation/README.md) for detailed documentation.

## Generating Proto Files

If you modify `.proto` files, regenerate the Go code:

```bash
# From project root
protoc -I proto -I example/basic/proto \
  --go_out=. --go_opt=module=github.com/yurasolovjov/protodefault \
  example/basic/proto/config.proto

protoc -I proto -I example/advanced/proto \
  --go_out=. --go_opt=module=github.com/yurasolovjov/protodefault \
  example/advanced/proto/service.proto
```

## default_value Formats

| Field Type | Format | Example |
|------------|--------|---------|
| `string` | Text | `"localhost"` |
| `int32/int64` | Number | `"8080"` |
| `bool` | `"true"` or `"false"` | `"true"` |
| `enum` | Constant name | `"LOG_LEVEL_INFO"` |
| `repeated T` | JSON array | `'["a", "b"]'` |
| `map<K,V>` | JSON object | `'{"key": 100}'` |
| `Duration` | Seconds with suffix | `"30s"`, `"300s"` |

> **Note:** For `Duration`, use seconds format (`"30s"`), not minutes (`"5m"`), as protojson only supports seconds and nanoseconds.
