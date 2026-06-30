package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"app/internal/domain"
	"app/internal/usecase"
)

func registerSearchTool(s *server.MCPServer, uc *usecase.SearchBooks) {
	tool := mcp.NewTool("search_books",
		mcp.WithDescription("Search the casadellibro catalog by free text. Returns matching books with price, availability, ISBN and a product_id usable with get_store_stock."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Free-text search, e.g. an author, title or ISBN.")),
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
		q := domain.SearchQuery{
			Query:    query,
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
		return jsonResult(result)
	})
}

func registerStockTool(s *server.MCPServer, uc *usecase.GetStoreStock) {
	tool := mcp.NewTool("get_store_stock",
		mcp.WithDescription("List per-bookstore stock and pickup availability for a product across casadellibro physical stores, grouped by province."),
		mcp.WithString("product_id", mcp.Required(), mcp.Description("Product id (the product_id returned by search_books, e.g. 16801604).")),
		mcp.WithNumber("country_cache", mcp.Description("casadellibro paiscache value. Default 63 (Spain).")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		productID, err := req.RequireString("product_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		provinces, err := uc.Execute(ctx, productID, req.GetInt("country_cache", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(provinces)
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
