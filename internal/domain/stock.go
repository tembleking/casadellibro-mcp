package domain

// Bookstore is a physical store with stock for a given product.
type Bookstore struct {
	StoreID      int     `json:"store_id"`
	City         string  `json:"city"`
	Address      string  `json:"address"`
	Phone        string  `json:"phone"`
	Email        string  `json:"email"`
	Schedule     string  `json:"schedule"`
	Stock        int     `json:"stock"`
	Availability string  `json:"availability"`
	PostalCode   string  `json:"postal_code"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	MapURL       string  `json:"map_url"`
}

// Province groups the bookstores of a Spanish province.
type Province struct {
	Name       string      `json:"name"`
	Bookstores []Bookstore `json:"bookstores"`
}

// Store is a physical casadellibro bookstore in the store directory (no product
// context, hence no stock/availability).
type Store struct {
	StoreID    int     `json:"store_id"`
	Province   string  `json:"province"`
	City       string  `json:"city"`
	Address    string  `json:"address"`
	PostalCode string  `json:"postal_code"`
	Phone      string  `json:"phone"`
	Email      string  `json:"email"`
	Schedule   string  `json:"schedule"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	MapURL     string  `json:"map_url"`
}
