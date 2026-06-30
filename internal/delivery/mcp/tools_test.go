package mcp_test

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mcpclient "github.com/mark3labs/mcp-go/client"
	mcpproto "github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/mock/gomock"

	deliverymcp "app/internal/delivery/mcp"
	"app/internal/domain"
	"app/internal/mocks"
	"app/internal/usecase"
)

var _ = Describe("MCP tools", func() {
	var (
		ctrl    *gomock.Controller
		catalog *mocks.MockCatalogRepository
		stock   *mocks.MockStockRepository
		client  *mcpclient.Client
		ctx     context.Context
	)

	// callText invokes a tool and returns its first text-content block plus the error flag.
	callText := func(name string, args map[string]any) (string, bool) {
		GinkgoHelper()
		req := mcpproto.CallToolRequest{}
		req.Params.Name = name
		req.Params.Arguments = args
		res, err := client.CallTool(ctx, req)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Content).ToNot(BeEmpty())
		txt, ok := mcpproto.AsTextContent(res.Content[0])
		Expect(ok).To(BeTrue())
		return txt.Text, res.IsError
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		catalog = mocks.NewMockCatalogRepository(ctrl)
		stock = mocks.NewMockStockRepository(ctrl)

		srv := deliverymcp.NewServer("test", "0.0.0", deliverymcp.Handlers{
			Search:  usecase.NewSearchBooks(catalog),
			Filters: usecase.NewListSearchFilters(catalog),
			Stock:   usecase.NewGetStoreStock(stock),
		})

		var err error
		client, err = mcpclient.NewInProcessClient(srv)
		Expect(err).ToNot(HaveOccurred())

		ctx = context.Background()
		Expect(client.Start(ctx)).To(Succeed())

		initReq := mcpproto.InitializeRequest{}
		initReq.Params.ProtocolVersion = mcpproto.LATEST_PROTOCOL_VERSION
		initReq.Params.ClientInfo = mcpproto.Implementation{Name: "test", Version: "0.0.0"}
		_, err = client.Initialize(ctx, initReq)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		_ = client.Close()
		ctrl.Finish()
	})

	It("advertises all tools", func() {
		res, err := client.ListTools(ctx, mcpproto.ListToolsRequest{})
		Expect(err).ToNot(HaveOccurred())
		names := []string{}
		for _, t := range res.Tools {
			names = append(names, t.Name)
		}
		Expect(names).To(ConsistOf("search_books", "search_books_available_filters", "get_store_stock"))
	})

	Context("search_books", func() {
		It("forwards args with use-case defaults and returns the result as JSON", func() {
			catalog.EXPECT().
				Search(gomock.Any(), domain.SearchQuery{
					Query: "Harry Potter", Start: 0, Rows: 16, Store: "ES", Lang: "es", Currency: "EUR",
				}).
				Return(domain.SearchResult{
					Total: 1758,
					Books: []domain.Book{{Name: "HP", ProductID: "17422393"}},
				}, nil)

			text, isErr := callText("search_books", map[string]any{"query": "Harry Potter"})
			Expect(isErr).To(BeFalse())

			var got domain.SearchResult
			Expect(json.Unmarshal([]byte(text), &got)).To(Succeed())
			Expect(got.Total).To(Equal(1758))
			Expect(got.Books[0].ProductID).To(Equal("17422393"))
		})

		It("forwards facet filters to the repository", func() {
			catalog.EXPECT().
				Search(gomock.Any(), gomock.AssignableToTypeOf(domain.SearchQuery{})).
				DoAndReturn(func(_ context.Context, q domain.SearchQuery) (domain.SearchResult, error) {
					Expect(q.Filters).To(ConsistOf("availability:Con stock", "facetLang:Castellano"))
					return domain.SearchResult{}, nil
				})

			_, isErr := callText("search_books", map[string]any{
				"query":   "Harry Potter",
				"filters": []any{"availability:Con stock", "facetLang:Castellano"},
			})
			Expect(isErr).To(BeFalse())
		})

		It("returns only the requested fields on each book", func() {
			catalog.EXPECT().
				Search(gomock.Any(), gomock.Any()).
				Return(domain.SearchResult{
					Total: 1,
					Books: []domain.Book{{Name: "HP", ProductID: "123", ISBN: "978", Editorial: "X"}},
				}, nil)

			text, isErr := callText("search_books", map[string]any{
				"query":  "Harry Potter",
				"fields": []any{"name", "product_id"},
			})
			Expect(isErr).To(BeFalse())

			var got struct {
				Total int              `json:"total"`
				Books []map[string]any `json:"books"`
			}
			Expect(json.Unmarshal([]byte(text), &got)).To(Succeed())
			Expect(got.Total).To(Equal(1))
			Expect(got.Books[0]).To(HaveKey("name"))
			Expect(got.Books[0]).To(HaveKey("product_id"))
			Expect(got.Books[0]).To(HaveLen(2))
		})

		It("returns a tool error for an unknown field without hitting the repository", func() {
			_, isErr := callText("search_books", map[string]any{
				"query":  "Harry Potter",
				"fields": []any{"nope"},
			})
			Expect(isErr).To(BeTrue())
		})

		It("returns a tool error for an empty query without hitting the repository", func() {
			_, isErr := callText("search_books", map[string]any{"query": "  "})
			Expect(isErr).To(BeTrue())
		})

		It("returns a tool error when query is missing", func() {
			_, isErr := callText("search_books", map[string]any{})
			Expect(isErr).To(BeTrue())
		})
	})

	Context("search_books_available_filters", func() {
		It("forwards args with use-case defaults and returns the facets as JSON", func() {
			catalog.EXPECT().
				Facets(gomock.Any(), domain.FacetQuery{
					Query: "Harry Potter", Store: "ES", Lang: "es", Currency: "EUR",
				}).
				Return([]domain.Facet{{
					Label: "Idioma",
					Type:  "value",
					Values: []domain.FacetValue{
						{Value: "Castellano", Count: 738, Filter: "facetLang:Castellano"},
					},
				}}, nil)

			text, isErr := callText("search_books_available_filters", map[string]any{"query": "Harry Potter"})
			Expect(isErr).To(BeFalse())

			var got []domain.Facet
			Expect(json.Unmarshal([]byte(text), &got)).To(Succeed())
			Expect(got).To(HaveLen(1))
			Expect(got[0].Values[0].Filter).To(Equal("facetLang:Castellano"))
		})

		It("returns a tool error when query is missing", func() {
			_, isErr := callText("search_books_available_filters", map[string]any{})
			Expect(isErr).To(BeTrue())
		})
	})

	Context("get_store_stock", func() {
		It("forwards the product id and default country cache", func() {
			stock.EXPECT().
				StockByStore(gomock.Any(), "16801604", 63).
				Return([]domain.Province{{Name: "Alicante"}}, nil)

			text, isErr := callText("get_store_stock", map[string]any{"product_id": "16801604"})
			Expect(isErr).To(BeFalse())

			var got []domain.Province
			Expect(json.Unmarshal([]byte(text), &got)).To(Succeed())
			Expect(got).To(HaveLen(1))
			Expect(got[0].Name).To(Equal("Alicante"))
		})

		It("returns only the requested fields on each bookstore", func() {
			stock.EXPECT().
				StockByStore(gomock.Any(), "16801604", 63).
				Return([]domain.Province{{
					Name:       "Alicante",
					Bookstores: []domain.Bookstore{{City: "Alicante", Stock: 3, Phone: "900", Email: "x@y"}},
				}}, nil)

			text, isErr := callText("get_store_stock", map[string]any{
				"product_id": "16801604",
				"fields":     []any{"city", "stock"},
			})
			Expect(isErr).To(BeFalse())

			var got []struct {
				Name       string           `json:"name"`
				Bookstores []map[string]any `json:"bookstores"`
			}
			Expect(json.Unmarshal([]byte(text), &got)).To(Succeed())
			Expect(got[0].Name).To(Equal("Alicante"))
			Expect(got[0].Bookstores[0]).To(HaveKey("city"))
			Expect(got[0].Bookstores[0]).To(HaveKey("stock"))
			Expect(got[0].Bookstores[0]).To(HaveLen(2))
		})

		It("returns a tool error for an unknown field", func() {
			_, isErr := callText("get_store_stock", map[string]any{
				"product_id": "16801604",
				"fields":     []any{"nope"},
			})
			Expect(isErr).To(BeTrue())
		})

		It("returns a tool error when product_id is missing", func() {
			_, isErr := callText("get_store_stock", map[string]any{})
			Expect(isErr).To(BeTrue())
		})
	})
})
