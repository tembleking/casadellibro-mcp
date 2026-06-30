package mcp_test

import (
	"context"
	"strings"

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

			text, isErr := callText("search_books", map[string]any{
				"query":  "Harry Potter",
				"fields": []any{"name", "product_id"},
			})
			Expect(isErr).To(BeFalse())

			lines := strings.Split(text, "\n")
			Expect(lines[0]).To(Equal("total=1758 start=0 rows=0"))
			Expect(lines[1]).To(Equal("name\tproduct_id"))
			Expect(lines[2]).To(Equal("HP\t17422393"))
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
				"fields":  []any{"name"},
			})
			Expect(isErr).To(BeFalse())
		})

		It("returns a tool error when fields is missing", func() {
			_, isErr := callText("search_books", map[string]any{"query": "Harry Potter"})
			Expect(isErr).To(BeTrue())
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

			lines := strings.Split(text, "\n")
			Expect(lines[1]).To(Equal("name\tproduct_id"))
			Expect(lines[2]).To(Equal("HP\t123"))
			// unrequested fields (isbn, editorial) are absent.
			Expect(text).ToNot(ContainSubstring("978"))
			Expect(text).ToNot(ContainSubstring("X"))
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
		It("forwards args with use-case defaults and groups facets as text", func() {
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

			lines := strings.Split(text, "\n")
			Expect(lines[0]).To(Equal("# Idioma [value]"))
			Expect(lines[1]).To(Equal("facetLang:Castellano\t(738)"))
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
				Return([]domain.Province{{
					Name:       "Alicante",
					Bookstores: []domain.Bookstore{{City: "Elche"}},
				}}, nil)

			text, isErr := callText("get_store_stock", map[string]any{
				"product_id": "16801604",
				"fields":     []any{"city"},
			})
			Expect(isErr).To(BeFalse())

			lines := strings.Split(text, "\n")
			Expect(lines[0]).To(Equal("province\tcity"))
			Expect(lines[1]).To(Equal("Alicante\tElche"))
		})

		It("returns a tool error when fields is missing", func() {
			_, isErr := callText("get_store_stock", map[string]any{"product_id": "16801604"})
			Expect(isErr).To(BeTrue())
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

			lines := strings.Split(text, "\n")
			Expect(lines[0]).To(Equal("province\tcity\tstock"))
			Expect(lines[1]).To(Equal("Alicante\tAlicante\t3"))
			// unrequested fields (phone, email) are absent.
			Expect(text).ToNot(ContainSubstring("900"))
			Expect(text).ToNot(ContainSubstring("x@y"))
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
