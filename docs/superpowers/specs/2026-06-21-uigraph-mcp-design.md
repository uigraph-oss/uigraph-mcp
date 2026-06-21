# uigraph-mcp Design Spec

**Date:** 2026-06-21  
**Status:** Approved  
**Module:** `github.com/uigraph/mcp`

---

## Problem

UIGraph holds rich engineering context: service catalog entries, API specs (OpenAPI / GraphQL / gRPC), database schemas, Mermaid architecture diagrams, test packs, and UI journey maps. All repo-synced content arrives via `uigraph-cli sync` as part of CI/CD — UIGraph is the structured, indexed, queryable version of artifacts already in source control.

When AI coding tools (Claude Code, Cursor) need architectural context, they autonomously explore the repository: finding and reading entire raw files, traversing directories, grepping for symbols. A full `openapi.yaml` might be 150 KB (≈37k tokens); a SQL schema file 80 KB. The AI reads the whole file even if it only needs to know which endpoints exist. Across multiple files and multiple conversations, this burns a large number of input tokens.

**Goal:** Build an MCP (Model Context Protocol) server that AI tools connect to and pull structured, pre-parsed UIGraph context on demand — instead of reading raw repo files. UIGraph has already indexed this data; MCP serves targeted, shaped answers rather than the full raw file. The server records raw usage metrics; uigraph-api owns all cost savings computation.

### Cost savings model

Two categories of savings:

| Content type | Without MCP | With MCP | Savings basis |
|---|---|---|---|
| Repo-synced (API specs, DB schemas, Mermaid diagrams) | AI reads entire raw files — the actual file token count is recorded at sync time | MCP serves structured subset | Actual file token count − MCP response tokens |
| UIGraph-native (Maps, Frames, Focal Points) | AI has no access; developer describes manually | MCP serves structured data | Multiplier estimate − MCP response tokens |

For repo-synced content, the savings estimate is **exact**: the CLI records the raw file token count at sync time and stores it alongside the spec. For UIGraph-native content (no repo file equivalent), a conservative multiplier is used.

---

## What we are building

`uigraph-mcp` is a standalone Go HTTP/SSE service implementing the Model Context Protocol. It:

1. Exposes UIGraph data as MCP tools that AI coding tools call directly
2. Fans out parallel requests to uigraph-api, assembles and shapes context for LLM consumption
3. Records raw usage metrics (tokens served, tokens raw equivalent) to uigraph-api
4. Is self-hostable in the same deployment pattern as uigraph-api

uigraph-api owns all cost computation: it joins usage events with the `llm_models` pricing table at query time, enabling accurate live cost savings for any model and any time range without re-recording.

---

## Architecture

### Transport

HTTP/SSE — the MCP standard for hosted servers.

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/sse` | `GET` | Client connects; server sends `event: endpoint` with session URL then streams `event: message` responses |
| `/message` | `POST` | Client sends JSON-RPC tool calls; server responds via the SSE stream |
| `/healthz` | `GET` | Liveness probe |

The server is fully stateless. No sessions stored. Every request carries `Authorization: Bearer <token>` which is threaded through to every uigraph-api call.

### Authentication

Two supported credential types, both verified by uigraph-api:

- **Service account tokens** — for CI/automation. Created in UIGraph, pasted into MCP client config.
- **User session tokens** — for individual developers. Obtained from uigraph-api's `/api/v1/auth/login` flow.

The MCP server never stores credentials. It passes the bearer token from the incoming request header directly to uigraph-api on every outbound call.

### Request flow

```
AI tool (Claude Code / Cursor)
  │  POST /message  {"method":"tools/call","params":{"name":"get_service_context",...}}
  │  Authorization: Bearer <token>
  ▼
uigraph-mcp HTTP/SSE server
  ├─ extract bearer token
  ├─ parse tool name + args
  ├─ fan out parallel calls to uigraph-api (all with same bearer)
  ├─ assemble + shape response for LLM consumption
  ├─ compute tokens_served + tokens_raw_equivalent (from actual file token counts or fallback multiplier)
  ├─ async POST /api/v1/orgs/{orgID}/mcp/usage  ← fire-and-forget goroutine
  └─ stream result back via SSE
