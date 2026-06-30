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

const facetsBody = `{
  "catalog": {
    "facets": [
      {
        "facet": "facetEditorial",
        "filter": "editorial",
        "label": "Editorial",
        "type": "value",
        "values": [
          {"id": " ", "value": " ", "count": 476, "filter": "editorial: "},
          {"id": "BLOOMSBURY", "value": "BLOOMSBURY", "count": 192, "filter": "editorial:BLOOMSBURY"}
        ]
      },
      {
        "facet": "lang",
        "filter": "facetLang",
        "label": "",
        "type": "value",
        "values": [
          {"id": "Castellano", "value": "Castellano", "count": 738, "filter": "facetLang:Castellano"}
        ]
      }
    ]
  }
}`

var _ = Describe("CatalogAdapter.Facets", func() {
	It("maps facets, drops blank values and falls back to the facet name as label", func() {
		var gotQuery string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(facetsBody))
		}))
		defer srv.Close()

		adapter := casadellibro.NewCatalogAdapter(casadellibro.NewClient(casadellibro.WithFacetsBaseURL(srv.URL)))
		facets, err := adapter.Facets(context.Background(), domain.FacetQuery{
			Query: "Harry Potter", Store: "ES", Lang: "es", Currency: "EUR",
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(gotQuery).To(ContainSubstring("query=Harry+Potter"))

		Expect(facets).To(HaveLen(2))
		Expect(facets[0].Label).To(Equal("Editorial"))
		// the blank " " value is dropped, only BLOOMSBURY remains.
		Expect(facets[0].Values).To(HaveLen(1))
		Expect(facets[0].Values[0].Filter).To(Equal("editorial:BLOOMSBURY"))
		// missing label falls back to the facet name.
		Expect(facets[1].Label).To(Equal("lang"))
	})

	It("returns an error on malformed JSON", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("not json"))
		}))
		defer srv.Close()

		adapter := casadellibro.NewCatalogAdapter(casadellibro.NewClient(casadellibro.WithFacetsBaseURL(srv.URL)))
		_, err := adapter.Facets(context.Background(), domain.FacetQuery{Query: "x"})
		Expect(err).To(MatchError(ContainSubstring("decode facets response")))
	})
})
