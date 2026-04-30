package protodefault

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var wktNames = map[protoreflect.FullName]bool{
	"google.protobuf.Duration":    true,
	"google.protobuf.Timestamp":   true,
	"google.protobuf.Struct":      true,
	"google.protobuf.ListValue":   true,
	"google.protobuf.Value":       true,
	"google.protobuf.Empty":       true,
	"google.protobuf.Any":         true,
	"google.protobuf.FieldMask":   true,
	"google.protobuf.BoolValue":   true,
	"google.protobuf.Int32Value":  true,
	"google.protobuf.Int64Value":  true,
	"google.protobuf.Uint32Value": true,
	"google.protobuf.Uint64Value": true,
	"google.protobuf.FloatValue":  true,
	"google.protobuf.DoubleValue": true,
	"google.protobuf.StringValue": true,
	"google.protobuf.BytesValue":  true,
}

func isWKT(name protoreflect.FullName) bool {
	return wktNames[name]
}

func parseWKT(fd protoreflect.FieldDescriptor, s string) (protoreflect.Value, error) {
	name := fd.Message().FullName()

	switch name {
	case "google.protobuf.Empty":
		return protoreflect.Value{}, fmt.Errorf("google.protobuf.Empty does not support default_value")
	case "google.protobuf.Any":
		return protoreflect.Value{}, fmt.Errorf("google.protobuf.Any does not support default_value")
	}

	// For most WKT, we can use protojson.
	// We create an instance of the WKT message and unmarshal the string.
	// If it's a wrapper, it's just the scalar value.
	// If it's Duration, it's "15s".
	// If it's Timestamp, it's RFC3339.
	// If it's Struct, it's JSON object.
	// If it's FieldMask, it's "a,b,c".

	// Special case for FieldMask because it might not be a valid JSON string if it's just a,b,c
	if name == "google.protobuf.FieldMask" {
		// FieldMask in JSON is a string. So we should wrap it in quotes if it's not.
		if !strings.HasPrefix(s, "\"") {
			s = fmt.Sprintf("%q", s)
		}
	} else if isWrapper(name) || name == "google.protobuf.Duration" || name == "google.protobuf.Timestamp" {
		// These expect a JSON-compliant scalar (string with quotes for Duration/Timestamp/String, or raw for bool/number)
		// Try to see if it needs quotes
		if needsQuotes(name, s) {
			s = fmt.Sprintf("%q", s)
		}
	}

	// We can't easily create a dynamic message from just a descriptor without a registry.
	// But we can use the proto.Clone/New if we have a concrete type.
	// Since these are WKT, they are linked into the binary.

	return protoreflect.Value{}, fmt.Errorf("use setWKTDefault instead")
}

func setWKTDefault(m protoreflect.Message, fd protoreflect.FieldDescriptor, s string) error {
	name := fd.Message().FullName()
	if name == "google.protobuf.Empty" {
		return fmt.Errorf("field %s: google.protobuf.Empty does not support default_value", fd.FullName())
	}
	if name == "google.protobuf.Any" {
		return fmt.Errorf("field %s: google.protobuf.Any does not support default_value", fd.FullName())
	}

	// Prepare JSON
	jsonVal := s
	if needsQuotes(name, s) {
		jsonVal = fmt.Sprintf("%q", s)
	}

	// Use a dummy message of the SAME type to unmarshal into, then copy.
	// We use m.NewField(fd).Message() to get a new instance of the correct WKT type.
	newWKT := m.NewField(fd).Message()
	if err := protojson.Unmarshal([]byte(jsonVal), newWKT.Interface()); err != nil {
		return fmt.Errorf("field %s: failed to parse WKT default %q: %w", fd.FullName(), s, err)
	}

	m.Set(fd, protoreflect.ValueOfMessage(newWKT))
	return nil
}

func isWrapper(name protoreflect.FullName) bool {
	return strings.HasSuffix(string(name), "Value") && strings.HasPrefix(string(name), "google.protobuf.")
}

func needsQuotes(name protoreflect.FullName, s string) bool {
	if strings.HasPrefix(s, "\"") {
		return false
	}
	switch name {
	case "google.protobuf.Duration", "google.protobuf.Timestamp", "google.protobuf.StringValue", "google.protobuf.BytesValue", "google.protobuf.FieldMask":
		return true
	case "google.protobuf.Value":
		// Value can be anything. If it's not obviously a bool/number/null/object/array, it's probably a string that needs quotes.
		if s == "true" || s == "false" || s == "null" {
			return false
		}
		if len(s) > 0 && (s[0] == '{' || s[0] == '[' || (s[0] >= '0' && s[0] <= '9') || s[0] == '-') {
			return false
		}
		return true
	}
	return false
}
