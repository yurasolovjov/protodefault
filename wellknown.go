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
