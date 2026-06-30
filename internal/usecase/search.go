package usecase

import (
	"context"
	"errors"
	"strings"

	"app/internal/domain"
)

// ErrEmptyQuery is returned when a search is requested with no query text.
var ErrEmptyQuery = errors.New("query must not be empty")

// Defaults applied when a search parameter is left unset.
const (
	defaultRows     = 16
	defaultStore    = "ES"
	defaultLang     = "es"
	defaultCurrency = "EUR"
	maxRows         = 100
)

// SearchBooks is the use case that searches the catalog.
type SearchBooks struct {
	repo domain.CatalogRepository
}

// NewSearchBooks wires the use case with its repository.
func NewSearchBooks(repo domain.CatalogRepository) *SearchBooks {
	return &SearchBooks{repo: repo}
}

// Execute validates the query, applies defaults and delegates to the repository.
func (uc *SearchBooks) Execute(ctx context.Context, q domain.SearchQuery) (domain.SearchResult, error) {
	q.Query = strings.TrimSpace(q.Query)
	if q.Query == "" {
		return domain.SearchResult{}, ErrEmptyQuery
	}

	if q.Start < 0 {
		q.Start = 0
	}
	switch {
	case q.Rows <= 0:
		q.Rows = defaultRows
	case q.Rows > maxRows:
		q.Rows = maxRows
	}
	if q.Store == "" {
		q.Store = defaultStore
	}
	if q.Lang == "" {
		q.Lang = defaultLang
	}
	if q.Currency == "" {
		q.Currency = defaultCurrency
	}

	return uc.repo.Search(ctx, q)
}

// ListSearchFilters is the use case that reports the filters available for a query.
type ListSearchFilters struct {
	repo domain.CatalogRepository
}

// NewListSearchFilters wires the use case with its repository.
func NewListSearchFilters(repo domain.CatalogRepository) *ListSearchFilters {
	return &ListSearchFilters{repo: repo}
}

// Execute validates the query, applies defaults and delegates to the repository.
func (uc *ListSearchFilters) Execute(ctx context.Context, q domain.FacetQuery) ([]domain.Facet, error) {
	q.Query = strings.TrimSpace(q.Query)
	if q.Query == "" {
		return nil, ErrEmptyQuery
	}
	if q.Store == "" {
		q.Store = defaultStore
	}
	if q.Lang == "" {
		q.Lang = defaultLang
	}
	if q.Currency == "" {
		q.Currency = defaultCurrency
	}

	return uc.repo.Facets(ctx, q)
}
