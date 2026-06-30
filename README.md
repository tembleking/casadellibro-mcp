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

MCP client config (local stdio):

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

## Deploy (Render, free tier)

The `http` transport plus a `Dockerfile` and `render.yaml` make this deployable
as a public HTTPS endpoint. The binary binds to `$PORT` (injected by the host)
when `--addr` is not passed.

1. Push this repo to GitHub.
2. On [Render](https://render.com): **New → Blueprint**, point it at the repo.
   `render.yaml` provisions a free Docker web service. Every push auto-deploys.
3. The MCP endpoint is `https://<service>.onrender.com/mcp`.

Note: the Render free tier sleeps after inactivity, so the first request after
idle has a cold start of ~30–60s.

### Use it from ChatGPT

Custom MCP connectors require a **paid ChatGPT plan** (Plus/Pro/Business/Enterprise)
and **Developer mode** enabled.

1. ChatGPT → **Settings → Connectors → Advanced → Developer mode**.
2. **Add custom connector** → URL `https://<service>.onrender.com/mcp`,
   authentication **None**.
3. `search_books` and `get_store_stock` then appear as tools in the composer.
