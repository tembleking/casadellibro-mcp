package casadellibro_test

import (
	"context"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"app/internal/infrastructure/casadellibro"
)

const stockBody = `[
  {
    "name": "Alicante",
    "bookstores": [
      {
        "city": "Elche",
        "idTienda": 90,
        "address": "C. C. L'Aljub",
        "phone": "911793463",
        "email": "elche@casadellibro.com",
        "schedule": "L-S 10-22",
        "stock": 2,
        "availabilityText": "recógelo hoy",
        "postalCode": "03205",
        "latitud": 38.26197,
        "longitud": -0.72091,
        "map": "https://maps/elche"
      }
    ]
  }
]`

var _ = Describe("StockAdapter", func() {
	It("maps the stock response and forwards query params", func() {
		var gotQuery string
		var gotReferer string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.RawQuery
			gotReferer = r.Header.Get("Referer")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(stockBody))
		}))
		defer srv.Close()

		adapter := casadellibro.NewStockAdapter(casadellibro.NewClient(casadellibro.WithStockBaseURL(srv.URL)))

		provinces, err := adapter.StockByStore(context.Background(), "16801604", 63)
		Expect(err).ToNot(HaveOccurred())

		Expect(gotQuery).To(ContainSubstring("idproducto=16801604"))
		Expect(gotQuery).To(ContainSubstring("paiscache=63"))
		Expect(gotReferer).To(Equal("https://www.casadellibro.com"))

		Expect(provinces).To(HaveLen(1))
		Expect(provinces[0].Name).To(Equal("Alicante"))
		store := provinces[0].Bookstores[0]
		Expect(store.StoreID).To(Equal(90))
		Expect(store.City).To(Equal("Elche"))
		Expect(store.Stock).To(Equal(2))
		Expect(store.Availability).To(Equal("recógelo hoy"))
		Expect(store.Latitude).To(Equal(38.26197))
	})

	It("returns an error on non-2xx responses", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "nope", http.StatusBadGateway)
		}))
		defer srv.Close()

		adapter := casadellibro.NewStockAdapter(casadellibro.NewClient(casadellibro.WithStockBaseURL(srv.URL)))
		_, err := adapter.StockByStore(context.Background(), "1", 63)
		Expect(err).To(HaveOccurred())
	})

	It("returns an error on malformed JSON", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("{nope"))
		}))
		defer srv.Close()

		adapter := casadellibro.NewStockAdapter(casadellibro.NewClient(casadellibro.WithStockBaseURL(srv.URL)))
		_, err := adapter.StockByStore(context.Background(), "1", 63)
		Expect(err).To(MatchError(ContainSubstring("decode stock response")))
	})
})
