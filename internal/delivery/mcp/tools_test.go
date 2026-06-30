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
			Search: usecase.NewSearchBooks(catalog),
			Stock:  usecase.NewGetStoreStock(stock),
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

	It("advertises both tools", func() {
		res, err := client.ListTools(ctx, mcpproto.ListToolsRequest{})
		Expect(err).ToNot(HaveOccurred())
		names := []string{}
		for _, t := range res.Tools {
			names = append(names, t.Name)
		}
		Expect(names).To(ConsistOf("search_books", "get_store_stock"))
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

		It("returns a tool error for an empty query without hitting the repository", func() {
			_, isErr := callText("search_books", map[string]any{"query": "  "})
			Expect(isErr).To(BeTrue())
		})

		It("returns a tool error when query is missing", func() {
			_, isErr := callText("search_books", map[string]any{})
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

		It("returns a tool error when product_id is missing", func() {
			_, isErr := callText("get_store_stock", map[string]any{})
			Expect(isErr).To(BeTrue())
		})
	})
})
