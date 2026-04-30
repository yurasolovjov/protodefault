package protodefault

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func setRepeatedField(m protoreflect.Message, fd protoreflect.FieldDescriptor, valStr string) error {
	list := m.Mutable(fd).List()
	// Clear any existing just in case, though isSet already checked len == 0
	for list.Len() > 0 {
		list.Truncate(0)
	}

	// We use protojson to parse the array
	// To do this easily, we can create a temporary message with just this field
	// but it's simpler to use a wrapper message if we had one.
	// Actually, protojson.Unmarshal can't unmarshal directly into a protoreflect.List.
	// We'll have to parse it as a JSON array and manually populate.

	// Better way: use a placeholder message that matches the structure.
	// Or just use a simple JSON parser for the array?
	// The spec says: "JSON array in a string. Parsed via protojson for compatibility with all types."
	// To use protojson, we need a message.

	msg := m.New().Interface()
	// We construct a JSON object {"field_name": [valStr]} and unmarshal it into msg.
	jsonStr := fmt.Sprintf(`{"%s": %s}`, fd.Name(), valStr)
	if err := protojson.Unmarshal([]byte(jsonStr), msg); err != nil {
		return fmt.Errorf("field %s: failed to parse default_value as JSON array: %w", fd.FullName(), err)
	}

	// Now copy from msg to m
	srcList := msg.ProtoReflect().Get(fd).List()
	for i := 0; i < srcList.Len(); i++ {
		val := srcList.Get(i)
		list.Append(val)
		// After appending, if it's a message, we must apply defaults to it
		if fd.Kind() == protoreflect.MessageKind {
			if err := applyMessage(val.Message(), make(map[protoreflect.FullName]bool)); err != nil {
				return err
			}
		}
	}

	return nil
}

func setMapField(m protoreflect.Message, fd protoreflect.FieldDescriptor, valStr string) error {
	mp := m.Mutable(fd).Map()

	msg := m.New().Interface()
	jsonStr := fmt.Sprintf(`{"%s": %s}`, fd.Name(), valStr)
	if err := protojson.Unmarshal([]byte(jsonStr), msg); err != nil {
		return fmt.Errorf("field %s: failed to parse default_value as JSON object: %w", fd.FullName(), err)
	}

	srcMap := msg.ProtoReflect().Get(fd).Map()
	var err error
	srcMap.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
		mp.Set(k, v)
		// After setting, if it's a message, apply defaults
		if fd.MapValue().Kind() == protoreflect.MessageKind {
			if e := applyMessage(v.Message(), make(map[protoreflect.FullName]bool)); e != nil {
				err = e
				return false
			}
		}
		return true
	})

	return err
}
