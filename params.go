package cocart

import (
	"fmt"
	"reflect"
	"strings"
)

// structToParamsReflect uses reflection to convert a struct with `url` tags
// to a map[string]string for query parameters.
func structToParamsReflect(v any) map[string]string {
	result := make(map[string]string)
	if v == nil {
		return result
	}

	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return result
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return result
	}

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		tag := field.Tag.Get("url")
		if tag == "" || tag == "-" {
			continue
		}

		parts := strings.SplitN(tag, ",", 2)
		name := parts[0]
		omitempty := len(parts) > 1 && parts[1] == "omitempty"

		if omitempty && isZeroValue(fieldVal) {
			continue
		}

		str := formatValue(fieldVal)
		if str != "" {
			result[name] = str
		}
	}

	return result
}

func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	default:
		return false
	}
}

func formatValue(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return ""
		}
		return formatValue(v.Elem())
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%g", v.Float())
	case reflect.Bool:
		if v.Bool() {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

// mergeMaps merges multiple maps into one. Later maps override earlier ones.
func mergeMaps(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}
