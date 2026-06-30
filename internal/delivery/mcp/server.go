// Package mcp wires the use cases into a mark3labs/mcp-go server exposing the
// casadellibro tools over stdio.
package mcp

import (
	"github.com/mark3labs/mcp-go/server"

	"app/internal/usecase"
)

// Handlers groups the use cases the MCP server exposes.
type Handlers struct {
	Search  *usecase.SearchBooks
	Filters *usecase.ListSearchFilters
	Stock   *usecase.GetStoreStock
}

// NewServer builds an MCP server with the casadellibro tools registered.
func NewServer(name, version string, h Handlers) *server.MCPServer {
	s := server.NewMCPServer(
		name,
		version,
		server.WithToolCapabilities(true),
	)
	registerFiltersTool(s, h.Filters)
	registerSearchTool(s, h.Search)
	registerStockTool(s, h.Stock)
	return s
}

// ServeStdio runs the MCP server over stdio until the process is stopped.
func ServeStdio(s *server.MCPServer) error {
	return server.ServeStdio(s)
}

// ServeSSE runs the MCP server over the SSE transport on the given address.
func ServeSSE(s *server.MCPServer, addr string) error {
	return server.NewSSEServer(s).Start(addr)
}

// ServeHTTP runs the MCP server over the streamable HTTP transport on the given address.
func ServeHTTP(s *server.MCPServer, addr string) error {
	return server.NewStreamableHTTPServer(s).Start(addr)
}
