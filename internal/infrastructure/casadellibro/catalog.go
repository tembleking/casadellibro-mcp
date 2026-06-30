package casadellibro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"app/internal/domain"
)

// CatalogAdapter implements domain.CatalogRepository against the empathy.co API.
type CatalogAdapter struct {
	client *Client
}

// NewCatalogAdapter builds a catalog adapter on top of the shared client.
func NewCatalogAdapter(client *Client) *CatalogAdapter {
	return &CatalogAdapter{client: client}
}

type searchResponse struct {
	Catalog struct {
		Content    []contentItem `json:"content"`
		Pagination struct {
			Start int `json:"start"`
			Rows  int `json:"rows"`
			Total int `json:"total"`
		} `json:"pagination"`
	} `json:"catalog"`
}

type contentItem struct {
	ID           string   `json:"id"`
	InternalID   string   `json:"internal_id"`
	Name         string   `json:"name"`
	Authors      []string `json:"authors"`
	ISBN         string   `json:"isbn"`
	EAN          string   `json:"ean"`
	Editorial    string   `json:"editorial"`
	ProductType  string   `json:"productType"`
	Year         string   `json:"yearPublication"`
	Availability string   `json:"availability"`
	URL          string   `json:"url"`
	Images       []string `json:"__images"`
	Description  string   `json:"description"`
	Price        struct {
		Current  float64 `json:"current"`
		Previous float64 `json:"previous"`
	} `json:"price"`
}

// Search queries the catalog and maps the response to the domain model.
func (a *CatalogAdapter) Search(ctx context.Context, q domain.SearchQuery) (domain.SearchResult, error) {
	params := url.Values{}
	params.Set("internal", "true")
	params.Set("query", q.Query)
	params.Set("start", strconv.Itoa(q.Start))
	params.Set("rows", strconv.Itoa(q.Rows))
	params.Set("lang", q.Lang)
	params.Set("currency", q.Currency)
	params.Set("store", q.Store)

	body, err := a.client.getJSON(ctx, a.client.searchBaseURL, params)
	if err != nil {
		return domain.SearchResult{}, err
	}

	var resp searchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return domain.SearchResult{}, fmt.Errorf("decode search response: %w", err)
	}

	books := make([]domain.Book, 0, len(resp.Catalog.Content))
	for _, it := range resp.Catalog.Content {
		var img string
		if len(it.Images) > 0 {
			img = it.Images[0]
		}
		books = append(books, domain.Book{
			ID:           it.ID,
			ProductID:    it.InternalID,
			Name:         it.Name,
			Authors:      it.Authors,
			ISBN:         it.ISBN,
			EAN:          it.EAN,
			Editorial:    it.Editorial,
			ProductType:  it.ProductType,
			Year:         it.Year,
			Price:        domain.Price{Current: it.Price.Current, Previous: it.Price.Previous},
			Availability: it.Availability,
			URL:          it.URL,
			ImageURL:     img,
			Description:  it.Description,
		})
	}

	return domain.SearchResult{
		Books: books,
		Total: resp.Catalog.Pagination.Total,
		Start: resp.Catalog.Pagination.Start,
		Rows:  resp.Catalog.Pagination.Rows,
	}, nil
}
