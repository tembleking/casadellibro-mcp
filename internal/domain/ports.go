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
}

// CatalogRepository searches the casadellibro catalog.
type CatalogRepository interface {
	Search(ctx context.Context, q SearchQuery) (SearchResult, error)
}

// StockRepository reports per-store stock for a product.
type StockRepository interface {
	StockByStore(ctx context.Context, productID string, countryCache int) ([]Province, error)
}

//go:generate go run go.uber.org/mock/mockgen -destination=../mocks/repositories.go -package=mocks app/internal/domain CatalogRepository,StockRepository
