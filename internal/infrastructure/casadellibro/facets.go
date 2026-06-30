package casadellibro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"app/internal/domain"
)

// maxFacetValues caps how many values per facet are surfaced, keeping the tool
// response small. The API already returns them ordered by descending count.
const maxFacetValues = 20

type facetsResponse struct {
	Catalog struct {
		Facets []facetItem `json:"facets"`
	} `json:"catalog"`
}

type facetItem struct {
	Facet  string       `json:"facet"`
	Label  string       `json:"label"`
	Type   string       `json:"type"`
	Values []facetValue `json:"values"`
}

type facetValue struct {
	Value  string `json:"value"`
	Count  int    `json:"count"`
	Filter string `json:"filter"`
}

// Facets queries the empathy facets endpoint and reports the filters available
// for the given search, with the exact filter strings to feed back into Search.
func (a *CatalogAdapter) Facets(ctx context.Context, q domain.FacetQuery) ([]domain.Facet, error) {
	params := url.Values{}
	params.Set("query", q.Query)
	params.Set("lang", q.Lang)
	params.Set("currency", q.Currency)
	params.Set("store", q.Store)

	body, err := a.client.getJSON(ctx, a.client.facetsBaseURL, params)
	if err != nil {
		return nil, err
	}

	var resp facetsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode facets response: %w", err)
	}

	facets := make([]domain.Facet, 0, len(resp.Catalog.Facets))
	for _, f := range resp.Catalog.Facets {
		label := f.Label
		if label == "" {
			label = f.Facet
		}

		values := make([]domain.FacetValue, 0, len(f.Values))
		for _, v := range f.Values {
			if strings.TrimSpace(v.Value) == "" {
				continue
			}
			values = append(values, domain.FacetValue{
				Value:  v.Value,
				Count:  v.Count,
				Filter: v.Filter,
			})
			if len(values) == maxFacetValues {
				break
			}
		}
		if len(values) == 0 {
			continue
		}

		facets = append(facets, domain.Facet{
			Label:  label,
			Type:   f.Type,
			Values: values,
		})
	}

	return facets, nil
}
