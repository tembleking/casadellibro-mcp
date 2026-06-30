package mcp

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// jsonFieldNames returns the json field names of the struct type of v, in
// declaration order, skipping fields tagged "-". It is the single source of
// truth for which fields a caller may request.
func jsonFieldNames(v any) []string {
	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	names := make([]string, 0, t.NumField())
	for i := range t.NumField() {
		tag := t.Field(i).Tag.Get("json")
		name, _, _ := strings.Cut(tag, ",")
		if name == "" || name == "-" {
			continue
		}
		names = append(names, name)
	}
	return names
}

// validateFields ensures every requested field is a valid json field of the
// allowed set, returning a helpful error otherwise.
func validateFields(requested, allowed []string) error {
	set := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		set[a] = true
	}
	for _, f := range requested {
		if !set[f] {
			return fmt.Errorf("unknown field %q; valid fields: %s", f, strings.Join(allowed, ", "))
		}
	}
	return nil
}

// projectItem marshals v and keeps only the requested json keys. When fields is
// empty all keys are kept.
func projectItem(v any, fields []string) (map[string]json.RawMessage, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		return m, nil
	}
	out := make(map[string]json.RawMessage, len(fields))
	for _, f := range fields {
		if raw, ok := m[f]; ok {
			out[f] = raw
		}
	}
	return out, nil
}

// projectItems applies projectItem to each element of a slice.
func projectItems[T any](items []T, fields []string) ([]map[string]json.RawMessage, error) {
	out := make([]map[string]json.RawMessage, 0, len(items))
	for i := range items {
		m, err := projectItem(items[i], fields)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

// fieldsDescription builds a tool-parameter description listing the valid fields.
func fieldsDescription(itemLabel string, allowed []string) string {
	return fmt.Sprintf(
		"Optional list of %s fields to return; omit to return all (uses more tokens). Valid fields: %s.",
		itemLabel, strings.Join(allowed, ", "),
	)
}
