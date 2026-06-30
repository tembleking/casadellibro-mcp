package mcp

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"app/internal/domain"
)

// The tool results are rendered as tab-separated tables instead of JSON: the
// field names appear once in a header row rather than being repeated on every
// item, which saves a large amount of tokens for the consuming LLM.

// renderSearch renders a search result as a summary line followed by a TSV table
// whose columns are the requested fields, one row per book.
func renderSearch(res domain.SearchResult, fields []string) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "total=%d start=%d rows=%d\n", res.Total, res.Start, res.Rows)
	b.WriteString(strings.Join(fields, "\t"))
	for i := range res.Books {
		m, err := projectItem(res.Books[i], fields)
		if err != nil {
			return "", err
		}
		b.WriteByte('\n')
		b.WriteString(rowString(m, fields))
	}
	return b.String(), nil
}

// renderStock renders per-store stock as a TSV table with a leading "province"
// column so every bookstore stays attributed to its province in a single table.
func renderStock(provinces []domain.Province, fields []string) (string, error) {
	var b strings.Builder
	b.WriteString("province\t")
	b.WriteString(strings.Join(fields, "\t"))
	for _, p := range provinces {
		for i := range p.Bookstores {
			m, err := projectItem(p.Bookstores[i], fields)
			if err != nil {
				return "", err
			}
			b.WriteByte('\n')
			b.WriteString(sanitize(p.Name))
			b.WriteByte('\t')
			b.WriteString(rowString(m, fields))
		}
	}
	return b.String(), nil
}

// renderFacets renders the available filters grouped by facet, each value as
// "<filter string>\t(<count>)" so the filter string can be copied verbatim into
// the search_books filters argument.
func renderFacets(facets []domain.Facet) string {
	var b strings.Builder
	for i, f := range facets {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "# %s [%s]", f.Label, f.Type)
		for _, v := range f.Values {
			fmt.Fprintf(&b, "\n%s\t(%d)", v.Filter, v.Count)
		}
	}
	return b.String()
}

// rowString joins the requested fields of a projected item into a TSV row.
func rowString(m map[string]json.RawMessage, fields []string) string {
	cells := make([]string, len(fields))
	for i, f := range fields {
		cells[i] = cellString(m[f])
	}
	return strings.Join(cells, "\t")
}

// cellString turns a single JSON value into a one-line cell. Arrays are joined
// with "; ", objects fall back to compact JSON, and tabs/newlines are stripped
// so a value can never break the table layout.
func cellString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return sanitize(string(raw))
	}
	return formatValue(v)
}

func formatValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return sanitize(t)
	case bool:
		return strconv.FormatBool(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case []any:
		parts := make([]string, 0, len(t))
		for _, e := range t {
			parts = append(parts, formatValue(e))
		}
		return strings.Join(parts, "; ")
	default:
		b, _ := json.Marshal(v)
		return sanitize(string(b))
	}
}

// sanitize collapses tab and newline characters to spaces to keep one item per row.
func sanitize(s string) string {
	return strings.NewReplacer("\t", " ", "\r", " ", "\n", " ").Replace(s)
}
