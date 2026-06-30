# casadellibro-mcp

MCP server exposing the [casadellibro](https://www.casadellibro.com) catalog and
per-store stock as Model Context Protocol tools.

> **Disclaimer**: This is an independent hobby project and is **not affiliated
> with, endorsed by, or connected to Casa del Libro** (or its operators) in any
> way. It merely consumes publicly available endpoints. All trademarks belong to
> their respective owners.

## Tools

| Tool | Description | Backing endpoint |
|------|-------------|------------------|
| `search_books_available_filters` | Discover the filters available for a query (language, binding, availability, price ranges, publisher…), each value with a ready-to-use filter string. | `api.empathy.co/search/v1/query/cdl/facets` |
| `search_books` | Free-text catalog search (price, availability, ISBN, `product_id`), narrowable with the filter strings above. | `api.empathy.co/search/v1/query/cdl/search` |
| `get_store_stock` | Per-bookstore stock + pickup availability grouped by province. | `casadellibro.com/cdlweb/api/libreria/stockTiendas` |

Typical flow: call `search_books_available_filters` to see what filters apply to a
query, then pass the exact `filter` strings it returns (e.g. `availability:Con stock`,
`facetLang:Castellano`, `priceOffer:8.0-14.0`) in the `search_books` `filters` argument —
they combine with AND. `search_books` returns a `product_id` that feeds straight into
`get_store_stock`.

Every `search_books` field describes the online catalog listing, not a physical
store. In particular `availability` is catalog-wide ("Con stock" means at least one
store/warehouse has it, **not** that any particular store does) and `price` is the
online price (a physical store may not match it). Use `get_store_stock` for per-store
stock and pickup.

`search_books` and `get_store_stock` accept an optional `fields` array to project
the response down to the fields you need (book fields and bookstore fields
respectively); omit it to get everything. Use it to keep responses small and save
tokens. Unknown field names are rejected with the list of valid fields.

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

## Deploy

The image is built by nix (`docker.nix`, exposed as the `dockerImage` flake
package) and published to GHCR by CI. It runs on any container host: the binary
binds to `$PORT` (injected by the host) when `--addr` is not passed, and serves
the streamable HTTP transport at `/mcp`.

Build the image locally (on Linux, or a Linux remote builder):

```sh
nix build .#dockerImage      # -> ./result (a docker-loadable tarball)
```

Pipeline:

1. `.github/workflows/image.yml` triggers on pushes to `master` that touch
   `package.nix`, and **only builds when the `version` actually changes**. It
   `nix build .#dockerImage` and pushes `ghcr.io/tembleking/casadellibro-mcp`
   tagged with that version and `latest`.
2. Make the GHCR package **public** (or give your host registry credentials).
3. Run the image on your host of choice, exposing port `$PORT` over HTTPS.
4. The MCP endpoint is `https://<your-host>/mcp`.

To auto-redeploy on each new image, set a deploy-hook URL as the repo secret
`DEPLOY_HOOK_URL`; CI calls it after pushing the image.

### Releasing a new version

Bump `version` in `package.nix`, commit to `master`. CI builds and pushes the
new image tag; the deploy hook (if set) redeploys.

### Use it from ChatGPT

Custom MCP connectors require a **paid ChatGPT plan** (Plus/Pro/Business/Enterprise)
and **Developer mode** enabled.

1. ChatGPT → **Settings → Connectors → Advanced → Developer mode**.
2. **Add custom connector** → URL `https://<your-host>/mcp`,
   authentication **None**.
3. `search_books` and `get_store_stock` then appear as tools in the composer.
