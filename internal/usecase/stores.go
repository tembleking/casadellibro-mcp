package usecase

import (
	"context"
	"strings"

	"app/internal/domain"
)

// ListStores is the use case that lists the casadellibro store directory,
// optionally filtered by a free-text query.
type ListStores struct {
	repo domain.StockRepository
}

// NewListStores wires the use case with its repository.
func NewListStores(repo domain.StockRepository) *ListStores {
	return &ListStores{repo: repo}
}

// Execute lists all stores and, when query is non-empty, keeps only those whose
// province, city or address contains it (case-insensitive). This is how a
// human-friendly name like "grancasa" resolves to a store_id.
func (uc *ListStores) Execute(ctx context.Context, query string, countryCache int) ([]domain.Store, error) {
	if countryCache <= 0 {
		countryCache = defaultCountryCache
	}

	stores, err := uc.repo.Stores(ctx, countryCache)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(query) == "" {
		return stores, nil
	}
	// Space-insensitive so "grancasa" matches the address "C. C. Gran Casa".
	needle := normalizeStoreText(query)

	out := make([]domain.Store, 0, len(stores))
	for _, s := range stores {
		hay := normalizeStoreText(s.Province + s.City + s.Address)
		if strings.Contains(hay, needle) {
			out = append(out, s)
		}
	}
	return out, nil
}

// normalizeStoreText lowercases and strips whitespace for lenient matching.
func normalizeStoreText(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), "")
}
