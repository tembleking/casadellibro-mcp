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

// Execute validates the product id, applies defaults and delegates to the repository.
func (uc *GetStoreStock) Execute(ctx context.Context, productID string, countryCache int) ([]domain.Province, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, ErrEmptyProductID
	}
	if countryCache <= 0 {
		countryCache = defaultCountryCache
	}

	return uc.repo.StockByStore(ctx, productID, countryCache)
}
