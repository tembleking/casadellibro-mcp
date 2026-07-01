package mcp

import "encoding/json"

// JSON Schema output descriptions for the tools. Because callers project the
// response down to a subset of fields, every item property is declared optional
// (no "required"): whichever fields were requested will be present, the rest
// omitted. Types mirror the domain entities.

// jsonType maps each entity field to its JSON Schema primitive type. "array"
// entries are string arrays (only authors is one today).
var (
	bookFieldTypes = map[string]string{
		"id": "string", "product_id": "string", "name": "string", "authors": "array",
		"isbn": "string", "ean": "string", "editorial": "string", "product_type": "string",
		"year": "string", "price": "number", "price_previous": "number",
		"availability": "string", "url": "string", "image_url": "string", "description": "string",
	}
	bookstoreFieldTypes = map[string]string{
		"store_id": "integer", "city": "string", "address": "string", "phone": "string",
		"email": "string", "schedule": "string", "stock": "integer", "availability": "string",
		"postal_code": "string", "latitude": "number", "longitude": "number", "map_url": "string",
	}
)

// itemSchema builds an object schema whose properties are the given fields.
func itemSchema(types map[string]string) map[string]any {
	props := make(map[string]any, len(types))
	for name, t := range types {
		if t == "array" {
			props[name] = map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
			continue
		}
		props[name] = map[string]any{"type": t}
	}
	return map[string]any{"type": "object", "properties": props}
}

func mustSchema(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err) // schemas are static; a failure here is a programming error.
	}
	return b
}

var (
	searchOutputSchema = mustSchema(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"total": map[string]any{"type": "integer"},
			"start": map[string]any{"type": "integer"},
			"rows":  map[string]any{"type": "integer"},
			"books": map[string]any{"type": "array", "items": itemSchema(bookFieldTypes)},
		},
	})

	// structuredContent must be a JSON object, so the lists are wrapped.
	stockOutputSchema = mustSchema(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"provinces": map[string]any{"type": "array", "items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":       map[string]any{"type": "string"},
					"bookstores": map[string]any{"type": "array", "items": itemSchema(bookstoreFieldTypes)},
				},
			}},
		},
	})

	facetsOutputSchema = mustSchema(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"facets": map[string]any{"type": "array", "items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"label": map[string]any{"type": "string"},
					"type":  map[string]any{"type": "string"},
					"values": map[string]any{"type": "array", "items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"value":  map[string]any{"type": "string"},
							"count":  map[string]any{"type": "integer"},
							"filter": map[string]any{"type": "string"},
						},
					}},
				},
			}},
		},
	})
)
