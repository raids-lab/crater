package util

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
	timeDurationType    = reflect.TypeOf(time.Duration(0))
)

//nolint:gocyclo // basic utility function
func MapToStruct(m map[string]string, ptr any) error {
	v := reflect.ValueOf(ptr)
	if !v.IsValid() || v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("dst must be a non-nil pointer to struct")
	}

	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("dst must point to struct")
	}

	t := elem.Type()
	errs := make([]error, 0)
	for i := range t.NumField() {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}

		fieldValue := elem.Field(i)
		tags := []string{field.Tag.Get("json"), field.Tag.Get("yaml"), field.Tag.Get("mapstructure")}
		keyNames := []string{field.Name}
		for _, tag := range tags {
			if tag == "" || tag == "-" {
				continue
			}
			name, _, _ := strings.Cut(tag, ",")
			if name == "" || name == "-" {
				continue
			}
			name = strings.TrimSpace(name)
			keyNames = []string{name}
		}
		if keyNames[0] == field.Name {
			lowerName := lowerFirst(field.Name)
			if lowerName != field.Name {
				keyNames = append(keyNames, lowerName)
			}
		}

		var s string
		var keyName string
		var ok bool
		for _, candidate := range keyNames {
			if s, ok = m[candidate]; ok {
				keyName = candidate
				break
			}
		}
		if !ok {
			continue
		}

		if err := stringToField(fieldValue, field.Type, s); err != nil {
			errs = append(errs, fmt.Errorf("invalid value for key %q (field %s): %w", keyName, field.Name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%v", errs)
	}

	return nil
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

//nolint:gocyclo // basic utility function
func stringToField(fieldValue reflect.Value, fieldType reflect.Type, str string) error {
	if fieldValue.Kind() == reflect.Ptr {
		elemPtr := reflect.New(fieldType.Elem())
		if err := stringToField(elemPtr.Elem(), fieldType.Elem(), str); err != nil {
			return err
		}
		fieldValue.Set(elemPtr)
		return nil
	}

	if reflect.PointerTo(fieldType).Implements(textUnmarshalerType) {
		ptr := reflect.New(fieldType)
		if err := ptr.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(str)); err != nil {
			return err
		}
		fieldValue.Set(ptr.Elem())
		return nil
	}

	if fieldType == timeDurationType {
		d, err := time.ParseDuration(str)
		if err != nil {
			seconds, secErr := strconv.ParseInt(str, 10, 64)
			if secErr != nil {
				return err
			}
			d = time.Duration(seconds) * time.Second
		}
		fieldValue.SetInt(int64(d))
		return nil
	}

	switch fieldValue.Kind() {
	case reflect.String:
		fieldValue.SetString(str)
		return nil
	case reflect.Bool:
		parsed, err := strconv.ParseBool(str)
		if err != nil {
			return err
		}
		fieldValue.SetBool(parsed)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(str, 10, fieldType.Bits())
		if err != nil {
			return err
		}
		fieldValue.SetInt(parsed)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		parsed, err := strconv.ParseUint(str, 10, fieldType.Bits())
		if err != nil {
			return err
		}
		fieldValue.SetUint(parsed)
		return nil
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(str, fieldType.Bits())
		if err != nil {
			return err
		}
		fieldValue.SetFloat(parsed)
		return nil
	}

	return fmt.Errorf("unsupported field type %s", fieldType.String())
}
