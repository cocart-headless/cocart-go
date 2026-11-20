package cocart

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// ObjectMap is a JSON object keyed by string, such as cart fees keyed by fee
// ID or product attributes keyed by attribute name.
//
// CoCart Starter 5 sends an empty map as {}; CoCart 4.x sent []. Both decode
// to an empty map. A non-empty array decodes with its indexes as keys
// ("0", "1", …).
type ObjectMap[V any] map[string]V

// UnmarshalJSON accepts a JSON object, an array or null.
func (m *ObjectMap[V]) UnmarshalJSON(b []byte) error {
	trimmed := bytes.TrimSpace(b)
	switch {
	case bytes.Equal(trimmed, []byte("null")):
		*m = nil
		return nil
	case len(trimmed) > 0 && trimmed[0] == '[':
		var list []V
		if err := json.Unmarshal(trimmed, &list); err != nil {
			return err
		}
		out := make(ObjectMap[V], len(list))
		for i, v := range list {
			out[strconv.Itoa(i)] = v
		}
		*m = out
		return nil
	}
	var plain map[string]V
	if err := json.Unmarshal(trimmed, &plain); err != nil {
		return err
	}
	*m = plain
	return nil
}

// FlexString holds a value CoCart sends as a string in some responses and a
// number in others, such as the cart's items_weight or a product's
// total_sales. A number keeps its JSON spelling ("25"); null is "".
type FlexString string

// UnmarshalJSON accepts a JSON string, number, boolean or null.
func (s *FlexString) UnmarshalJSON(b []byte) error {
	trimmed := bytes.TrimSpace(b)
	switch {
	case bytes.Equal(trimmed, []byte("null")):
		*s = ""
	case len(trimmed) > 0 && trimmed[0] == '"':
		var v string
		if err := json.Unmarshal(trimmed, &v); err != nil {
			return err
		}
		*s = FlexString(v)
	case len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '['):
		return fmt.Errorf("cocart: cannot decode %s into FlexString", trimmed[:1])
	default:
		*s = FlexString(trimmed)
	}
	return nil
}

// String returns the value as a string.
func (s FlexString) String() string {
	return string(s)
}

// orderedValues returns the values of a JSON object in the order they were
// sent, or the elements of a JSON array. Anything else gives no values.
func orderedValues(raw json.RawMessage) ([]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, nil
	}
	switch trimmed[0] {
	case '[':
		var list []json.RawMessage
		err := json.Unmarshal(trimmed, &list)
		return list, err
	case '{':
		dec := json.NewDecoder(bytes.NewReader(trimmed))
		if _, err := dec.Token(); err != nil { // {
			return nil, err
		}
		var values []json.RawMessage
		for dec.More() {
			if _, err := dec.Token(); err != nil { // key
				return nil, err
			}
			var v json.RawMessage
			if err := dec.Decode(&v); err != nil {
				return nil, err
			}
			values = append(values, v)
		}
		return values, nil
	}
	return nil, nil
}

// decodeEach decodes each raw value into T, skipping values that don't fit.
func decodeEach[T any](values []json.RawMessage) []T {
	out := make([]T, 0, len(values))
	for _, v := range values {
		var item T
		if err := json.Unmarshal(v, &item); err == nil {
			out = append(out, item)
		}
	}
	return out
}

// isJSONObject reports whether raw holds a JSON object.
func isJSONObject(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && trimmed[0] == '{'
}

// decodeObjectOrEmpty decodes raw into v unless it's an array (CoCart 4.x
// sends [] for an empty object) or null, which leave v as it is.
func decodeObjectOrEmpty(raw []byte, v any) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] == '[' || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}
	return json.Unmarshal(trimmed, v)
}
