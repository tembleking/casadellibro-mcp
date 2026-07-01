package casadellibro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"app/internal/domain"
)

// StockAdapter implements domain.StockRepository against the casadellibro web API.
type StockAdapter struct {
	client *Client
}

// NewStockAdapter builds a stock adapter on top of the shared client.
func NewStockAdapter(client *Client) *StockAdapter {
	return &StockAdapter{client: client}
}

type provinceDTO struct {
	Name       string         `json:"name"`
	Bookstores []bookstoreDTO `json:"bookstores"`
}

type bookstoreDTO struct {
	City             string  `json:"city"`
	StoreID          int     `json:"idTienda"`
	Address          string  `json:"address"`
	Phone            string  `json:"phone"`
	Email            string  `json:"email"`
	Schedule         string  `json:"schedule"`
	Stock            int     `json:"stock"`
	AvailabilityText string  `json:"availabilityText"`
	PostalCode       string  `json:"postalCode"`
	Latitude         float64 `json:"latitud"`
	Longitude        float64 `json:"longitud"`
	MapURL           string  `json:"map"`
}

// StockByStore fetches per-store stock and maps it to the domain model.
func (a *StockAdapter) StockByStore(ctx context.Context, productID string, countryCache int) ([]domain.Province, error) {
	params := url.Values{}
	params.Set("paiscache", strconv.Itoa(countryCache))
	params.Set("idproducto", productID)

	body, err := a.client.getJSON(ctx, a.client.stockBaseURL, params)
	if err != nil {
		return nil, err
	}

	var dtos []provinceDTO
	if err := json.Unmarshal(body, &dtos); err != nil {
		return nil, fmt.Errorf("decode stock response: %w", err)
	}

	provinces := make([]domain.Province, 0, len(dtos))
	for _, p := range dtos {
		stores := make([]domain.Bookstore, 0, len(p.Bookstores))
		for _, b := range p.Bookstores {
			stores = append(stores, domain.Bookstore{
				StoreID:      b.StoreID,
				City:         b.City,
				Address:      b.Address,
				Phone:        b.Phone,
				Email:        b.Email,
				Schedule:     b.Schedule,
				Stock:        b.Stock,
				Availability: b.AvailabilityText,
				PostalCode:   b.PostalCode,
				Latitude:     b.Latitude,
				Longitude:    b.Longitude,
				MapURL:       b.MapURL,
			})
		}
		provinces = append(provinces, domain.Province{Name: p.Name, Bookstores: stores})
	}

	return provinces, nil
}

// Stores lists the full casadellibro store directory (no product context), via
// the todasTiendas endpoint. The response is grouped by province just like
// stockTiendas, so it reuses the same DTOs and is flattened into []domain.Store.
func (a *StockAdapter) Stores(ctx context.Context, countryCache int) ([]domain.Store, error) {
	params := url.Values{}
	params.Set("paiscache", strconv.Itoa(countryCache))

	body, err := a.client.getJSON(ctx, a.client.storesBaseURL, params)
	if err != nil {
		return nil, err
	}

	var dtos []provinceDTO
	if err := json.Unmarshal(body, &dtos); err != nil {
		return nil, fmt.Errorf("decode stores response: %w", err)
	}

	var stores []domain.Store
	for _, p := range dtos {
		for _, b := range p.Bookstores {
			stores = append(stores, domain.Store{
				StoreID:    b.StoreID,
				Province:   p.Name,
				City:       b.City,
				Address:    b.Address,
				PostalCode: b.PostalCode,
				Phone:      b.Phone,
				Email:      b.Email,
				Schedule:   b.Schedule,
				Latitude:   b.Latitude,
				Longitude:  b.Longitude,
				MapURL:     b.MapURL,
			})
		}
	}

	return stores, nil
}
