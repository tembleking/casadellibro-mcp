# casadellibro-mcp

MCP server exposing the [casadellibro](https://www.casadellibro.com) catalog and
per-store stock as Model Context Protocol tools.

## Tools

| Tool | Description | Backing endpoint |
|------|-------------|------------------|
| `search_books` | Free-text catalog search (price, availability, ISBN, `product_id`). | `api.empathy.co/search/v1/query/cdl/search` |
| `get_store_stock` | Per-bookstore stock + pickup availability grouped by province. | `casadellibro.com/cdlweb/api/libreria/stockTiendas` |

`search_books` returns a `product_id` that feeds straight into `get_store_stock`.

## Architecture (clean architecture)

Dependencies point inward; `domain` knows nothing about HTTP or MCP.

```
cmd/app                       process entry point
└─ internal/
   ├─ domain/                 entities + repository interfaces (ports)
   ├─ usecase/                application rules (validation, defaults)
   ├─ infrastructure/
   │  └─ casadellibro/        HTTP adapters implementing the ports
   ├─ delivery/
   │  ├─ mcp/                 mark3labs/mcp-go server + tool wiring
   │  └─ cli/                 cobra command tree (serve, version)
   └─ mocks/                  gomock-generated repository mocks
```

- **Ports**: `domain.CatalogRepository`, `domain.StockRepository`.
- **Adapters**: `casadellibro.CatalogAdapter`, `casadellibro.StockAdapter`.
- **Use cases** depend only on the ports, so they are unit-tested with mocks.

## Stack

- `github.com/mark3labs/mcp-go` — MCP server (stdio transport)
- `github.com/spf13/cobra` — CLI
- `github.com/onsi/ginkgo/v2` + `github.com/onsi/gomega` — BDD tests
- `go.uber.org/mock` — generated mocks for the ports
- `just` — task runner

## Usage

```sh
just build          # -> bin/casadellibro-mcp
just serve          # run the MCP server over stdio
just test           # ginkgo -r
just check          # tidy + generate + lint + test
```

### Transports

`serve` defaults to stdio; pick another with `--transport`/`-t` and set the
listen address for the network transports with `--addr`/`-a` (default `:8080`):

```sh
casadellibro-mcp serve                       # stdio (default)
casadellibro-mcp serve -t sse  -a :8080       # SSE        -> /sse, /message
casadellibro-mcp serve -t http -a :8080       # streamable -> /mcp
```

MCP client config:

```json
{
  "mcpServers": {
    "casadellibro": {
      "command": "/path/to/bin/casadellibro-mcp",
      "args": ["serve"]
    }
  }
}
```