```

The savings POST is fire-and-forget — zero latency added to the tool response.

---

## Project layout

Mirrors uigraph-api conventions exactly.

```
uigraph-mcp/
  cmd/mcp/
    main.go                   ← binary entrypoint
  internal/
    config/
      config.go               ← env-var config (UIGRAPH_API_URL, PORT, MCP_SERVER_NAME, MCP_SERVER_VERSION)
    apiclient/
      client.go               ← typed HTTP client for uigraph-api (base + auth header threading)
      catalog.go              ← service / api-group / endpoint / doc / db calls
      diagram.go              ← diagram calls
      maps.go                 ← map / frame calls
      usage.go                ← POST /api/v1/orgs/{orgID}/mcp/usage
    mcp/
      server.go               ← HTTP/SSE server, MCP handshake, tool dispatch
      tools.go                ← tool registry: names, descriptions, input schemas
      handler.go              ← per-tool handler wiring
    tools/
      service_context.go      ← get_service_context (fan-out assembler)
      catalog.go              ← list_services, get_service, list_api_groups, get_api_spec, list_endpoints
      diagrams.go             ← list_diagrams, get_diagram
      maps.go                 ← list_maps, get_map
      schemas.go              ← list_service_dbs, get_db_schema
      folders.go              ← list_folders
    tokencount/
      estimate.go             ← tokens_served counter + raw-equivalent logic (actual counts + fallback multipliers)
    server/
      server.go               ← HTTP server lifecycle (graceful shutdown)
  docs/
    superpowers/specs/
      2026-06-21-uigraph-mcp-design.md
  go.mod                      ← module: github.com/uigraph/mcp
  Dockerfile
  Dockerfile.dev
  .air.toml
  .gitignore
