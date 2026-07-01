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
| `list_stores` | Physical store directory; `query` resolves a name/location (e.g. `grancasa`) to a `store_id`. | `casadellibro.com/cdlweb/api/libreria/todasTiendas` |
| `get_store_stock` | Per-bookstore stock + pickup availability grouped by province; optional `store_id` / `in_stock_only`. | `casadellibro.com/cdlweb/api/libreria/stockTiendas` |
| `find_books_in_store` | Search + per-store stock join: books matching a query/filters that are actually in stock at one `store_id`. | both search + stockTiendas |

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

To answer "is X available at store Y", resolve the store with `list_stores`
(e.g. `query: "grancasa"` → `store_id: 38`), then either `get_store_stock` with that
`store_id`, or `find_books_in_store` which does the search + per-store stock join
server-side. To browse a whole publisher/collection at a store, pass the facet value
as the `query` (e.g. `"Unión Editorial"`) plus its filter — the catalog requires a
non-empty query, but querying the facet's own name matches its full set.

Pagination: `search_books` uses `start`/`rows` and returns `next_start` + `has_more`.
`find_books_in_store` is O(N) stock calls, so it paginates over catalog candidates —
it scans `limit` candidates from `start` and returns `next_start` + `has_more`; loop
by passing the previous `next_start` as `start` until `has_more` is false. It
de-duplicates by `product_id` within a page (the empathy search occasionally repeats
an item across page boundaries, and only exposes roughly the first ~2000 matches).

`search_books` and `get_store_stock` require a `fields` array that projects the
response down to the fields you ask for (book fields and bookstore fields
respectively). It is mandatory on purpose: it forces the caller to request only
what it needs, keeping responses small. Unknown or empty field lists are rejected
with the list of valid fields.

Every tool declares an `outputSchema` and returns `structuredContent` (JSON,
projected to the requested `fields`) so schema-aware clients understand the shape
of the result. `structuredContent` is always a JSON object: `search_books` returns
`{total, start, rows, books[]}`, `get_store_stock` `{provinces[]}`, and
`search_books_available_filters` `{facets[]}`.

For clients that read plain text instead, the same result is mirrored as a compact
tab-separated fallback where the field names appear once in a header row: `search_books`
prepends a `total=… start=… rows=…` summary line; `get_store_stock` adds a leading
`province` column; `search_books_available_filters` groups values under
`# <facet> [<type>]` headers, one `<filter string>\t(<count>)` per line.

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
