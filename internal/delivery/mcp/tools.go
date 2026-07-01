package mcp

import (
	"context"

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
	storeFields     = jsonFieldNames(domain.Store{})
)

func registerFiltersTool(s *server.MCPServer, uc *usecase.ListSearchFilters) {
	tool := mcp.NewTool("search_books_available_filters",
		mcp.WithDescription("Discover the filters available for a catalog search BEFORE calling search_books. Returns the facets for the query (e.g. language, binding, availability, price ranges, publisher), each with its selectable values and, for every value, a ready-to-use 'filter' string. Pass those exact strings in the search_books 'filters' argument to narrow the search."),
		mcp.WithString("query", mcp.Required(), mcp.Description("The same free-text search you intend to pass to search_books.")),
		mcp.WithString("store", mcp.Description("Store/market code. Default ES.")),
		mcp.WithString("lang", mcp.Description("Language code. Default es.")),
		mcp.WithString("currency", mcp.Description("Currency code. Default EUR.")),
		mcp.WithRawOutputSchema(facetsOutputSchema),
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
		return mcp.NewToolResultStructured(map[string]any{"facets": facets}, renderFacets(facets)), nil
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
			mcp.Required(),
			mcp.Description(fieldsDescription("book", bookFields)),
			mcp.WithStringItems(),
		),
		mcp.WithNumber("start", mcp.Description("Zero-based offset of the first result (pagination). Default 0.")),
		mcp.WithNumber("rows", mcp.Description("Number of results to return (1-100). Default 16.")),
		mcp.WithString("store", mcp.Description("Store/market code. Default ES.")),
		mcp.WithString("lang", mcp.Description("Language code. Default es.")),
		mcp.WithString("currency", mcp.Description("Currency code. Default EUR.")),
		mcp.WithRawOutputSchema(searchOutputSchema),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		fields := req.GetStringSlice("fields", nil)
		if err := requireFields(fields, bookFields); err != nil {
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
		structured := map[string]any{
			"total": result.Total,
			"start": result.Start,
			"rows":  result.Rows,
			"books": books,
		}
		text, err := renderSearch(result, fields)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultStructured(structured, text), nil
	})
}

func registerStockTool(s *server.MCPServer, uc *usecase.GetStoreStock) {
	tool := mcp.NewTool("get_store_stock",
		mcp.WithDescription("List per-bookstore stock and pickup availability for a product across casadellibro physical stores, grouped by province. Use store_id (from list_stores) to scope to a single store, and in_stock_only to drop stores with zero stock."),
		mcp.WithString("product_id", mcp.Required(), mcp.Description("Product id (the product_id returned by search_books, e.g. 16801604).")),
		mcp.WithNumber("country_cache", mcp.Description("casadellibro paiscache value. Default 63 (Spain).")),
		mcp.WithNumber("store_id", mcp.Description("Restrict to a single store by its store_id (from list_stores). Omit for all stores.")),
		mcp.WithBoolean("in_stock_only", mcp.Description("When true, only return bookstores whose stock is greater than zero.")),
		mcp.WithArray("fields",
			mcp.Required(),
			mcp.Description(fieldsDescription("bookstore", bookstoreFields)),
			mcp.WithStringItems(),
		),
		mcp.WithRawOutputSchema(stockOutputSchema),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		productID, err := req.RequireString("product_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		fields := req.GetStringSlice("fields", nil)
		if err := requireFields(fields, bookstoreFields); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		provinces, err := uc.Execute(ctx, usecase.StockQuery{
			ProductID:    productID,
			CountryCache: req.GetInt("country_cache", 0),
			StoreID:      req.GetInt("store_id", 0),
			InStockOnly:  req.GetBool("in_stock_only", false),
		})
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
		text, err := renderStock(provinces, fields)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultStructured(map[string]any{"provinces": out}, text), nil
	})
}

func registerStoresTool(s *server.MCPServer, uc *usecase.ListStores) {
	tool := mcp.NewTool("list_stores",
		mcp.WithDescription("List the casadellibro physical store directory (no product needed). Use the query argument to resolve a store name/location to a store_id (e.g. query \"grancasa\" or \"zaragoza\"), then pass that store_id to get_store_stock or find_books_in_store."),
		mcp.WithString("query", mcp.Description("Optional case-insensitive filter matched against province, city and address (e.g. \"grancasa\", \"madrid\"). Omit to list all stores.")),
		mcp.WithNumber("country_cache", mcp.Description("casadellibro paiscache value. Default 63 (Spain).")),
		mcp.WithArray("fields",
			mcp.Required(),
			mcp.Description(fieldsDescription("store", storeFields)),
			mcp.WithStringItems(),
		),
		mcp.WithRawOutputSchema(storesOutputSchema),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fields := req.GetStringSlice("fields", nil)
		if err := requireFields(fields, storeFields); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		stores, err := uc.Execute(ctx, req.GetString("query", ""), req.GetInt("country_cache", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		items, err := projectItems(stores, fields)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		text, err := renderStores(stores, fields)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultStructured(map[string]any{"stores": items}, text), nil
	})
}

func registerFindInStoreTool(s *server.MCPServer, uc *usecase.FindBooksInStore) {
	tool := mcp.NewTool("find_books_in_store",
		mcp.WithDescription("Find books matching a query/filters that are actually in stock at ONE physical store. Does the search + per-store stock join server-side, so you don't fan out get_store_stock yourself. Get the store_id from list_stores. To browse a whole publisher/collection, set query to that facet value (e.g. \"Unión Editorial\") and add its filter. Scans up to max_scan catalog candidates; if truncated is true, there were more matches than scanned."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Free-text search. To browse a publisher/collection, use its exact name here plus the matching filter.")),
		mcp.WithArray("filters",
			mcp.Description("Facet filter strings from search_books_available_filters (e.g. \"editorial:Unión Editorial\", \"facetLang:Castellano\"). Combined with AND."),
			mcp.WithStringItems(),
		),
		mcp.WithNumber("store_id", mcp.Required(), mcp.Description("The store to check, by store_id (from list_stores).")),
		mcp.WithNumber("max_scan", mcp.Description("Max catalog candidates to check against the store. Default 120, max 400.")),
		mcp.WithNumber("country_cache", mcp.Description("casadellibro paiscache value. Default 63 (Spain).")),
		mcp.WithArray("fields",
			mcp.Required(),
			mcp.Description(fieldsDescription("book", bookFields)+" store_stock and store_availability are always included."),
			mcp.WithStringItems(),
		),
		mcp.WithRawOutputSchema(findInStoreOutputSchema),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		fields := req.GetStringSlice("fields", nil)
		if err := requireFields(fields, bookFields); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := uc.Execute(ctx, usecase.FindInStoreQuery{
			Query:        query,
			Filters:      req.GetStringSlice("filters", nil),
			StoreID:      req.GetInt("store_id", 0),
			MaxScan:      req.GetInt("max_scan", 0),
			CountryCache: req.GetInt("country_cache", 0),
		})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		books, err := projectBooksInStore(res.Books, fields)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		text, err := renderFindInStore(res, fields)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		structured := map[string]any{
			"books":     books,
			"found":     len(res.Books),
			"scanned":   res.Scanned,
			"total":     res.Total,
			"truncated": res.Truncated,
		}
		return mcp.NewToolResultStructured(structured, text), nil
	})
}
