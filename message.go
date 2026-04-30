package protodefault

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	defaultsv1 "github.com/yurasolovjov/protodefault/proto/defaults/v1"
)

func handleOneOf(m protoreflect.Message, od protoreflect.OneofDescriptor, visited map[protoreflect.FullName]bool) error {
	existing := m.WhichOneof(od)
	if existing != nil {
		// Field already selected, recurse if it's a message
		fd := existing
		if fd.Kind() == protoreflect.MessageKind && !fd.IsList() && !fd.IsMap() {
			val := m.Get(fd)
			if val.Message().IsValid() {
				return applyMessage(val.Message(), visited)
			}
		}
		return nil
	}

	// No field selected, look for default_value
	var defaultField protoreflect.FieldDescriptor
	for i := 0; i < od.Fields().Len(); i++ {
		fd := od.Fields().Get(i)
		opts := proto.GetExtension(fd.Options(), defaultsv1.E_DefaultValue).(string)
		if opts != "" {
			if defaultField != nil {
				return fmt.Errorf("oneof %s has multiple fields with default_value: %s and %s",
					od.FullName(), defaultField.FullName(), fd.FullName())
			}
			defaultField = fd
		}
	}

	if defaultField != nil {
		opts := proto.GetExtension(defaultField.Options(), defaultsv1.E_DefaultValue).(string)
		if err := setFieldDefault(m, defaultField, opts); err != nil {
			return err
		}
		// After setting default, if it's a message, recurse
		val := m.Get(defaultField)
		if defaultField.Kind() == protoreflect.MessageKind && val.Message().IsValid() {
			return applyMessage(val.Message(), visited)
		}
	}

	return nil
}

func handleMessageField(m protoreflect.Message, fd protoreflect.FieldDescriptor, defaultValueStr string, visited map[protoreflect.FullName]bool) error {
	md := fd.Message()

	// Check for cycles in schema
	if visited[md.FullName()] {
		return nil
	}

	if !m.Has(fd) {
		// Field not set. Should we initialize it?
		// Only if it has a default_value OR contains fields with default_value.
		hasDefaults, err := hasDefaultsDeep(md, make(map[protoreflect.FullName]bool))
		if err != nil {
			return err
		}

		if defaultValueStr != "" || hasDefaults {
			// Initialize empty message
			m.Set(fd, m.NewField(fd))
		} else {
			return nil
		}
	}

	// Recurse
	visited[md.FullName()] = true
	defer delete(visited, md.FullName())
	return applyMessage(m.Get(fd).Message(), visited)
}

func hasDefaultsDeep(md protoreflect.MessageDescriptor, visited map[protoreflect.FullName]bool) (bool, error) {
	if visited[md.FullName()] {
		return false, nil
	}
	visited[md.FullName()] = true

	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		opts := proto.GetExtension(fd.Options(), defaultsv1.E_DefaultValue).(string)
		if opts != "" {
			return true, nil
		}
		if fd.Kind() == protoreflect.MessageKind && !fd.IsList() && !fd.IsMap() {
			innerHas, err := hasDefaultsDeep(fd.Message(), visited)
			if err != nil {
				return false, err
			}
			if innerHas {
				return true, nil
			}
		}
	}
	return false, nil
}

func handleRecursionOnSetField(m protoreflect.Message, fd protoreflect.FieldDescriptor, visited map[protoreflect.FullName]bool) error {
	if fd.IsList() {
		if fd.Kind() == protoreflect.MessageKind {
			list := m.Get(fd).List()
			for i := 0; i < list.Len(); i++ {
				if err := applyMessage(list.Get(i).Message(), visited); err != nil {
					return err
				}
			}
		}
	} else if fd.IsMap() {
		if fd.MapValue().Kind() == protoreflect.MessageKind {
			mp := m.Get(fd).Map()
			var err error
			mp.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
				if e := applyMessage(v.Message(), visited); e != nil {
					err = e
					return false
				}
				return true
			})
			return err
		}
	}
	return nil
}
