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
	findScanPageSize = 100
	defaultFindLimit = 120
	maxFindLimit     = 400
	findConcurrency  = 8
)

// BookInStore is a catalog book annotated with its stock at the requested store.
type BookInStore struct {
	domain.Book
	StoreStock        int    `json:"store_stock"`
	StoreAvailability string `json:"store_availability"`
}

// FindInStoreResult is one page of a store-scoped search. Scanning is paginated
// over catalog candidates: resume the next page by passing NextStart as Start
// while HasMore is true.
type FindInStoreResult struct {
	Books     []BookInStore
	Start     int
	Scanned   int
	NextStart int
	Total     int
	HasMore   bool
}

// FindInStoreQuery are the parameters of a store-scoped search.
type FindInStoreQuery struct {
	Query        string
	Filters      []string
	StoreID      int
	CountryCache int
	// Start is the catalog offset to begin scanning from (0-based).
	Start int
	// Limit is how many catalog candidates to scan in this call (the page size).
	Limit int
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

// Execute scans one page of catalog candidates (Limit entries from Start) and,
// concurrently, keeps the books with stock > 0 at StoreID.
func (uc *FindBooksInStore) Execute(ctx context.Context, q FindInStoreQuery) (FindInStoreResult, error) {
	q.Query = strings.TrimSpace(q.Query)
	if q.Query == "" {
		return FindInStoreResult{}, ErrEmptyQuery
	}
	if q.StoreID <= 0 {
		return FindInStoreResult{}, ErrNoStoreID
	}
	if q.Start < 0 {
		q.Start = 0
	}
	switch {
	case q.Limit <= 0:
		q.Limit = defaultFindLimit
	case q.Limit > maxFindLimit:
		q.Limit = maxFindLimit
	}

	candidates, consumed, total, err := uc.gatherCandidates(ctx, q)
	if err != nil {
		return FindInStoreResult{}, err
	}

	found := uc.checkStock(ctx, candidates, q)

	nextStart := q.Start + consumed
	return FindInStoreResult{
		Books:     found,
		Start:     q.Start,
		Scanned:   consumed,
		NextStart: nextStart,
		Total:     total,
		HasMore:   consumed > 0 && nextStart < total,
	}, nil
}

// gatherCandidates scans Limit catalog positions from Start, de-duplicating by
// product_id (empathy occasionally repeats an item across page boundaries). It
// returns the unique candidates, how many catalog positions were consumed and
// the total match count.
func (uc *FindBooksInStore) gatherCandidates(ctx context.Context, q FindInStoreQuery) (candidates []domain.Book, consumed, total int, err error) {
	seen := make(map[string]bool)
	offset := q.Start
	for consumed < q.Limit {
		rows := findScanPageSize
		if remaining := q.Limit - consumed; remaining < rows {
			rows = remaining
		}
		res, err := uc.search.Execute(ctx, domain.SearchQuery{
			Query:   q.Query,
			Filters: q.Filters,
			Start:   offset,
			Rows:    rows,
		})
		if err != nil {
			return nil, 0, 0, err
		}
		total = res.Total
		if len(res.Books) == 0 {
			break
		}
		for _, b := range res.Books {
			consumed++
			if !seen[b.ProductID] {
				seen[b.ProductID] = true
				candidates = append(candidates, b)
			}
		}
		offset += len(res.Books)
		if offset >= total {
			break
		}
	}
	return candidates, consumed, total, nil
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
