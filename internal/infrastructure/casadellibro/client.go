// Package casadellibro contains the HTTP adapters that implement the domain
// repositories against the public casadellibro / empathy.co APIs.
package casadellibro

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Default API endpoints.
const (
	defaultSearchBaseURL = "https://api.empathy.co/search/v1/query/cdl/search"
	defaultStockBaseURL  = "https://www.casadellibro.com/cdlweb/api/libreria/stockTiendas"
	referer              = "https://www.casadellibro.com"
)

// Client is the shared HTTP transport for the casadellibro adapters.
type Client struct {
	http          *http.Client
	searchBaseURL string
	stockBaseURL  string
}

// Option customizes a Client.
type Option func(*Client)

// WithHTTPClient overrides the underlying *http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// WithSearchBaseURL overrides the catalog search endpoint (useful in tests).
func WithSearchBaseURL(u string) Option {
	return func(c *Client) { c.searchBaseURL = u }
}

// WithStockBaseURL overrides the stock endpoint (useful in tests).
func WithStockBaseURL(u string) Option {
	return func(c *Client) { c.stockBaseURL = u }
}

// NewClient builds a Client with sane defaults.
func NewClient(opts ...Option) *Client {
	c := &Client{
		http:          &http.Client{Timeout: 15 * time.Second},
		searchBaseURL: defaultSearchBaseURL,
		stockBaseURL:  defaultStockBaseURL,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// getJSON performs a GET request and returns the raw body, failing on non-2xx.
func (c *Client) getJSON(ctx context.Context, endpoint string, query url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", referer)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}
