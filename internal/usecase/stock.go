package usecase

import (
	"context"
	"errors"
	"strings"

	"app/internal/domain"
)

// ErrEmptyProductID is returned when no product id is supplied.
var ErrEmptyProductID = errors.New("product id must not be empty")

// defaultCountryCache is the "paiscache" value used by casadellibro for Spain.
const defaultCountryCache = 63

// GetStoreStock is the use case that reports per-store stock for a product.
type GetStoreStock struct {
	repo domain.StockRepository
}

// NewGetStoreStock wires the use case with its repository.
func NewGetStoreStock(repo domain.StockRepository) *GetStoreStock {
	return &GetStoreStock{repo: repo}
}

// StockQuery are the parameters of a per-store stock lookup.
type StockQuery struct {
	ProductID    string
	CountryCache int
	// StoreID, when > 0, restricts the result to that single store.
	StoreID int
	// InStockOnly drops bookstores with zero stock.
	InStockOnly bool
}

// Execute validates the product id, applies defaults, fetches the stock and
// applies the optional store/in-stock filters.
func (uc *GetStoreStock) Execute(ctx context.Context, q StockQuery) ([]domain.Province, error) {
	q.ProductID = strings.TrimSpace(q.ProductID)
	if q.ProductID == "" {
		return nil, ErrEmptyProductID
	}
	if q.CountryCache <= 0 {
		q.CountryCache = defaultCountryCache
	}

	provinces, err := uc.repo.StockByStore(ctx, q.ProductID, q.CountryCache)
	if err != nil {
		return nil, err
	}
	return filterProvinces(provinces, q.StoreID, q.InStockOnly), nil
}

// filterProvinces keeps only the bookstores matching the store/in-stock filters,
// dropping provinces left with no bookstores. Zero-value filters are no-ops.
func filterProvinces(provinces []domain.Province, storeID int, inStockOnly bool) []domain.Province {
	if storeID <= 0 && !inStockOnly {
		return provinces
	}
	out := make([]domain.Province, 0, len(provinces))
	for _, p := range provinces {
		kept := make([]domain.Bookstore, 0, len(p.Bookstores))
		for _, b := range p.Bookstores {
			if storeID > 0 && b.StoreID != storeID {
				continue
			}
			if inStockOnly && b.Stock <= 0 {
				continue
			}
			kept = append(kept, b)
		}
		if len(kept) > 0 {
			out = append(out, domain.Province{Name: p.Name, Bookstores: kept})
		}
	}
	return out
}
