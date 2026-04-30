package protodefault

import (
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	defaultsv1 "github.com/yurasolovjov/protodefault/proto/defaults/v1"
)

// Apply рекурсивно проставляет default values из (defaults.v1.default_value)
// во всех полях m и его вложенных message, у которых поле не задано.
func Apply(m proto.Message) error {
	if m == nil {
		return fmt.Errorf("message is nil")
	}
	visited := make(map[protoreflect.FullName]bool)
	return applyMessage(m.ProtoReflect(), visited)
}

// MustApply — как Apply, но паникует при ошибке.
func MustApply(m proto.Message) {
	if err := Apply(m); err != nil {
		panic(err)
	}
}

func applyMessage(m protoreflect.Message, visited map[protoreflect.FullName]bool) error {
	md := m.Descriptor()

	// Обработка Well-Known Types (WKT)
	if isWKT(md.FullName()) {
		// WKT обрабатываются как специальные типы, но если они сами являются полями другого сообщения,
		// их обработка происходит в applyField. Здесь мы можем оказаться если Apply вызван напрямую для WKT.
		// Большинство WKT не имеют полей с дефолтами внутри себя в смысле нашей логики.
		return nil
	}

	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if err := applyField(m, fd, visited); err != nil {
			return err
		}
	}

	return nil
}

func applyField(m protoreflect.Message, fd protoreflect.FieldDescriptor, visited map[protoreflect.FullName]bool) error {
	// 1. Проверяем наличие дефолта в опциях
	defaultValueStr := ""
	if proto.HasExtension(fd.Options(), defaultsv1.E_DefaultValue) {
		defaultValueStr = proto.GetExtension(fd.Options(), defaultsv1.E_DefaultValue).(string)
	}

	// 2. Обработка oneof
	if od := fd.ContainingOneof(); od != nil {
		// handleOneOf only handles selection and recursion.
		// It doesn't need to be called for every field in oneof, just once per oneof.
		// protoreflect.Message.WhichOneof(od) helps.
		// But we are iterating over all fields.
		// To avoid multiple calls, we only call it for the first field of the oneof.
		if od.Fields().Get(0) == fd {
			return handleOneOf(m, od, visited)
		}
		return nil
	}

	// 3. Если это сообщение (не repeated и не map), идем вглубь или инициализируем
	if fd.Kind() == protoreflect.MessageKind && !fd.IsList() && !fd.IsMap() {
		if isWKT(fd.Message().FullName()) {
			// Fallthrough to step 4, handling WKT as scalars with setWKTDefault
		} else {
			return handleMessageField(m, fd, defaultValueStr, visited)
		}
	}

	// 4. Если значения нет в опции, и это не сообщение (где нужна рекурсия), ничего не делаем
	if defaultValueStr == "" {
		return nil
	}

	// 5. Проверяем, задано ли поле
	if isSet(m, fd) {
		// Если поле задано и это repeated message или map message, нужно пройти по элементам
		return handleRecursionOnSetField(m, fd, visited)
	}

	// 6. Применяем дефолт
	return setFieldDefault(m, fd, defaultValueStr)
}

func isSet(m protoreflect.Message, fd protoreflect.FieldDescriptor) bool {
	if fd.HasPresence() {
		return m.Has(fd)
	}
	if fd.IsList() {
		return m.Get(fd).List().Len() > 0
	}
	if fd.IsMap() {
		return m.Get(fd).Map().Len() > 0
	}
	// proto3 scalar without presence
	return !isZeroValue(m.Get(fd), fd)
}

func isZeroValue(v protoreflect.Value, fd protoreflect.FieldDescriptor) bool {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		return !v.Bool()
	case protoreflect.StringKind:
		return v.String() == ""
	case protoreflect.BytesKind:
		return len(v.Bytes()) == 0
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return v.Int() == 0
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return v.Uint() == 0
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return v.Float() == 0
	case protoreflect.EnumKind:
		return v.Enum() == 0
	}
	return false
}

func setFieldDefault(m protoreflect.Message, fd protoreflect.FieldDescriptor, valStr string) error {
	if fd.IsList() {
		return setRepeatedField(m, fd, valStr)
	}
	if fd.IsMap() {
		return setMapField(m, fd, valStr)
	}

	if fd.Kind() == protoreflect.MessageKind {
		if isWKT(fd.Message().FullName()) {
			return setWKTDefault(m, fd, valStr)
		}
		return fmt.Errorf("field %s: internal error: message kind without WKT handling", fd.FullName())
	}

	val, err := parseScalar(fd, valStr)
	if err != nil {
		return fmt.Errorf("field %s: %w", fd.FullName(), err)
	}
	m.Set(fd, val)
	return nil
}

func parseScalar(fd protoreflect.FieldDescriptor, s string) (protoreflect.Value, error) {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfBool(b), nil
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(s), nil
	case protoreflect.BytesKind:
		return protoreflect.ValueOfBytes([]byte(s)), nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		i, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt32(int32(i)), nil
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt64(i), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		i, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfUint32(uint32(i)), nil
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		i, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfUint64(i), nil
	case protoreflect.FloatKind:
		f, err := strconv.ParseFloat(s, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfFloat32(float32(f)), nil
	case protoreflect.DoubleKind:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfFloat64(f), nil
	case protoreflect.EnumKind:
		enumVal := fd.Enum().Values().ByName(protoreflect.Name(s))
		if enumVal != nil {
			return protoreflect.ValueOfEnum(enumVal.Number()), nil
		}
		// Try parse as number
		num, err := strconv.Atoi(s)
		if err == nil {
			enumVal = fd.Enum().Values().ByNumber(protoreflect.EnumNumber(num))
			if enumVal != nil {
				return protoreflect.ValueOfEnum(enumVal.Number()), nil
			}
		}
		var allowed []string
		for i := 0; i < fd.Enum().Values().Len(); i++ {
			allowed = append(allowed, string(fd.Enum().Values().Get(i).Name()))
		}
		return protoreflect.Value{}, fmt.Errorf("invalid enum value %q, allowed: %s", s, strings.Join(allowed, ", "))
	}
	return protoreflect.Value{}, fmt.Errorf("unsupported kind %s", fd.Kind())
}
