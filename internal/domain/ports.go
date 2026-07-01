package domain

import "context"

// SearchQuery are the parameters of a catalog search.
type SearchQuery struct {
	Query    string
	Start    int
	Rows     int
	Store    string
	Lang     string
	Currency string
	// Filters are raw facet filter strings (e.g. "availability:Con stock"),
	// as returned by Facets. Multiple filters combine with AND.
	Filters []string
}

// CatalogRepository searches the casadellibro catalog and reports the facets
// (available filters) for a query.
type CatalogRepository interface {
	Search(ctx context.Context, q SearchQuery) (SearchResult, error)
	Facets(ctx context.Context, q FacetQuery) ([]Facet, error)
}

// StockRepository reports per-store stock for a product and lists the store directory.
type StockRepository interface {
	StockByStore(ctx context.Context, productID string, countryCache int) ([]Province, error)
	Stores(ctx context.Context, countryCache int) ([]Store, error)
}

//go:generate go run go.uber.org/mock/mockgen -destination=../mocks/repositories.go -package=mocks app/internal/domain CatalogRepository,StockRepository
