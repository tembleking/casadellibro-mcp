package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"app/internal/domain"
	"app/internal/usecase"
)

// Valid field names a caller may request, derived from the domain entities so
// they cannot drift from what is actually returned.
var (
	bookFields      = jsonFieldNames(domain.Book{})
	bookstoreFields = jsonFieldNames(domain.Bookstore{})
)

func registerFiltersTool(s *server.MCPServer, uc *usecase.ListSearchFilters) {
	tool := mcp.NewTool("search_books_available_filters",
		mcp.WithDescription("Discover the filters available for a catalog search BEFORE calling search_books. Returns the facets for the query (e.g. language, binding, availability, price ranges, publisher), each with its selectable values and, for every value, a ready-to-use 'filter' string. Pass those exact strings in the search_books 'filters' argument to narrow the search."),
		mcp.WithString("query", mcp.Required(), mcp.Description("The same free-text search you intend to pass to search_books.")),
		mcp.WithString("store", mcp.Description("Store/market code. Default ES.")),
		mcp.WithString("lang", mcp.Description("Language code. Default es.")),
		mcp.WithString("currency", mcp.Description("Currency code. Default EUR.")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		facets, err := uc.Execute(ctx, domain.FacetQuery{
			Query:    query,
			Store:    req.GetString("store", ""),
			Lang:     req.GetString("lang", ""),
			Currency: req.GetString("currency", ""),
		})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(facets)
	})
}

func registerSearchTool(s *server.MCPServer, uc *usecase.SearchBooks) {
	tool := mcp.NewTool("search_books",
		mcp.WithDescription("Search the casadellibro catalog by free text. Returns matching books with price, availability, ISBN and a product_id usable with get_store_stock. NOTE: every field describes the online catalog listing, not any physical store. In particular `availability` (e.g. \"Con stock\") is catalog-wide — it means at least one store/warehouse has it, NOT that any given store does — and `price` is the online price, which a physical store may not match. Use get_store_stock with the product_id for per-store stock and pickup. To narrow results, first call search_books_available_filters and pass the returned filter strings here."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Free-text search, e.g. an author, title or ISBN.")),
		mcp.WithArray("filters",
			mcp.Description("Facet filter strings to narrow the search, exactly as returned by search_books_available_filters (e.g. \"availability:Con stock\", \"facetLang:Castellano\", \"priceOffer:8.0-14.0\"). Multiple filters combine with AND."),
			mcp.WithStringItems(),
		),
		mcp.WithArray("fields",
			mcp.Description(fieldsDescription("book", bookFields)),
			mcp.WithStringItems(),
		),
		mcp.WithNumber("start", mcp.Description("Zero-based offset of the first result (pagination). Default 0.")),
		mcp.WithNumber("rows", mcp.Description("Number of results to return (1-100). Default 16.")),
		mcp.WithString("store", mcp.Description("Store/market code. Default ES.")),
		mcp.WithString("lang", mcp.Description("Language code. Default es.")),
		mcp.WithString("currency", mcp.Description("Currency code. Default EUR.")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		fields := req.GetStringSlice("fields", nil)
		if err := validateFields(fields, bookFields); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		q := domain.SearchQuery{
			Query:    query,
			Filters:  req.GetStringSlice("filters", nil),
			Start:    req.GetInt("start", 0),
			Rows:     req.GetInt("rows", 0),
			Store:    req.GetString("store", ""),
			Lang:     req.GetString("lang", ""),
			Currency: req.GetString("currency", ""),
		}
		result, err := uc.Execute(ctx, q)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		books, err := projectItems(result.Books, fields)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(map[string]any{
			"books": books,
			"total": result.Total,
			"start": result.Start,
			"rows":  result.Rows,
		})
	})
}

func registerStockTool(s *server.MCPServer, uc *usecase.GetStoreStock) {
	tool := mcp.NewTool("get_store_stock",
		mcp.WithDescription("List per-bookstore stock and pickup availability for a product across casadellibro physical stores, grouped by province."),
		mcp.WithString("product_id", mcp.Required(), mcp.Description("Product id (the product_id returned by search_books, e.g. 16801604).")),
		mcp.WithNumber("country_cache", mcp.Description("casadellibro paiscache value. Default 63 (Spain).")),
		mcp.WithArray("fields",
			mcp.Description(fieldsDescription("bookstore", bookstoreFields)),
			mcp.WithStringItems(),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		productID, err := req.RequireString("product_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		fields := req.GetStringSlice("fields", nil)
		if err := validateFields(fields, bookstoreFields); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		provinces, err := uc.Execute(ctx, productID, req.GetInt("country_cache", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		out := make([]map[string]any, 0, len(provinces))
		for _, p := range provinces {
			stores, err := projectItems(p.Bookstores, fields)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			out = append(out, map[string]any{"name": p.Name, "bookstores": stores})
		}
		return jsonResult(out)
	})
}

// jsonResult marshals a value to indented JSON as a tool text result.
func jsonResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}
