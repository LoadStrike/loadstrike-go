package loadstrike

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"runtime"
	"strings"
)

const runtimeServiceBaseURL = "https://licensing.loadstrike.com"

const runtimeSettingsMaximumDepth = 128

func runtimeGOOS() string {
	return runtime.GOOS
}

func runtimeGOARCH() string {
	return runtime.GOARCH
}

type runtimeSettingsVisit struct {
	kind     reflect.Kind
	typeKey  reflect.Type
	pointer  uintptr
	length   int
	capacity int
}

// cloneJSONCompatibleSettings validates settings before any custom marshaler
// can run, then uses encoding/json as the canonical snapshot boundary.
func cloneJSONCompatibleSettings(source map[string]any) (map[string]any, error) {
	if source == nil {
		return map[string]any{}, nil
	}
	if err := validateJSONCompatibleSettingsValue(
		reflect.ValueOf(source),
		map[runtimeSettingsVisit]struct{}{},
		"settings",
		0,
	); err != nil {
		return nil, err
	}

	encoded, err := json.Marshal(source)
	if err != nil {
		return nil, errorsForRuntimeSettings(fmt.Sprintf("settings is not JSON-compatible: %v", err))
	}
	var snapshot map[string]any
	if err := json.Unmarshal(encoded, &snapshot); err != nil {
		return nil, errorsForRuntimeSettings(fmt.Sprintf("settings is not JSON-compatible: %v", err))
	}
	if snapshot == nil {
		snapshot = map[string]any{}
	}
	return snapshot, nil
}

func cloneKnownJSONCompatibleSettings(source map[string]any) map[string]any {
	cloned, err := cloneJSONCompatibleSettings(source)
	if err != nil {
		panic(fmt.Sprintf("clone validated runtime settings: %v", err))
	}
	return cloned
}

func validateJSONCompatibleSettingsValue(
	value reflect.Value,
	active map[runtimeSettingsVisit]struct{},
	path string,
	depth int,
) error {
	if depth > runtimeSettingsMaximumDepth {
		return errorsForRuntimeSettings(fmt.Sprintf(
			"%s exceeds the maximum nesting depth of %d",
			path,
			runtimeSettingsMaximumDepth,
		))
	}
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return nil
		}
		return validateJSONCompatibleSettingsValue(value.Elem(), active, path, depth)
	}

	switch value.Kind() {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return nil
	case reflect.Float32, reflect.Float64:
		floatValue := value.Float()
		if math.IsNaN(floatValue) || math.IsInf(floatValue, 0) {
			return errorsForRuntimeSettings(path + " contains a non-finite number")
		}
		return nil
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return errorsForRuntimeSettings(path + " contains a map with non-string keys")
		}
		if value.IsNil() {
			return nil
		}
		visit, err := enterRuntimeSettingsContainer(value, active, path)
		if err != nil {
			return err
		}
		defer delete(active, visit)

		iterator := value.MapRange()
		for iterator.Next() {
			key := iterator.Key().String()
			if err := validateJSONCompatibleSettingsValue(
				iterator.Value(),
				active,
				runtimeSettingsPath(path, key),
				depth+1,
			); err != nil {
				return err
			}
		}
		return nil
	case reflect.Slice:
		if value.IsNil() {
			return nil
		}
		visit, err := enterRuntimeSettingsContainer(value, active, path)
		if err != nil {
			return err
		}
		defer delete(active, visit)
		fallthrough
	case reflect.Array:
		for index := 0; index < value.Len(); index++ {
			if err := validateJSONCompatibleSettingsValue(
				value.Index(index),
				active,
				fmt.Sprintf("%s[%d]", path, index),
				depth+1,
			); err != nil {
				return err
			}
		}
		return nil
	case reflect.Pointer:
		if value.IsNil() {
			return nil
		}
		visit, err := enterRuntimeSettingsContainer(value, active, path)
		if err != nil {
			return err
		}
		defer delete(active, visit)
		return validateJSONCompatibleSettingsValue(value.Elem(), active, path, depth+1)
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			fieldType := value.Type().Field(index)
			if !runtimeSettingsStructFieldVisible(fieldType) {
				continue
			}
			if err := validateJSONCompatibleSettingsValue(
				value.Field(index),
				active,
				runtimeSettingsPath(path, fieldType.Name),
				depth+1,
			); err != nil {
				return err
			}
		}
		return nil
	case reflect.Invalid:
		return nil
	case reflect.Chan, reflect.Func, reflect.Complex64, reflect.Complex128, reflect.UnsafePointer:
		return errorsForRuntimeSettings(fmt.Sprintf("%s contains unsupported %s value", path, value.Kind()))
	default:
		return errorsForRuntimeSettings(fmt.Sprintf("%s contains unsupported %s value", path, value.Kind()))
	}
}

func runtimeSettingsStructFieldVisible(field reflect.StructField) bool {
	if field.Tag.Get("json") == "-" {
		return false
	}
	if field.PkgPath == "" {
		return true
	}
	if !field.Anonymous {
		return false
	}
	embeddedType := field.Type
	if embeddedType.Kind() == reflect.Pointer {
		embeddedType = embeddedType.Elem()
	}
	return embeddedType.Kind() == reflect.Struct
}

func enterRuntimeSettingsContainer(
	value reflect.Value,
	active map[runtimeSettingsVisit]struct{},
	path string,
) (runtimeSettingsVisit, error) {
	visit := runtimeSettingsVisit{
		kind:     value.Kind(),
		typeKey:  value.Type(),
		pointer:  value.Pointer(),
		length:   -1,
		capacity: -1,
	}
	if value.Kind() == reflect.Slice {
		visit.length = value.Len()
		visit.capacity = value.Cap()
	}
	if _, found := active[visit]; found {
		return runtimeSettingsVisit{}, errorsForRuntimeSettings(path + " contains a reference cycle")
	}
	active[visit] = struct{}{}
	return visit, nil
}

func runtimeSettingsPath(parent string, key string) string {
	if strings.TrimSpace(parent) == "" {
		return key
	}
	return parent + "." + key
}

func errorsForRuntimeSettings(message string) error {
	return fmt.Errorf("invalid JSON-compatible settings: %s", message)
}
