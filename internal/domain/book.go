package domain

// Book is a catalog entry returned by a search. Prices are flat scalar fields
// (current and previous) so a projected result never embeds a nested object.
type Book struct {
	ID            string   `json:"id"`
	ProductID     string   `json:"product_id"`
	Name          string   `json:"name"`
	Authors       []string `json:"authors"`
	ISBN          string   `json:"isbn"`
	EAN           string   `json:"ean"`
	Editorial     string   `json:"editorial"`
	ProductType   string   `json:"product_type"`
	Year          string   `json:"year"`
	Price         float64  `json:"price"`
	PricePrevious float64  `json:"price_previous"`
	Availability  string   `json:"availability"`
	URL           string   `json:"url"`
	ImageURL      string   `json:"image_url"`
	Description   string   `json:"description"`
}

// SearchResult is a page of books plus pagination metadata.
type SearchResult struct {
	Books []Book `json:"books"`
	Total int    `json:"total"`
	Start int    `json:"start"`
	Rows  int    `json:"rows"`
}
