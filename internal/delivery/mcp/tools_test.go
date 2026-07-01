package mcp_test

import (
	"context"
	"encoding/json"
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

	call := func(name string, args map[string]any) *mcpproto.CallToolResult {
		GinkgoHelper()
		req := mcpproto.CallToolRequest{}
		req.Params.Name = name
		req.Params.Arguments = args
		res, err := client.CallTool(ctx, req)
		Expect(err).ToNot(HaveOccurred())
		return res
	}

	// callText invokes a tool and returns its first text-content block plus the error flag.
	callText := func(name string, args map[string]any) (string, bool) {
		GinkgoHelper()
		res := call(name, args)
		Expect(res.Content).ToNot(BeEmpty())
		txt, ok := mcpproto.AsTextContent(res.Content[0])
		Expect(ok).To(BeTrue())
		return txt.Text, res.IsError
	}

	// structured invokes a tool and returns its structuredContent as a generic map.
	structured := func(name string, args map[string]any) map[string]any {
		GinkgoHelper()
		res := call(name, args)
		b, err := json.Marshal(res.StructuredContent)
		Expect(err).ToNot(HaveOccurred())
		var m map[string]any
		Expect(json.Unmarshal(b, &m)).To(Succeed())
		return m
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		catalog = mocks.NewMockCatalogRepository(ctrl)
		stock = mocks.NewMockStockRepository(ctrl)

		search := usecase.NewSearchBooks(catalog)
		stockUC := usecase.NewGetStoreStock(stock)
		srv := deliverymcp.NewServer("test", "0.0.0", deliverymcp.Handlers{
			Search:      search,
			Filters:     usecase.NewListSearchFilters(catalog),
			Stock:       stockUC,
			Stores:      usecase.NewListStores(stock),
			FindInStore: usecase.NewFindBooksInStore(search, stockUC),
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
		Expect(names).To(ConsistOf(
			"search_books", "search_books_available_filters", "get_store_stock",
			"list_stores", "find_books_in_store",
		))
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

		It("also returns structuredContent projected to the requested fields", func() {
			catalog.EXPECT().
				Search(gomock.Any(), gomock.Any()).
				Return(domain.SearchResult{
					Total: 42,
					Books: []domain.Book{{Name: "HP", ProductID: "123", ISBN: "978"}},
				}, nil)

			sc := structured("search_books", map[string]any{
				"query":  "Harry Potter",
				"fields": []any{"name", "product_id"},
			})
			Expect(sc["total"]).To(BeNumerically("==", 42))
			books := sc["books"].([]any)
			Expect(books).To(HaveLen(1))
			book := books[0].(map[string]any)
			Expect(book).To(HaveKeyWithValue("product_id", "123"))
			Expect(book).ToNot(HaveKey("isbn"))
		})

		It("renders scalar cells (price, joined authors) as plain text, never JSON", func() {
			catalog.EXPECT().
				Search(gomock.Any(), gomock.Any()).
				Return(domain.SearchResult{
					Books: []domain.Book{{Price: 12.3, Authors: []string{"J.K. Rowling", "VV.AA."}}},
				}, nil)

			text, isErr := callText("search_books", map[string]any{
				"query":  "Harry Potter",
				"fields": []any{"price", "authors"},
			})
			Expect(isErr).To(BeFalse())
			Expect(text).ToNot(ContainSubstring("{"))
			lines := strings.Split(text, "\n")
			Expect(lines[2]).To(Equal("12.3\tJ.K. Rowling; VV.AA."))
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

		It("wraps the facets under a facets key in structuredContent", func() {
			catalog.EXPECT().
				Facets(gomock.Any(), gomock.Any()).
				Return([]domain.Facet{{Label: "Idioma", Type: "value", Values: []domain.FacetValue{
					{Value: "Castellano", Count: 738, Filter: "facetLang:Castellano"},
				}}}, nil)

			sc := structured("search_books_available_filters", map[string]any{"query": "Harry Potter"})
			facets := sc["facets"].([]any)
			Expect(facets).To(HaveLen(1))
			Expect(facets[0].(map[string]any)).To(HaveKeyWithValue("label", "Idioma"))
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

		It("wraps provinces under a provinces key in structuredContent", func() {
			stock.EXPECT().
				StockByStore(gomock.Any(), "16801604", 63).
				Return([]domain.Province{{
					Name:       "Alicante",
					Bookstores: []domain.Bookstore{{City: "Elche", Stock: 2, Phone: "900"}},
				}}, nil)

			sc := structured("get_store_stock", map[string]any{
				"product_id": "16801604",
				"fields":     []any{"city", "stock"},
			})
			provinces := sc["provinces"].([]any)
			Expect(provinces).To(HaveLen(1))
			p := provinces[0].(map[string]any)
			Expect(p).To(HaveKeyWithValue("name", "Alicante"))
			store := p["bookstores"].([]any)[0].(map[string]any)
			Expect(store).To(HaveKeyWithValue("city", "Elche"))
			Expect(store).ToNot(HaveKey("phone"))
		})

		It("returns a tool error for an unknown field", func() {
			_, isErr := callText("get_store_stock", map[string]any{
				"product_id": "16801604",
				"fields":     []any{"nope"},
			})
			Expect(isErr).To(BeTrue())
		})

		It("scopes to a single store when store_id is given", func() {
			stock.EXPECT().
				StockByStore(gomock.Any(), "16801604", 63).
				Return([]domain.Province{{
					Name: "Zaragoza",
					Bookstores: []domain.Bookstore{
						{StoreID: 20, City: "Zaragoza", Stock: 0},
						{StoreID: 38, City: "Zaragoza", Stock: 2},
					},
				}}, nil)

			text, isErr := callText("get_store_stock", map[string]any{
				"product_id": "16801604",
				"store_id":   38,
				"fields":     []any{"store_id", "stock"},
			})
			Expect(isErr).To(BeFalse())
			lines := strings.Split(text, "\n")
			Expect(lines).To(HaveLen(2)) // header + single store row
			Expect(lines[1]).To(Equal("Zaragoza\t38\t2"))
		})

		It("returns a tool error when product_id is missing", func() {
			_, isErr := callText("get_store_stock", map[string]any{})
			Expect(isErr).To(BeTrue())
		})
	})

	Context("list_stores", func() {
		It("resolves a space-insensitive query to the matching store", func() {
			stock.EXPECT().
				Stores(gomock.Any(), 63).
				Return([]domain.Store{
					{StoreID: 20, Province: "Zaragoza", City: "Zaragoza", Address: "San Miguel, 4"},
					{StoreID: 38, Province: "Zaragoza", City: "Zaragoza", Address: "C. C. Gran Casa, av. María Zambrano, 35"},
				}, nil)

			sc := structured("list_stores", map[string]any{
				"query":  "grancasa",
				"fields": []any{"store_id", "address"},
			})
			stores := sc["stores"].([]any)
			Expect(stores).To(HaveLen(1))
			Expect(stores[0].(map[string]any)).To(HaveKeyWithValue("store_id", BeNumerically("==", 38)))
		})

		It("returns a tool error when fields is missing", func() {
			_, isErr := callText("list_stores", map[string]any{"query": "madrid"})
			Expect(isErr).To(BeTrue())
		})
	})

	Context("find_books_in_store", func() {
		It("returns only books in stock at the store, with the store annotations", func() {
			catalog.EXPECT().
				Search(gomock.Any(), gomock.Any()).
				Return(domain.SearchResult{
					Total: 2,
					Books: []domain.Book{
						{ProductID: "1", Name: "In stock here"},
						{ProductID: "2", Name: "Not here"},
					},
				}, nil)
			// product 1 has stock at store 38, product 2 does not.
			stock.EXPECT().StockByStore(gomock.Any(), "1", 63).
				Return([]domain.Province{{Name: "Zaragoza", Bookstores: []domain.Bookstore{
					{StoreID: 38, Stock: 2, Availability: "recógelo hoy"},
				}}}, nil)
			stock.EXPECT().StockByStore(gomock.Any(), "2", 63).
				Return([]domain.Province{{Name: "Zaragoza", Bookstores: []domain.Bookstore{
					{StoreID: 38, Stock: 0},
				}}}, nil)

			sc := structured("find_books_in_store", map[string]any{
				"query":    "algo",
				"store_id": 38,
				"fields":   []any{"product_id", "name"},
			})
			Expect(sc["found"]).To(BeNumerically("==", 1))
			books := sc["books"].([]any)
			Expect(books).To(HaveLen(1))
			b := books[0].(map[string]any)
			Expect(b).To(HaveKeyWithValue("product_id", "1"))
			Expect(b).To(HaveKeyWithValue("store_stock", BeNumerically("==", 2)))
			Expect(b).To(HaveKeyWithValue("store_availability", "recógelo hoy"))
		})

		It("returns a tool error when store_id is missing", func() {
			_, isErr := callText("find_books_in_store", map[string]any{
				"query":  "algo",
				"fields": []any{"name"},
			})
			Expect(isErr).To(BeTrue())
		})
	})
})
