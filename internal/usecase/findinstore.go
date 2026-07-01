package usecase

import (
	"context"
	"errors"
	"strings"
	"sync"

	"app/internal/domain"
)

// ErrNoStoreID is returned when a store-scoped search is requested without a store id.
var ErrNoStoreID = errors.New("store_id must be a positive store id")

// Scan/concurrency tuning for FindBooksInStore.
const (
	findScanPageSize   = 100
	defaultFindMaxScan = 120
	maxFindMaxScan     = 400
	findConcurrency    = 8
)

// BookInStore is a catalog book annotated with its stock at the requested store.
type BookInStore struct {
	domain.Book
	StoreStock        int    `json:"store_stock"`
	StoreAvailability string `json:"store_availability"`
}

// FindInStoreResult is the outcome of a store-scoped search.
type FindInStoreResult struct {
	Books     []BookInStore
	Scanned   int
	Total     int
	Truncated bool
}

// FindInStoreQuery are the parameters of a store-scoped search.
type FindInStoreQuery struct {
	Query        string
	Filters      []string
	StoreID      int
	CountryCache int
	// MaxScan caps how many catalog candidates are checked against the store.
	MaxScan int
}

// FindBooksInStore searches the catalog and returns only the books actually in
// stock at a specific physical store, joining search with per-store stock so a
// client does not have to fan out N stock calls itself.
type FindBooksInStore struct {
	search *SearchBooks
	stock  *GetStoreStock
}

// NewFindBooksInStore composes the search and stock use cases.
func NewFindBooksInStore(search *SearchBooks, stock *GetStoreStock) *FindBooksInStore {
	return &FindBooksInStore{search: search, stock: stock}
}

// Execute paginates the catalog (up to MaxScan candidates) and, concurrently,
// keeps the books with stock > 0 at StoreID.
func (uc *FindBooksInStore) Execute(ctx context.Context, q FindInStoreQuery) (FindInStoreResult, error) {
	q.Query = strings.TrimSpace(q.Query)
	if q.Query == "" {
		return FindInStoreResult{}, ErrEmptyQuery
	}
	if q.StoreID <= 0 {
		return FindInStoreResult{}, ErrNoStoreID
	}
	switch {
	case q.MaxScan <= 0:
		q.MaxScan = defaultFindMaxScan
	case q.MaxScan > maxFindMaxScan:
		q.MaxScan = maxFindMaxScan
	}

	candidates, total, err := uc.gatherCandidates(ctx, q)
	if err != nil {
		return FindInStoreResult{}, err
	}

	found := uc.checkStock(ctx, candidates, q)

	return FindInStoreResult{
		Books:     found,
		Scanned:   len(candidates),
		Total:     total,
		Truncated: total > len(candidates),
	}, nil
}

// gatherCandidates paginates the search until MaxScan candidates or the last page.
func (uc *FindBooksInStore) gatherCandidates(ctx context.Context, q FindInStoreQuery) ([]domain.Book, int, error) {
	var candidates []domain.Book
	total := 0
	for start := 0; len(candidates) < q.MaxScan; start += findScanPageSize {
		res, err := uc.search.Execute(ctx, domain.SearchQuery{
			Query:   q.Query,
			Filters: q.Filters,
			Start:   start,
			Rows:    findScanPageSize,
		})
		if err != nil {
			return nil, 0, err
		}
		total = res.Total
		if len(res.Books) == 0 {
			break
		}
		for _, b := range res.Books {
			candidates = append(candidates, b)
			if len(candidates) >= q.MaxScan {
				break
			}
		}
		if start+findScanPageSize >= res.Total {
			break
		}
	}
	return candidates, total, nil
}

// checkStock fans out per-store stock lookups over a bounded worker pool and
// returns, in catalog order, the books with stock at the store.
func (uc *FindBooksInStore) checkStock(ctx context.Context, candidates []domain.Book, q FindInStoreQuery) []BookInStore {
	results := make([]*BookInStore, len(candidates))
	sem := make(chan struct{}, findConcurrency)
	var wg sync.WaitGroup
	for i := range candidates {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			provinces, err := uc.stock.Execute(ctx, StockQuery{
				ProductID:    candidates[i].ProductID,
				CountryCache: q.CountryCache,
				StoreID:      q.StoreID,
				InStockOnly:  true,
			})
			if err != nil || len(provinces) == 0 || len(provinces[0].Bookstores) == 0 {
				return
			}
			bs := provinces[0].Bookstores[0]
			results[i] = &BookInStore{Book: candidates[i], StoreStock: bs.Stock, StoreAvailability: bs.Availability}
		}(i)
	}
	wg.Wait()

	found := make([]BookInStore, 0, len(results))
	for _, r := range results {
		if r != nil {
			found = append(found, *r)
		}
	}
	return found
}
