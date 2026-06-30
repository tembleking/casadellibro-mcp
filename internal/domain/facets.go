package domain

// FacetQuery are the parameters used to discover the filters available for a
// given catalog search.
type FacetQuery struct {
	Query    string
	Store    string
	Lang     string
	Currency string
}

// FacetValue is a single selectable value of a facet. Filter is the exact
// string to pass back in SearchQuery.Filters to narrow a search by this value.
type FacetValue struct {
	Value  string `json:"value"`
	Count  int    `json:"count"`
	Filter string `json:"filter"`
}

// Facet is a group of related filters (e.g. language, binding, availability)
// available for a search, together with the values it can take.
type Facet struct {
	Label  string       `json:"label"`
	Type   string       `json:"type"`
	Values []FacetValue `json:"values"`
}