```

---

## MCP Tools catalog

### High-level composite

| Tool | Inputs | What it assembles |
|------|--------|-------------------|
| `get_service_context` | `org_id`, `service_id_or_slug`, `model_id?` | Service metadata + API groups with spec summaries + DB schemas + linked diagram names + doc list. Fans out ~5 parallel uigraph-api calls. `tokens_raw_equivalent` = sum of actual `spec_token_count` + `schema_token_count` + `content_token_count` for all fetched resources. `service_id_or_slug`: UUID used as ID directly; otherwise resolved via slug lookup. |

### Catalog tools

| Tool | Inputs | Returns | `tokens_raw_equivalent` source |
|------|--------|---------|-------------------------------|
| `list_services` | `org_id`, `folder_id?`, `team_id?` | id, name, slug, status, tier, language, description | multiplier (1.5×) |
| `get_service` | `org_id`, `service_id` | Full service + stats (endpoint/diagram/doc/db/test counts) | multiplier (1.5×) |
| `list_api_groups` | `org_id`, `service_id` | API groups: protocol, version, label | multiplier (1.5×) |
| `get_api_spec` | `org_id`, `api_group_id` | Full spec content from object storage. Truncated at 50k chars with note if over limit. | `api_groups.spec_token_count` (exact); fallback 4.0× |
| `list_endpoints` | `org_id`, `service_id`, `api_group_id?` | Flat list: method, path, summary, tags | `api_groups.spec_token_count` of parent group; fallback 3.0× |

### Diagram tools

| Tool | Inputs | Returns | `tokens_raw_equivalent` source |
|------|--------|---------|-------------------------------|
| `list_diagrams` | `org_id`, `folder_id?`, `team_id?` | id, name, created_at | multiplier (1.5×) |
| `get_diagram` | `org_id`, `diagram_id` | Mermaid/diagram content. Truncated at 100k chars. | `diagrams.content_token_count` (exact); fallback 2.0× |

### Map tools

| Tool | Inputs | Returns | `tokens_raw_equivalent` source |
|------|--------|---------|-------------------------------|
| `list_maps` | `org_id`, `folder_id?`, `team_id?` | id, name, status, description | multiplier (1.5×) |
| `get_map` | `org_id`, `map_id` | Map + all frames (name, description, template type, status, parent) + focal point counts | multiplier (2.0×) — UIGraph-native |

### DB Schema tools

| Tool | Inputs | Returns | `tokens_raw_equivalent` source |
|------|--------|---------|-------------------------------|
| `list_service_dbs` | `org_id`, `service_id` | id, db_name, db_type, dialect, updated_at | multiplier (1.5×) |
| `get_db_schema` | `org_id`, `service_db_id` | Full schema shaped as table list with columns, types, constraints. Truncated at 50k chars. | `service_dbs.schema_token_count` (exact); fallback 3.5× |

### Folder tools

| Tool | Inputs | Returns | `tokens_raw_equivalent` source |
|------|--------|---------|-------------------------------|
| `list_folders` | `org_id`, `type?` | id, name, type, parent_id, order | multiplier (1.5×) |

---

## Token estimation (uigraph-mcp side)

The `tokencount` package has no pricing logic — only token counting and raw-equivalent estimation.

```
tokens_served = len(responseJSON) / 4
```

For `tokens_raw_equivalent`:

1. **Exact path** (repo-synced content): use the actual file token count stored on the resource record in uigraph-api (`spec_token_count`, `schema_token_count`, `content_token_count`). This is the token cost the AI would have paid reading the raw file.
2. **Fallback path** (UIGraph-native content, or exact count not yet populated): `tokens_served × multiplier` using the per-tool constants in the table above.

```
tokens_saved = tokens_raw_equivalent - tokens_served
```

The MCP server records `tokens_served`, `tokens_raw_equivalent`, `tokens_saved`, `model_id`, and `resource_ids` to uigraph-api. **No cost computation happens in uigraph-mcp.**

---

## uigraph-api additions

### 1. New table: `llm_models`

Global pricing table. Server-admin managed. The source of truth for cost computation across all usage events and all time ranges.

```sql
CREATE TABLE llm_models (
  id                       TEXT PRIMARY KEY,
  model_id                 TEXT NOT NULL UNIQUE,   -- e.g. "claude-sonnet-4-6"
  provider                 TEXT NOT NULL,           -- "anthropic", "openai", "cursor", etc.
  display_name             TEXT NOT NULL,           -- "Claude Sonnet 4.6"
  input_cost_per_million   NUMERIC(10,4) NOT NULL,  -- USD per 1M input tokens
  output_cost_per_million  NUMERIC(10,4) NOT NULL,  -- USD per 1M output tokens (stored for completeness)
  is_active                BOOLEAN NOT NULL DEFAULT TRUE,
  created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Seeded on first migration with known models. Adding a new model = new row. Updating pricing = one UPDATE. No redeployment of uigraph-mcp needed.

**Initial seed data:**

| model_id | provider | display_name | input $/1M | output $/1M |
|----------|----------|--------------|-----------|------------|
| `claude-sonnet-4-6` | anthropic | Claude Sonnet 4.6 | 3.00 | 15.00 |
| `claude-opus-4-8` | anthropic | Claude Opus 4.8 | 15.00 | 75.00 |
| `claude-haiku-4-5` | anthropic | Claude Haiku 4.5 | 0.80 | 4.00 |
| `gpt-4o` | openai | GPT-4o | 2.50 | 10.00 |
| `cursor-default` | cursor | Cursor (default) | 3.00 | 15.00 |

### 2. `llm_models` API endpoints

| Method | Pattern | Auth | Purpose |
|--------|---------|------|---------|
| `GET` | `/api/v1/llm/models` | any authenticated | List active models + pricing (used by dashboard and MCP client config) |
| `POST` | `/api/v1/llm/models` | server-admin | Add a new model |
| `PUT` | `/api/v1/llm/models/{modelID}` | server-admin | Update pricing or display name |
| `DELETE` | `/api/v1/llm/models/{modelID}` | server-admin | Deactivate a model |

### 3. Token count columns on synced-content tables

The CLI records the raw file token count at sync time alongside the existing spec/schema content. All columns are nullable — existing records fall back to multiplier estimates until re-synced.

| Table | New column | Populated by |
|-------|-----------|-------------|
| `api_groups` | `spec_token_count INT` | `uigraph-cli sync` (APIs) |
| `service_dbs` | `schema_token_count INT` | `uigraph-cli sync` (databases) |
| `diagrams` | `content_token_count INT` | `uigraph-cli sync` (architectureDiagrams) |
| `service_docs` | `doc_token_count INT` | `uigraph-cli sync` (docs) |

Token count computation in the CLI: `len(fileContent) / 4` — same approximation used throughout.

### 4. New table: `mcp_usage_events`

Stores raw token metrics only. No pre-computed cost columns — costs are computed live at query time by joining with `llm_models`.

```sql
CREATE TABLE mcp_usage_events (
  id                    TEXT PRIMARY KEY,
  org_id                TEXT NOT NULL REFERENCES orgs(id),
  user_id               TEXT,                  -- null if service account
  service_account_id    TEXT,                  -- null if user session
  tool_name             TEXT NOT NULL,         -- e.g. "get_service_context"
  resource_ids          TEXT[],                -- service/api-group/diagram IDs fetched
  model_id              TEXT NOT NULL,         -- AI client reported model
  tokens_served         INT NOT NULL,          -- actual tokens in MCP response
  tokens_raw_equivalent INT NOT NULL,          -- exact file token count or multiplier estimate
  tokens_saved          INT NOT NULL,          -- tokens_raw_equivalent - tokens_served
  response_size_bytes   INT NOT NULL,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX mcp_usage_events_org_id_idx ON mcp_usage_events(org_id, created_at DESC);
```

### 5. New domain package: `internal/mcp/`

```go
type UsageEvent struct {
    ID                  string    `json:"id"`
    OrgID               string    `json:"orgId"`
    UserID              *string   `json:"userId,omitempty"`
    ServiceAccountID    *string   `json:"serviceAccountId,omitempty"`
    ToolName            string    `json:"toolName"`
    ResourceIDs         []string  `json:"resourceIds"`
    ModelID             string    `json:"modelId"`
    TokensServed        int       `json:"tokensServed"`
    TokensRawEquivalent int       `json:"tokensRawEquivalent"`
    TokensSaved         int       `json:"tokensSaved"`
    ResponseSizeBytes   int       `json:"responseSizeBytes"`
    CreatedAt           time.Time `json:"createdAt"`
}

// SavingsSummary is computed live at query time from mcp_usage_events × llm_models.
type SavingsSummary struct {
    OrgID              string  `json:"orgId"`
    Period             string  `json:"period"`              // "1d", "7d", "30d", "1y"
    ModelID            string  `json:"modelId"`             // pricing model used for computation
    TotalCalls         int     `json:"totalCalls"`
    TotalTokensServed  int     `json:"totalTokensServed"`
    TotalTokensSaved   int     `json:"totalTokensSaved"`
    CostServedUSD      float64 `json:"costServedUsd"`
    CostRawUSD         float64 `json:"costRawUsd"`
    CostSavedUSD       float64 `json:"costSavedUsd"`
    UniqueUsersCount   int     `json:"uniqueUsersCount"`
}
```

### 6. New API handler: `internal/api/mcp/`

```
internal/api/mcp/
  handler.go    ← Handler, Register(), narrow store interface
  usage.go      ← Record (POST) + List (GET) handlers
  summary.go    ← SavingsSummary (GET) handler — live SQL aggregation
```

| Method | Pattern | Scope | Purpose |
|--------|---------|-------|---------|
| `POST` | `/api/v1/orgs/{orgID}/mcp/usage` | `services:read` | MCP server records a tool call event |
| `GET` | `/api/v1/orgs/{orgID}/mcp/usage` | `services:read` | List events (paginated, filterable by `?tool=`, `?from=`, `?to=`) |
| `GET` | `/api/v1/orgs/{orgID}/mcp/savings/summary` | `services:read` | Aggregated savings; params: `?period=7d`, `?model_id=claude-sonnet-4-6` |

The summary endpoint computes costs live:

```sql
SELECT
  COUNT(*)                                                    AS total_calls,
  SUM(e.tokens_served)                                        AS total_tokens_served,
  SUM(e.tokens_saved)                                         AS total_tokens_saved,
  SUM(e.tokens_served)        / 1e6 * m.input_cost_per_million AS cost_served_usd,
  SUM(e.tokens_raw_equivalent)/ 1e6 * m.input_cost_per_million AS cost_raw_usd,
  SUM(e.tokens_saved)         / 1e6 * m.input_cost_per_million AS cost_saved_usd,
  COUNT(DISTINCT e.user_id)                                   AS unique_users_count
FROM mcp_usage_events e
CROSS JOIN llm_models m
WHERE e.org_id    = $1
  AND m.model_id  = $2          -- query param; can differ from recorded model_id
  AND e.created_at >= NOW() - $3::INTERVAL
```

This means: changing a model's price in `llm_models` immediately corrects all historical savings figures on the next dashboard load.

---

## Config (uigraph-mcp env vars)

Pricing is DB-owned — no `PRICING_*` env vars needed.

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `UIGRAPH_API_URL` | yes | — | Base URL of uigraph-api |
| `PORT` | no | `8080` | HTTP listen port |
| `MCP_SERVER_NAME` | no | `uigraph-mcp` | Reported in MCP `initialize` response |
| `MCP_SERVER_VERSION` | no | `0.1.0` | Reported in MCP `initialize` response |

---

## Deployment

Follows uigraph-api's existing patterns.

- `Dockerfile` — multi-stage Go build, scratch final image
- `Dockerfile.dev` — with `air` for hot reload
- `.air.toml` — mirrors uigraph-api
- Added to `uigraph-deploy/docker-compose.yml` and `k8s.yaml` as a new service

### Claude Code MCP client config

```json
{
  "mcpServers": {
    "uigraph": {
      "url": "http://uigraph-mcp.your-cluster.internal:8080/sse",
      "headers": {
        "Authorization": "Bearer <your-token>"
      }
    }
  }
}
```

---

## Out of scope for v1

- OAuth PKCE interactive login flow in the MCP server itself
- In-process caching layer (uigraph-api already caches internally)
- MCP Resources and Prompts primitives (Tools only for v1)
- Test case / test run tools (can be added as v2 tools)
- Webhook-based savings notifications
- Output token tracking in usage events (input tokens are what MCP context delivery affects)
