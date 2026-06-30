package casadellibro_test

import (
	"context"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"app/internal/domain"
	"app/internal/infrastructure/casadellibro"
)

const searchBody = `{
  "catalog": {
    "content": [
      {
        "id": "17422393-ES",
        "internal_id": "17422393",
        "name": "Harry Potter: Cuaderno para colorear",
        "authors": ["Varios autores", "VV.AA."],
        "isbn": "8448045149",
        "ean": "9788448045142",
        "editorial": "Libros Cúpula",
        "productType": "Libro",
        "yearPublication": "2025",
        "availability": "Con stock",
        "url": "https://www.casadellibro.com/libro/9788448045142/17422393",
        "__images": ["https://img/9788448045142.jpg"],
        "description": "desc",
        "price": {"current": 17.05, "previous": 17.95}
      }
    ],
    "pagination": {"start": 0, "rows": 16, "total": 1758}
  }
}`

var _ = Describe("CatalogAdapter", func() {
	It("maps the empathy response and forwards query params", func() {
		var gotQuery string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(searchBody))
		}))
		defer srv.Close()

		adapter := casadellibro.NewCatalogAdapter(casadellibro.NewClient(casadellibro.WithSearchBaseURL(srv.URL)))

		res, err := adapter.Search(context.Background(), domain.SearchQuery{
			Query: "Harry Potter", Start: 0, Rows: 16, Store: "ES", Lang: "es", Currency: "EUR",
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(gotQuery).To(ContainSubstring("query=Harry+Potter"))
		Expect(gotQuery).To(ContainSubstring("store=ES"))
		Expect(gotQuery).To(ContainSubstring("internal=true"))

		Expect(res.Total).To(Equal(1758))
		Expect(res.Books).To(HaveLen(1))
		b := res.Books[0]
		Expect(b.ProductID).To(Equal("17422393"))
		Expect(b.Name).To(Equal("Harry Potter: Cuaderno para colorear"))
		Expect(b.Price.Current).To(Equal(17.05))
		Expect(b.ImageURL).To(Equal("https://img/9788448045142.jpg"))
	})

	It("forwards facet filters as repeated query params", func() {
		var gotQuery string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(searchBody))
		}))
		defer srv.Close()

		adapter := casadellibro.NewCatalogAdapter(casadellibro.NewClient(casadellibro.WithSearchBaseURL(srv.URL)))
		_, err := adapter.Search(context.Background(), domain.SearchQuery{
			Query:   "Harry Potter",
			Filters: []string{"availability:Con stock", "facetLang:Castellano"},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(gotQuery).To(ContainSubstring("filter=availability%3ACon+stock"))
		Expect(gotQuery).To(ContainSubstring("filter=facetLang%3ACastellano"))
	})

	It("returns an error on non-2xx responses", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "nope", http.StatusInternalServerError)
		}))
		defer srv.Close()

		adapter := casadellibro.NewCatalogAdapter(casadellibro.NewClient(casadellibro.WithSearchBaseURL(srv.URL)))
		_, err := adapter.Search(context.Background(), domain.SearchQuery{Query: "x"})
		Expect(err).To(HaveOccurred())
	})

	It("returns an error on malformed JSON", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("not json"))
		}))
		defer srv.Close()

		adapter := casadellibro.NewCatalogAdapter(casadellibro.NewClient(casadellibro.WithSearchBaseURL(srv.URL)))
		_, err := adapter.Search(context.Background(), domain.SearchQuery{Query: "x"})
		Expect(err).To(MatchError(ContainSubstring("decode search response")))
	})
})
