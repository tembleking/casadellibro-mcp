// Package cli builds the cobra command tree for the casadellibro MCP binary.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"app/internal/delivery/mcp"
	"app/internal/infrastructure/casadellibro"
	"app/internal/usecase"
)

// Supported MCP transports.
const (
	transportStdio = "stdio"
	transportSSE   = "sse"
	transportHTTP  = "http"
)

// Build version, overridable at link time with -ldflags "-X ...".
var (
	appName = "casadellibro-mcp"
	version = "0.1.0"
)

// NewRootCmd builds the root command and its subcommands.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   appName,
		Short: "MCP server for the casadellibro catalog and store stock",
		Long:  "casadellibro-mcp exposes the casadellibro search catalog and per-store stock as Model Context Protocol tools.",
	}
	root.AddCommand(newServeCmd(), newVersionCmd())
	return root
}

func newServeCmd() *cobra.Command {
	var (
		transport string
		addr      string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the MCP server",
		Long:  "Run the MCP server over the stdio (default), sse or http transport.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := casadellibro.NewClient()
			catalog := casadellibro.NewCatalogAdapter(client)
			handlers := mcp.Handlers{
				Search:  usecase.NewSearchBooks(catalog),
				Filters: usecase.NewListSearchFilters(catalog),
				Stock:   usecase.NewGetStoreStock(casadellibro.NewStockAdapter(client)),
			}
			srv := mcp.NewServer(appName, version, handlers)

			// Honor $PORT (set by most PaaS, e.g. Render) when --addr was not given.
			if !cmd.Flags().Changed("addr") {
				if port := os.Getenv("PORT"); port != "" {
					addr = ":" + port
				}
			}

			switch transport {
			case transportStdio:
				return mcp.ServeStdio(srv)
			case transportSSE:
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s listening (sse) on %s\n", appName, addr)
				return mcp.ServeSSE(srv, addr)
			case transportHTTP:
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s listening (http) on %s\n", appName, addr)
				return mcp.ServeHTTP(srv, addr)
			default:
				return fmt.Errorf("unknown transport %q (want %s, %s or %s)", transport, transportStdio, transportSSE, transportHTTP)
			}
		},
	}

	cmd.Flags().StringVarP(&transport, "transport", "t", transportStdio, "transport: stdio, sse or http")
	cmd.Flags().StringVarP(&addr, "addr", "a", ":8080", "listen address for sse/http transports")
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, _ []string) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", appName, version)
		},
	}
}
