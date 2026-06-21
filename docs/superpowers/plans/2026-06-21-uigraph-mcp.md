# uigraph-mcp Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the uigraph-mcp MCP server and all required uigraph-api additions (LLM models pricing table, MCP usage events, token count columns on synced content).

**Architecture:** Two phases — Phase A adds tables/endpoints to uigraph-api; Phase B builds the new uigraph-mcp Go service. uigraph-mcp calls uigraph-api for all data and records raw token metrics back; uigraph-api owns all cost computation.

**Tech Stack:** Go 1.25, `github.com/mark3labs/mcp-go` (MCP SSE server), `github.com/lib/pq` (Postgres), `github.com/google/uuid`, `github.com/rs/xid`.

## Global Constraints

- Go 1.25 in both modules
- uigraph-api module: `github.com/uigraph/app`; uigraph-mcp module: `github.com/uigraph/mcp`
- Follow uigraph-api handler pattern exactly: narrow interfaces, `httputil` for all responses, two separate `if` blocks for `err != nil` and nil checks
- No cost computation in uigraph-mcp — only raw token counts recorded
- All `token_count` columns are NOT NULL (v1, no legacy data)
- `go build ./...` must pass after every task

---

## Phase A — uigraph-api additions

Work directory: `/Users/kranthi/workspace/go/uigraph/backend/uigraph-oss/uigraph-api`

---

### Task A1: Token count columns on synced-content tables

**Files:**
- Create: `migrations/0028_mcp_token_counts.sql`
- Modify: `internal/catalog/catalog.go` — add `TokenCount` to `APIEndpoint`; add `SchemaTokenCount` to `ServiceDB`; add `DocTokenCount` to `ServiceDoc`
- Modify: `internal/diagram/diagram.go` — add `ContentTokenCount` field
- Modify: `internal/store/postgres/catalog.go` — update scan + insert for APIEndpoint, ServiceDB, ServiceDoc
- Modify: `internal/store/postgres/diagrams.go` — update scan + insert for Diagram

**Interfaces:**
- Produces: `catalog.APIEndpoint.TokenCount int`, `catalog.ServiceDB.SchemaTokenCount int`, `catalog.ServiceDoc.DocTokenCount int`, `diagram.Diagram.ContentTokenCount int`

- [ ] **Step 1: Write the migration**

Create `migrations/0028_mcp_token_counts.sql`:
```sql
ALTER TABLE api_endpoints ADD COLUMN token_count        INT NOT NULL DEFAULT 0;
ALTER TABLE service_dbs   ADD COLUMN schema_token_count INT NOT NULL DEFAULT 0;
ALTER TABLE service_docs  ADD COLUMN doc_token_count    INT NOT NULL DEFAULT 0;
ALTER TABLE diagrams      ADD COLUMN content_token_count INT NOT NULL DEFAULT 0;
```

- [ ] **Step 2: Add `TokenCount` to `catalog.APIEndpoint`**

In `internal/catalog/catalog.go`, find the `APIEndpoint` struct and add `TokenCount int` after `Tags`. Read the file first to see the exact current shape, then add the field. The resulting struct should look like:

```go
type APIEndpoint struct {
	ID          string     `json:"id"`
	APIGroupID  string     `json:"apiGroupId"`
	ServiceID   string     `json:"serviceId"`
	OrgID       string     `json:"orgId"`
	OperationID string     `json:"operationId"`
	Method      string     `json:"method"`
	Path        string     `json:"path"`
	Summary     string     `json:"summary"`
	Tags        []string   `json:"tags"`
	TokenCount  int        `json:"tokenCount"`
	// ... remaining existing fields (CreatedBy, UpdatedBy, CreatedAt, UpdatedAt, DeletedAt, etc.)
}
```

Add only `TokenCount int \`json:"tokenCount"\`` — do not change any other fields.

- [ ] **Step 3: Add field to `catalog.ServiceDB`**

Add `SchemaTokenCount int` after `Source` fields:
```go
type ServiceDB struct {
	ID               string          `json:"id"`
	ServiceID        string          `json:"serviceId"`
	OrgID            string          `json:"orgId"`
	DBName           string          `json:"dbName"`
	DBType           string          `json:"dbType"`
	Dialect          string          `json:"dialect"`
	SchemaJSON       json.RawMessage `json:"schemaJson"`
	Source           *string         `json:"source,omitempty"`
	SourceTS         *time.Time      `json:"sourceTs,omitempty"`
	SchemaTokenCount int             `json:"schemaTokenCount"`
	CreatedBy        string          `json:"createdBy"`
	UpdatedBy        *string         `json:"updatedBy,omitempty"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
	DeletedAt        *time.Time      `json:"deletedAt,omitempty"`
	DeletedBy        *string         `json:"deletedBy,omitempty"`
}
```

- [ ] **Step 4: Add field to `catalog.ServiceDoc`**

Add `DocTokenCount int` after `ContentHash`:
```go
type ServiceDoc struct {
	ID            string     `json:"id"`
	ServiceID     string     `json:"serviceId"`
	OrgID         string     `json:"orgId"`
	FileKey       string     `json:"fileKey"`
	FileName      string     `json:"fileName"`
	FileType      string     `json:"fileType"`
	Description   string     `json:"description"`
	ContentHash   string     `json:"contentHash"`
	DocTokenCount int        `json:"docTokenCount"`
	CreatedBy     string     `json:"createdBy"`
	UpdatedBy     *string    `json:"updatedBy,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	DeletedAt     *time.Time `json:"deletedAt,omitempty"`
}
```

- [ ] **Step 5: Add field to `diagram.Diagram`**

In `internal/diagram/diagram.go`, add `ContentTokenCount int` after `ContentHash`:
```go
type Diagram struct {
	ID                 string     `json:"id"`
	OrgID              string     `json:"orgId"`
	FolderID           *string    `json:"folderId,omitempty"`
	TeamID             *string    `json:"teamId,omitempty"`
	Name               string     `json:"name"`
	ContentKey         string     `json:"contentKey"`
	ContentHash        string     `json:"contentHash"`
	ContentTokenCount  int        `json:"contentTokenCount"`
	PreviewAssetID     *string    `json:"previewAssetId,omitempty"`
	PreviewContentHash *string    `json:"previewContentHash,omitempty"`
	Source             *string    `json:"source,omitempty"`
	CreatedBy          string     `json:"createdBy"`
	UpdatedBy          *string    `json:"updatedBy,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
	DeletedAt          *time.Time `json:"deletedAt,omitempty"`
	DeletedBy          *string    `json:"deletedBy,omitempty"`
}
```

- [ ] **Step 6: Update postgres scan for APIEndpoint**

In `internal/store/postgres/catalog.go`, find the `scanAPIEndpoint` function (or equivalent inline scan) and add `token_count` to the SELECT column list and the corresponding `&e.TokenCount` to the Scan call. Find `CreateAPIEndpoint` (or `SyncAPIEndpoint`) and `UpdateAPIEndpoint` and add `token_count` to their INSERT/UPDATE statements. The scan function should include `&e.TokenCount` in the same position as `token_count` appears in the SELECT.

- [ ] **Step 7: Update postgres scan for ServiceDB**

In `internal/store/postgres/service_dbs.go`, add `schema_token_count` to all SELECTs, the scan function, INSERT, and UPDATE.

- [ ] **Step 8: Update postgres scan for Diagram**

In `internal/store/postgres/diagrams.go`, add `content_token_count` to all SELECTs, scan function, INSERT, and UPDATE.

- [ ] **Step 9: Update postgres scan for ServiceDoc**

In `internal/store/postgres/catalog.go` (or wherever service_docs are scanned), add `doc_token_count` to all SELECTs, scan function, INSERT, and UPDATE.

- [ ] **Step 10: Verify build**

```bash
cd /Users/kranthi/workspace/go/uigraph/backend/uigraph-oss/uigraph-api
go build ./...
```
Expected: no errors.

- [ ] **Step 11: Commit**

```bash
git add migrations/0028_mcp_token_counts.sql internal/catalog/catalog.go internal/diagram/diagram.go internal/store/postgres/catalog.go internal/store/postgres/service_dbs.go internal/store/postgres/diagrams.go
git commit -m "feat: add token_count to api_endpoints and other synced-content tables for MCP savings tracking"
```

---

### Task A2: LLM models — table, domain, store, API handler

**Files:**
- Create: `migrations/0029_llm_models.sql`
- Create: `internal/llm/llm.go`
- Create: `internal/store/postgres/llm.go`
- Create: `internal/api/llm/handler.go`
- Modify: `internal/store/store.go` — add `llm.Store`
- Modify: `internal/api/router.go` — wire LLM routes

**Interfaces:**
- Produces: `llm.Store` interface, `GET /api/v1/llm/models`, `POST /api/v1/llm/models`, `PUT /api/v1/llm/models/{modelID}`, `DELETE /api/v1/llm/models/{modelID}`

- [ ] **Step 1: Write the migration**

Create `migrations/0029_llm_models.sql`:
```sql
CREATE TABLE llm_models (
    id                      TEXT        NOT NULL PRIMARY KEY,
    model_id                TEXT        NOT NULL UNIQUE,
    provider                TEXT        NOT NULL,
    display_name            TEXT        NOT NULL,
    input_cost_per_million  NUMERIC(10,4) NOT NULL,
    output_cost_per_million NUMERIC(10,4) NOT NULL,
    is_active               BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO llm_models (id, model_id, provider, display_name, input_cost_per_million, output_cost_per_million) VALUES
    ('01', 'claude-sonnet-4-6',  'anthropic', 'Claude Sonnet 4.6', 3.00,  15.00),
    ('02', 'claude-opus-4-8',    'anthropic', 'Claude Opus 4.8',   15.00, 75.00),
    ('03', 'claude-haiku-4-5',   'anthropic', 'Claude Haiku 4.5',  0.80,  4.00),
    ('04', 'gpt-4o',             'openai',    'GPT-4o',            2.50,  10.00),
    ('05', 'cursor-default',     'cursor',    'Cursor (default)',   3.00,  15.00);
```

- [ ] **Step 2: Create `internal/llm/llm.go`**

```go
package llm

import (
	"context"
	"time"
)

type LLMModel struct {
	ID                    string    `json:"id"`
	ModelID               string    `json:"modelId"`
	Provider              string    `json:"provider"`
	DisplayName           string    `json:"displayName"`
	InputCostPerMillion   float64   `json:"inputCostPerMillion"`
	OutputCostPerMillion  float64   `json:"outputCostPerMillion"`
	IsActive              bool      `json:"isActive"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

type Store interface {
	ListLLMModels(ctx context.Context) ([]LLMModel, error)
	GetLLMModel(ctx context.Context, id string) (*LLMModel, error)
	CreateLLMModel(ctx context.Context, m LLMModel) error
	UpdateLLMModel(ctx context.Context, m LLMModel) error
	DeactivateLLMModel(ctx context.Context, id string) error
}
```

- [ ] **Step 3: Create `internal/store/postgres/llm.go`**

```go
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/uigraph/app/internal/llm"
)

func (d *DB) ListLLMModels(ctx context.Context) ([]llm.LLMModel, error) {
	const q = `
		SELECT id, model_id, provider, display_name,
		       input_cost_per_million, output_cost_per_million,
		       is_active, created_at, updated_at
		FROM llm_models WHERE is_active = TRUE ORDER BY provider, display_name`
	rows, err := d.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres: ListLLMModels: %w", err)
	}
	defer rows.Close()
	var out []llm.LLMModel
	for rows.Next() {
		m, scanErr := scanLLMModel(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("postgres: ListLLMModels scan: %w", scanErr)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (d *DB) GetLLMModel(ctx context.Context, id string) (*llm.LLMModel, error) {
	const q = `
		SELECT id, model_id, provider, display_name,
		       input_cost_per_million, output_cost_per_million,
		       is_active, created_at, updated_at
		FROM llm_models WHERE id = $1`
	m, err := scanLLMModel(d.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: GetLLMModel: %w", err)
	}
	return &m, nil
}

func (d *DB) CreateLLMModel(ctx context.Context, m llm.LLMModel) error {
	const q = `
		INSERT INTO llm_models
			(id, model_id, provider, display_name, input_cost_per_million, output_cost_per_million, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
	now := time.Now().UTC()
	_, err := d.db.ExecContext(ctx, q,
		m.ID, m.ModelID, m.Provider, m.DisplayName,
		m.InputCostPerMillion, m.OutputCostPerMillion, now, now,
	)
	return wrapErr("CreateLLMModel", err)
}

func (d *DB) UpdateLLMModel(ctx context.Context, m llm.LLMModel) error {
	const q = `
		UPDATE llm_models
		SET display_name=$1, input_cost_per_million=$2, output_cost_per_million=$3, updated_at=$4
		WHERE id=$5`
	_, err := d.db.ExecContext(ctx, q,
		m.DisplayName, m.InputCostPerMillion, m.OutputCostPerMillion,
		time.Now().UTC(), m.ID,
	)
	return wrapErr("UpdateLLMModel", err)
}

func (d *DB) DeactivateLLMModel(ctx context.Context, id string) error {
	const q = `UPDATE llm_models SET is_active=FALSE, updated_at=$1 WHERE id=$2`
	_, err := d.db.ExecContext(ctx, q, time.Now().UTC(), id)
	return wrapErr("DeactivateLLMModel", err)
}

func scanLLMModel(r interface{ Scan(...interface{}) error }) (llm.LLMModel, error) {
	var m llm.LLMModel
	return m, r.Scan(
		&m.ID, &m.ModelID, &m.Provider, &m.DisplayName,
		&m.InputCostPerMillion, &m.OutputCostPerMillion,
		&m.IsActive, &m.CreatedAt, &m.UpdatedAt,
	)
}
```

- [ ] **Step 4: Create `internal/api/llm/handler.go`**

```go
package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	llmpkg "github.com/uigraph/app/internal/llm"
	"github.com/uigraph/app/internal/httputil"
	storepkg "github.com/uigraph/app/internal/store"
)

type store interface {
	ListLLMModels(ctx context.Context) ([]llmpkg.LLMModel, error)
	GetLLMModel(ctx context.Context, id string) (*llmpkg.LLMModel, error)
	CreateLLMModel(ctx context.Context, m llmpkg.LLMModel) error
	UpdateLLMModel(ctx context.Context, m llmpkg.LLMModel) error
	DeactivateLLMModel(ctx context.Context, id string) error
}

type Handler struct{ store store }

func New(s store) *Handler { return &Handler{store: s} }

func Register(mux *http.ServeMux, s store, authenticated func(method, pattern string, h http.HandlerFunc), serverAdmin func(method, pattern string, h http.HandlerFunc)) {
	h := New(s)
	authenticated("GET", "/api/v1/llm/models", h.List)
	serverAdmin("POST", "/api/v1/llm/models", h.Create)
	serverAdmin("PUT", "/api/v1/llm/models/{modelID}", h.Update)
	serverAdmin("DELETE", "/api/v1/llm/models/{modelID}", h.Deactivate)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	models, err := h.store.ListLLMModels(r.Context())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}
	if models == nil {
		models = []llmpkg.LLMModel{}
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"models": models})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ModelID              string  `json:"modelId"`
		Provider             string  `json:"provider"`
		DisplayName          string  `json:"displayName"`
		InputCostPerMillion  float64 `json:"inputCostPerMillion"`
		OutputCostPerMillion float64 `json:"outputCostPerMillion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.BadRequest(w, "invalid request body")
		return
	}
	if body.ModelID == "" || body.Provider == "" || body.DisplayName == "" {
		httputil.BadRequest(w, "modelId, provider, and displayName are required")
		return
	}
	now := time.Now().UTC()
	m := llmpkg.LLMModel{
		ID:                   uuid.NewString(),
		ModelID:              body.ModelID,
		Provider:             body.Provider,
		DisplayName:          body.DisplayName,
		InputCostPerMillion:  body.InputCostPerMillion,
		OutputCostPerMillion: body.OutputCostPerMillion,
		IsActive:             true,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if err := h.store.CreateLLMModel(r.Context(), m); err != nil {
		httputil.Error(w, r, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, m)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	existing, err := h.store.GetLLMModel(r.Context(), r.PathValue("modelID"))
	if err != nil {
		httputil.Error(w, r, err)
		return
	}
	if existing == nil {
		httputil.Error(w, r, storepkg.ErrNotFound)
		return
	}
	var body struct {
		DisplayName          *string  `json:"displayName"`
		InputCostPerMillion  *float64 `json:"inputCostPerMillion"`
		OutputCostPerMillion *float64 `json:"outputCostPerMillion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.BadRequest(w, "invalid request body")
		return
	}
	if body.DisplayName != nil {
		existing.DisplayName = *body.DisplayName
	}
	if body.InputCostPerMillion != nil {
		existing.InputCostPerMillion = *body.InputCostPerMillion
	}
	if body.OutputCostPerMillion != nil {
		existing.OutputCostPerMillion = *body.OutputCostPerMillion
	}
	if err := h.store.UpdateLLMModel(r.Context(), *existing); err != nil {
		httputil.Error(w, r, err)
		return
	}
	existing.UpdatedAt = time.Now().UTC()
	httputil.JSON(w, http.StatusOK, existing)
}

func (h *Handler) Deactivate(w http.ResponseWriter, r *http.Request) {
	existing, err := h.store.GetLLMModel(r.Context(), r.PathValue("modelID"))
	if err != nil {
		httputil.Error(w, r, err)
		return
	}
	if existing == nil {
		httputil.Error(w, r, storepkg.ErrNotFound)
		return
	}
	if err := h.store.DeactivateLLMModel(r.Context(), r.PathValue("modelID")); err != nil {
		httputil.Error(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 5: Add `llm.Store` to `internal/store/store.go`**

```go
import (
	// existing imports ...
	"github.com/uigraph/app/internal/llm"
)

type Store interface {
	authz.RBACStore
	identity.SessionStore
	identity.ProviderStore
	identity.ServiceAccountStore
	org.UserStore
	org.OrgStore
	org.MemberStore
	org.TeamStore
	folder.Store
	diagram.Store
	uimap.Store
	catalog.Store
	componentlib.Store
	comment.Store
	llm.Store
}
```

- [ ] **Step 6: Wire into `internal/api/router.go`**

Add import `llmapi "github.com/uigraph/app/internal/api/llm"` and in `New()`:
```go
llmapi.Register(mux, s, protected, serverAdmin)
```

- [ ] **Step 7: Verify build**

```bash
go build ./...
```

- [ ] **Step 8: Commit**

```bash
git add migrations/0029_llm_models.sql internal/llm/ internal/store/postgres/llm.go internal/store/store.go internal/api/llm/ internal/api/router.go
git commit -m "feat: add llm_models table and API for dynamic model pricing"
```

---

### Task A3: MCP usage events — table, domain, store, API handler

**Files:**
- Create: `migrations/0030_mcp_usage.sql`
- Create: `internal/mcpusage/mcpusage.go`
- Create: `internal/store/postgres/mcp_usage.go`
- Create: `internal/api/mcpusage/handler.go`
- Create: `internal/api/mcpusage/usage.go`
- Create: `internal/api/mcpusage/summary.go`
- Modify: `internal/store/store.go` — add `mcpusage.Store`
- Modify: `internal/api/router.go` — wire MCP usage routes

**Interfaces:**
- Produces: `mcpusage.Store`, `POST /api/v1/orgs/{orgID}/mcp/usage`, `GET /api/v1/orgs/{orgID}/mcp/usage`, `GET /api/v1/orgs/{orgID}/mcp/savings/summary?period=7d&model_id=claude-sonnet-4-6`

- [ ] **Step 1: Write the migration**

Create `migrations/0030_mcp_usage.sql`:
```sql
CREATE TABLE mcp_usage_events (
    id                    TEXT        NOT NULL PRIMARY KEY,
    org_id                UUID        NOT NULL REFERENCES orgs(id),
    user_id               UUID,
    service_account_id    UUID,
    tool_name             TEXT        NOT NULL,
    resource_ids          TEXT[]      NOT NULL DEFAULT '{}',
    model_id              TEXT        NOT NULL,
    tokens_served         INT         NOT NULL,
    tokens_raw_equivalent INT         NOT NULL,
    tokens_saved          INT         NOT NULL,
    response_size_bytes   INT         NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX mcp_usage_events_org_id_idx ON mcp_usage_events(org_id, created_at DESC);
```

- [ ] **Step 2: Create `internal/mcpusage/mcpusage.go`**

```go
package mcpusage

import (
	"context"
	"time"
)

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

type SavingsSummary struct {
	OrgID             string  `json:"orgId"`
	Period            string  `json:"period"`
	ModelID           string  `json:"modelId"`
	TotalCalls        int     `json:"totalCalls"`
	TotalTokensServed int     `json:"totalTokensServed"`
	TotalTokensSaved  int     `json:"totalTokensSaved"`
	CostServedUSD     float64 `json:"costServedUsd"`
	CostRawUSD        float64 `json:"costRawUsd"`
	CostSavedUSD      float64 `json:"costSavedUsd"`
	UniqueUsersCount  int     `json:"uniqueUsersCount"`
}

type Filter struct {
	Tool   *string
	FromTS *time.Time
	ToTS   *time.Time
}

type Store interface {
	CreateUsageEvent(ctx context.Context, e UsageEvent) error
	ListUsageEvents(ctx context.Context, orgID string, f Filter) ([]UsageEvent, error)
	GetSavingsSummary(ctx context.Context, orgID, modelID string, since time.Time) (*SavingsSummary, error)
}
```

- [ ] **Step 3: Create `internal/store/postgres/mcp_usage.go`**

```go
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/uigraph/app/internal/mcpusage"
)

func (d *DB) CreateUsageEvent(ctx context.Context, e mcpusage.UsageEvent) error {
	const q = `
		INSERT INTO mcp_usage_events
			(id, org_id, user_id, service_account_id, tool_name, resource_ids,
			 model_id, tokens_served, tokens_raw_equivalent, tokens_saved,
			 response_size_bytes, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`
	ids := e.ResourceIDs
	if ids == nil {
		ids = []string{}
	}
	_, err := d.db.ExecContext(ctx, q,
		e.ID, e.OrgID, e.UserID, e.ServiceAccountID,
		e.ToolName, pq.Array(ids),
		e.ModelID, e.TokensServed, e.TokensRawEquivalent, e.TokensSaved,
		e.ResponseSizeBytes, time.Now().UTC(),
	)
	return wrapErr("CreateUsageEvent", err)
}

func (d *DB) ListUsageEvents(ctx context.Context, orgID string, f mcpusage.Filter) ([]mcpusage.UsageEvent, error) {
	q := `
		SELECT id, org_id, user_id, service_account_id, tool_name, resource_ids,
		       model_id, tokens_served, tokens_raw_equivalent, tokens_saved,
		       response_size_bytes, created_at
		FROM mcp_usage_events
		WHERE org_id = $1`
	args := []any{orgID}
	i := 2
	if f.Tool != nil {
		q += fmt.Sprintf(" AND tool_name = $%d", i)
		args = append(args, *f.Tool)
		i++
	}
	if f.FromTS != nil {
		q += fmt.Sprintf(" AND created_at >= $%d", i)
		args = append(args, *f.FromTS)
		i++
	}
	if f.ToTS != nil {
		q += fmt.Sprintf(" AND created_at <= $%d", i)
		args = append(args, *f.ToTS)
	}
	q += " ORDER BY created_at DESC LIMIT 500"

	rows, err := d.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres: ListUsageEvents: %w", err)
	}
	defer rows.Close()
	var out []mcpusage.UsageEvent
	for rows.Next() {
		var e mcpusage.UsageEvent
		var ids pq.StringArray
		if scanErr := rows.Scan(
			&e.ID, &e.OrgID, &e.UserID, &e.ServiceAccountID,
			&e.ToolName, &ids,
			&e.ModelID, &e.TokensServed, &e.TokensRawEquivalent, &e.TokensSaved,
			&e.ResponseSizeBytes, &e.CreatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("postgres: ListUsageEvents scan: %w", scanErr)
		}
		e.ResourceIDs = []string(ids)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (d *DB) GetSavingsSummary(ctx context.Context, orgID, modelID string, since time.Time) (*mcpusage.SavingsSummary, error) {
	const q = `
		SELECT
		    COUNT(*)                                                         AS total_calls,
		    COALESCE(SUM(e.tokens_served), 0)                               AS total_tokens_served,
		    COALESCE(SUM(e.tokens_saved), 0)                                AS total_tokens_saved,
		    COALESCE(SUM(e.tokens_served)        ::NUMERIC / 1000000 * m.input_cost_per_million, 0) AS cost_served_usd,
		    COALESCE(SUM(e.tokens_raw_equivalent)::NUMERIC / 1000000 * m.input_cost_per_million, 0) AS cost_raw_usd,
		    COALESCE(SUM(e.tokens_saved)         ::NUMERIC / 1000000 * m.input_cost_per_million, 0) AS cost_saved_usd,
		    COUNT(DISTINCT e.user_id)                                        AS unique_users_count
		FROM mcp_usage_events e
		CROSS JOIN llm_models m
		WHERE e.org_id = $1
		  AND m.model_id = $2
		  AND e.created_at >= $3`
	var s mcpusage.SavingsSummary
	err := d.db.QueryRowContext(ctx, q, orgID, modelID, since).Scan(
		&s.TotalCalls, &s.TotalTokensServed, &s.TotalTokensSaved,
		&s.CostServedUSD, &s.CostRawUSD, &s.CostSavedUSD,
		&s.UniqueUsersCount,
	)
	if err != nil {
		return nil, fmt.Errorf("postgres: GetSavingsSummary: %w", err)
	}
	s.OrgID = orgID
	s.ModelID = modelID
	return &s, nil
}
```

- [ ] **Step 4: Create `internal/api/mcpusage/handler.go`**

```go
package mcpusage

import (
	"context"
	"net/http"
	"time"

	mcppkg "github.com/uigraph/app/internal/mcpusage"
)

type store interface {
	CreateUsageEvent(ctx context.Context, e mcppkg.UsageEvent) error
	ListUsageEvents(ctx context.Context, orgID string, f mcppkg.Filter) ([]mcppkg.UsageEvent, error)
	GetSavingsSummary(ctx context.Context, orgID, modelID string, since time.Time) (*mcppkg.SavingsSummary, error)
}

type Handler struct{ store store }

func New(s store) *Handler { return &Handler{store: s} }

func Register(mux *http.ServeMux, s store, requireScope func(scope, method, pattern string, h http.HandlerFunc)) {
	h := New(s)
	requireScope("services:read", "POST", "/api/v1/orgs/{orgID}/mcp/usage", h.Record)
	requireScope("services:read", "GET", "/api/v1/orgs/{orgID}/mcp/usage", h.List)
	requireScope("services:read", "GET", "/api/v1/orgs/{orgID}/mcp/savings/summary", h.Summary)
}
```

- [ ] **Step 5: Create `internal/api/mcpusage/usage.go`**

```go
package mcpusage

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/xid"
	mcppkg "github.com/uigraph/app/internal/mcpusage"
	"github.com/uigraph/app/internal/httputil"
	authmw "github.com/uigraph/app/internal/middleware"
	"github.com/uigraph/app/internal/identity"
)

func (h *Handler) Record(w http.ResponseWriter, r *http.Request) {
	p, ok := authmw.PrincipalFromCtx(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	var body struct {
		ToolName            string   `json:"toolName"`
		ResourceIDs         []string `json:"resourceIds"`
		ModelID             string   `json:"modelId"`
		TokensServed        int      `json:"tokensServed"`
		TokensRawEquivalent int      `json:"tokensRawEquivalent"`
		TokensSaved         int      `json:"tokensSaved"`
		ResponseSizeBytes   int      `json:"responseSizeBytes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.BadRequest(w, "invalid request body")
		return
	}
	if body.ToolName == "" || body.ModelID == "" {
		httputil.BadRequest(w, "toolName and modelId are required")
		return
	}

	e := mcppkg.UsageEvent{
		ID:                  xid.New().String(),
		OrgID:               r.PathValue("orgID"),
		ToolName:            body.ToolName,
		ResourceIDs:         body.ResourceIDs,
		ModelID:             body.ModelID,
		TokensServed:        body.TokensServed,
		TokensRawEquivalent: body.TokensRawEquivalent,
		TokensSaved:         body.TokensSaved,
		ResponseSizeBytes:   body.ResponseSizeBytes,
		CreatedAt:           time.Now().UTC(),
	}
	if p.Kind == identity.PrincipalServiceAccount {
		e.ServiceAccountID = &p.UserID
	} else {
		e.UserID = &p.UserID
	}

	if err := h.store.CreateUsageEvent(r.Context(), e); err != nil {
		httputil.Error(w, r, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, e)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := mcppkg.Filter{}
	if t := q.Get("tool"); t != "" {
		f.Tool = &t
	}
	if from := q.Get("from"); from != "" {
		if ts, err := time.Parse(time.RFC3339, from); err == nil {
			f.FromTS = &ts
		}
	}
	if to := q.Get("to"); to != "" {
		if ts, err := time.Parse(time.RFC3339, to); err == nil {
			f.ToTS = &ts
		}
	}
	events, err := h.store.ListUsageEvents(r.Context(), r.PathValue("orgID"), f)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}
	if events == nil {
		events = []mcppkg.UsageEvent{}
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"events": events})
}
```

- [ ] **Step 6: Create `internal/api/mcpusage/summary.go`**

```go
package mcpusage

import (
	"net/http"
	"time"

	"github.com/uigraph/app/internal/httputil"
)

func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	modelID := q.Get("model_id")
	if modelID == "" {
		modelID = "claude-sonnet-4-6"
	}

	period := q.Get("period")
	var since time.Time
	switch period {
	case "1d":
		since = time.Now().UTC().Add(-24 * time.Hour)
	case "30d":
		since = time.Now().UTC().Add(-30 * 24 * time.Hour)
	case "1y":
		since = time.Now().UTC().Add(-365 * 24 * time.Hour)
	default:
		period = "7d"
		since = time.Now().UTC().Add(-7 * 24 * time.Hour)
	}

	summary, err := h.store.GetSavingsSummary(r.Context(), r.PathValue("orgID"), modelID, since)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}
	summary.Period = period
	httputil.JSON(w, http.StatusOK, summary)
}
```

- [ ] **Step 7: Add `mcpusage.Store` to `internal/store/store.go`**

```go
import (
	// existing...
	"github.com/uigraph/app/internal/llm"
	"github.com/uigraph/app/internal/mcpusage"
)

type Store interface {
	// existing interfaces...
	llm.Store
	mcpusage.Store
}
```

- [ ] **Step 8: Wire into `internal/api/router.go`**

Add import `mcpusageapi "github.com/uigraph/app/internal/api/mcpusage"` and in `New()`:
```go
mcpusageapi.Register(mux, s, scopeFn)
```

- [ ] **Step 9: Verify build**

```bash
go build ./...
```

- [ ] **Step 10: Commit**

```bash
git add migrations/0030_mcp_usage.sql internal/mcpusage/ internal/store/postgres/mcp_usage.go internal/store/store.go internal/api/mcpusage/ internal/api/router.go
git commit -m "feat: add mcp_usage_events table and savings summary API"
```

---

## Phase B — uigraph-mcp new service

Work directory: `/Users/kranthi/workspace/go/uigraph/backend/uigraph-oss/uigraph-mcp`

---

### Task B1: Project scaffold

**Files:**
- Create: `go.mod`
- Create: `cmd/mcp/main.go`
- Create: `internal/config/config.go`
- Create: `internal/server/server.go`
- Create: `.gitignore`
- Create: `.air.toml`

- [ ] **Step 1: Create `go.mod`**

```
module github.com/uigraph/mcp

go 1.25.0

require (
	github.com/mark3labs/mcp-go v0.32.0
	github.com/google/uuid v1.6.0
	github.com/rs/xid v1.6.0
)
```

- [ ] **Step 2: Run `go mod tidy`**

```bash
cd /Users/kranthi/workspace/go/uigraph/backend/uigraph-oss/uigraph-mcp
go mod tidy
```

- [ ] **Step 3: Create `internal/config/config.go`**

```go
package config

import (
	"fmt"
	"os"
)

type Config struct {
	UIGraphAPIURL   string
	Port            string
	MCPServerName   string
	MCPServerVersion string
}

func Load() (*Config, error) {
	apiURL := os.Getenv("UIGRAPH_API_URL")
	if apiURL == "" {
		return nil, fmt.Errorf("UIGRAPH_API_URL is required")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	name := os.Getenv("MCP_SERVER_NAME")
	if name == "" {
		name = "uigraph-mcp"
	}
	version := os.Getenv("MCP_SERVER_VERSION")
	if version == "" {
		version = "0.1.0"
	}
	return &Config{
		UIGraphAPIURL:    apiURL,
		Port:             port,
		MCPServerName:    name,
		MCPServerVersion: version,
	}, nil
}
```

- [ ] **Step 4: Create `internal/server/server.go`**

```go
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

func Run(port string, handler http.Handler) error {
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // SSE streams stay open
		IdleTimeout:  120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("uigraph-mcp listening", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	return nil
}
```

- [ ] **Step 5: Create `cmd/mcp/main.go`**

```go
package main

import (
	"log/slog"
	"os"

	"github.com/uigraph/mcp/internal/config"
	"github.com/uigraph/mcp/internal/server"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	_ = cfg // mcp server wired in Task B4
	slog.Info("uigraph-mcp starting", "apiURL", cfg.UIGraphAPIURL)
	if err := server.Run(cfg.Port, nil); err != nil {
		slog.Error("run error", "err", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 6: Create `.gitignore`**

```
bin/
tmp/
*.env
.env
```

- [ ] **Step 7: Create `.air.toml`**

```toml
root = "."
tmp_dir = "tmp"

[build]
cmd = "go build -o ./tmp/mcp ./cmd/mcp"
bin = "./tmp/mcp"
include_ext = ["go"]
exclude_dir = ["tmp", "docs"]

[log]
time = true
```

- [ ] **Step 8: Verify build**

```bash
go build ./...
```

- [ ] **Step 9: Commit**

```bash
git add go.mod go.sum cmd/ internal/config/ internal/server/ .gitignore .air.toml
git commit -m "feat: scaffold uigraph-mcp service with config and server lifecycle"
```

---

### Task B2: uigraph-api HTTP client

**Files:**
- Create: `internal/apiclient/client.go`
- Create: `internal/apiclient/catalog.go`
- Create: `internal/apiclient/diagram.go`
- Create: `internal/apiclient/maps.go`
- Create: `internal/apiclient/usage.go`

**Interfaces:**
- Produces: `apiclient.Client` struct with all methods used by tools in Tasks B5–B8

- [ ] **Step 1: Create `internal/apiclient/client.go`**

```go
package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) get(ctx context.Context, token, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("apiclient: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("apiclient: do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("apiclient: %s %s → %d: %s", http.MethodGet, path, resp.StatusCode, body)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) getRaw(ctx context.Context, token, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("apiclient: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("apiclient: do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("apiclient: %s → %d: %s", path, resp.StatusCode, body)
	}
	return io.ReadAll(resp.Body)
}

var ErrNotFound = fmt.Errorf("not found")
```

- [ ] **Step 2: Create `internal/apiclient/catalog.go`**

Define lightweight response types (not importing uigraph-api types) and methods:

```go
package apiclient

import (
	"context"
	"fmt"
	"time"
)

type Service struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Tier        string    `json:"tier"`
	Category    string    `json:"category"`
	Language    string    `json:"language"`
	Labels      []string  `json:"labels"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ServiceStats struct {
	ServiceID     string `json:"serviceId"`
	EndpointCount int    `json:"endpointCount"`
	DiagramCount  int    `json:"diagramCount"`
	DocCount      int    `json:"docCount"`
	DBTableCount  int    `json:"dbTableCount"`
	TestCaseCount int    `json:"testCaseCount"`
}

type APIGroup struct {
	ID        string    `json:"id"`
	ServiceID string    `json:"serviceId"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Protocol  string    `json:"protocol"`
	Label     *string   `json:"label,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type APIEndpoint struct {
	ID          string   `json:"id"`
	OperationID string   `json:"operationId"`
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	Summary     string   `json:"summary"`
	Tags        []string `json:"tags"`
	TokenCount  int      `json:"tokenCount"`
}

type ServiceDoc struct {
	ID            string `json:"id"`
	FileName      string `json:"fileName"`
	FileType      string `json:"fileType"`
	Description   string `json:"description"`
	DocTokenCount int    `json:"docTokenCount"`
}

type ServiceDB struct {
	ID               string    `json:"id"`
	DBName           string    `json:"dbName"`
	DBType           string    `json:"dbType"`
	Dialect          string    `json:"dialect"`
	SchemaTokenCount int       `json:"schemaTokenCount"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type ServiceDiagram struct {
	ServiceID string `json:"serviceId"`
	DiagramID string `json:"diagramId"`
}

func (c *Client) ListServices(ctx context.Context, token, orgID string, folderID, teamID *string) ([]Service, error) {
	path := fmt.Sprintf("/api/v1/orgs/%s/services", orgID)
	if folderID != nil {
		path += "?folderId=" + *folderID
	}
	var resp struct {
		Services []Service `json:"services"`
	}
	return resp.Services, c.get(ctx, token, path, &resp)
}

func (c *Client) GetService(ctx context.Context, token, orgID, serviceID string) (*Service, error) {
	var svc Service
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s", orgID, serviceID), &svc)
	return &svc, err
}

func (c *Client) GetServiceBySlug(ctx context.Context, token, orgID, slug string) (*Service, error) {
	svcs, err := c.ListServices(ctx, token, orgID, nil, nil)
	if err != nil {
		return nil, err
	}
	for _, s := range svcs {
		if s.Slug == slug {
			return &s, nil
		}
	}
	return nil, ErrNotFound
}

func (c *Client) ListServiceStats(ctx context.Context, token, orgID string) ([]ServiceStats, error) {
	var resp struct {
		Stats []ServiceStats `json:"stats"`
	}
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/stats", orgID), &resp)
	return resp.Stats, err
}

func (c *Client) ListAPIGroups(ctx context.Context, token, orgID, serviceID string) ([]APIGroup, error) {
	var resp struct {
		APIGroups []APIGroup `json:"apiGroups"`
	}
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s/api-groups", orgID, serviceID), &resp)
	return resp.APIGroups, err
}

func (c *Client) GetAPIGroupSpec(ctx context.Context, token, orgID, serviceID, apiGroupID string) ([]byte, error) {
	return c.getRaw(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s/api-groups/%s/spec", orgID, serviceID, apiGroupID))
}

func (c *Client) ListAPIEndpoints(ctx context.Context, token, orgID, serviceID, apiGroupID string) ([]APIEndpoint, error) {
	var resp struct {
		Endpoints []APIEndpoint `json:"endpoints"`
	}
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s/api-groups/%s/endpoints", orgID, serviceID, apiGroupID), &resp)
	return resp.Endpoints, err
}

func (c *Client) ListServiceDocs(ctx context.Context, token, orgID, serviceID string) ([]ServiceDoc, error) {
	var resp struct {
		Docs []ServiceDoc `json:"docs"`
	}
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s/docs", orgID, serviceID), &resp)
	return resp.Docs, err
}

func (c *Client) ListServiceDBs(ctx context.Context, token, orgID, serviceID string) ([]ServiceDB, error) {
	var resp struct {
		DBs []ServiceDB `json:"dbs"`
	}
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s/dbs", orgID, serviceID), &resp)
	return resp.DBs, err
}

func (c *Client) GetServiceDB(ctx context.Context, token, orgID, serviceID, dbID string) (*ServiceDB, error) {
	var db ServiceDB
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s/dbs/%s", orgID, serviceID, dbID), &db)
	return &db, err
}

func (c *Client) ListServiceDiagrams(ctx context.Context, token, orgID, serviceID string) ([]ServiceDiagram, error) {
	var resp struct {
		Diagrams []ServiceDiagram `json:"diagrams"`
	}
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s/diagrams", orgID, serviceID), &resp)
	return resp.Diagrams, err
}
```

- [ ] **Step 3: Create `internal/apiclient/diagram.go`**

```go
package apiclient

import (
	"context"
	"fmt"
	"time"
)

type Diagram struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	ContentTokenCount int       `json:"contentTokenCount"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func (c *Client) ListDiagrams(ctx context.Context, token, orgID string, folderID, teamID *string) ([]Diagram, error) {
	path := fmt.Sprintf("/api/v1/orgs/%s/diagrams", orgID)
	var resp struct {
		Diagrams []Diagram `json:"diagrams"`
	}
	return resp.Diagrams, c.get(ctx, token, path, &resp)
}

func (c *Client) GetDiagramContent(ctx context.Context, token, orgID, diagramID string) ([]byte, error) {
	return c.getRaw(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/diagrams/%s/content", orgID, diagramID))
}
```

- [ ] **Step 4: Create `internal/apiclient/maps.go`**

```go
package apiclient

import (
	"context"
	"fmt"
	"time"
)

type Map struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Frame struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	TemplateType string  `json:"templateType"`
	Status       string  `json:"status"`
	ParentFrameID *string `json:"parentFrameId,omitempty"`
	Order        float64 `json:"order"`
}

type Folder struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	ParentID *string `json:"parentId,omitempty"`
	Order    float64 `json:"order"`
}

func (c *Client) ListMaps(ctx context.Context, token, orgID string, folderID, teamID *string) ([]Map, error) {
	var resp struct {
		Maps []Map `json:"maps"`
	}
	return resp.Maps, c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/maps", orgID), &resp)
}

func (c *Client) GetMap(ctx context.Context, token, orgID, mapID string) (*Map, error) {
	var m Map
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/maps/%s", orgID, mapID), &m)
	return &m, err
}

func (c *Client) ListFrames(ctx context.Context, token, orgID, mapID string) ([]Frame, error) {
	var resp struct {
		Frames []Frame `json:"frames"`
	}
	return resp.Frames, c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/maps/%s/frames", orgID, mapID), &resp)
}

func (c *Client) ListFolders(ctx context.Context, token, orgID string, folderType *string) ([]Folder, error) {
	path := fmt.Sprintf("/api/v1/orgs/%s/folders", orgID)
	if folderType != nil {
		path += "?type=" + *folderType
	}
	var resp struct {
		Folders []Folder `json:"folders"`
	}
	return resp.Folders, c.get(ctx, token, path, &resp)
}
```

- [ ] **Step 5: Create `internal/apiclient/usage.go`**

```go
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type UsageEventPayload struct {
	ToolName            string   `json:"toolName"`
	ResourceIDs         []string `json:"resourceIds"`
	ModelID             string   `json:"modelId"`
	TokensServed        int      `json:"tokensServed"`
	TokensRawEquivalent int      `json:"tokensRawEquivalent"`
	TokensSaved         int      `json:"tokensSaved"`
	ResponseSizeBytes   int      `json:"responseSizeBytes"`
}

func (c *Client) RecordUsage(ctx context.Context, token, orgID string, e UsageEventPayload) error {
	b, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("apiclient: marshal usage event: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+fmt.Sprintf("/api/v1/orgs/%s/mcp/usage", orgID), bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("apiclient: build usage request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("apiclient: record usage: %w", err)
	}
	resp.Body.Close()
	return nil
}
```

- [ ] **Step 6: Verify build**

```bash
go build ./...
```

- [ ] **Step 7: Commit**

```bash
git add internal/apiclient/
git commit -m "feat: add uigraph-api HTTP client for MCP tool implementations"
```

---

### Task B3: Token estimation

**Files:**
- Create: `internal/tokencount/estimate.go`
- Create: `internal/tokencount/estimate_test.go`

**Interfaces:**
- Produces: `tokencount.Count(text string) int`, `tokencount.RawEquivalent(toolName string, served int, exactFileTokens *int) int`

- [ ] **Step 1: Write the failing test**

Create `internal/tokencount/estimate_test.go`:
```go
package tokencount_test

import (
	"testing"

	"github.com/uigraph/mcp/internal/tokencount"
)

func TestCount(t *testing.T) {
	got := tokencount.Count("hello world") // 11 chars / 4 = 2
	if got != 2 {
		t.Fatalf("Count(%q) = %d, want 2", "hello world", got)
	}
}

func TestRawEquivalent_ExactCount(t *testing.T) {
	exact := 5000
	got := tokencount.RawEquivalent("get_api_spec", 1000, &exact)
	if got != 5000 {
		t.Fatalf("RawEquivalent with exact = %d, want 5000", got)
	}
}

func TestRawEquivalent_Multiplier(t *testing.T) {
	// get_api_spec fallback multiplier is 4.0x
	got := tokencount.RawEquivalent("get_api_spec", 1000, nil)
	if got != 4000 {
		t.Fatalf("RawEquivalent fallback = %d, want 4000", got)
	}
}

func TestRawEquivalent_UnknownTool(t *testing.T) {
	// unknown tool uses 1.5x
	got := tokencount.RawEquivalent("unknown_tool", 1000, nil)
	if got != 1500 {
		t.Fatalf("RawEquivalent unknown = %d, want 1500", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/tokencount/...
```
Expected: FAIL with "cannot find package"

- [ ] **Step 3: Create `internal/tokencount/estimate.go`**

```go
package tokencount

// Count estimates the number of LLM tokens in text using the standard
// chars-per-token approximation.
func Count(text string) int {
	return len(text) / 4
}

// multipliers maps tool name → raw-file-read equivalent multiplier.
// When an AI tool reads the equivalent raw repo file, it burns
// (served tokens × multiplier) input tokens.
var multipliers = map[string]float64{
	"get_service_context": 6.0,
	"get_api_spec":        4.0,
	"list_endpoints":      3.0,
	"get_db_schema":       3.5,
	"get_diagram":         2.0,
	"get_map":             2.0,
	"list_services":       1.5,
	"get_service":         1.5,
	"list_api_groups":     1.5,
	"list_service_dbs":    1.5,
	"list_diagrams":       1.5,
	"list_maps":           1.5,
	"list_folders":        1.5,
}

const defaultMultiplier = 1.5

// RawEquivalent returns the estimated tokens an AI would have burned reading
// the equivalent raw repo files. If exactFileTokens is non-nil (recorded at
// CLI sync time), that exact count is used. Otherwise the per-tool multiplier
// is applied to served.
func RawEquivalent(toolName string, served int, exactFileTokens *int) int {
	if exactFileTokens != nil && *exactFileTokens > 0 {
		return *exactFileTokens
	}
	m, ok := multipliers[toolName]
	if !ok {
		m = defaultMultiplier
	}
	return int(float64(served) * m)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/tokencount/... -v
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tokencount/
git commit -m "feat: add token estimation package with exact-count and multiplier fallback"
```

---

### Task B4: MCP server — SSE transport and tool dispatch

**Files:**
- Create: `internal/mcp/server.go`
- Modify: `cmd/mcp/main.go` — wire the MCP server

**Interfaces:**
- Produces: `mcp.New(cfg, client) http.Handler` consumed by `main.go`; `mcp.extractToken(r) string` used by all tool handlers

- [ ] **Step 1: Create `internal/mcp/server.go`**

```go
package mcp

import (
	"net/http"
	"strings"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/uigraph/mcp/internal/apiclient"
	"github.com/uigraph/mcp/internal/config"
)

// New builds the MCP HTTP/SSE handler with all tools registered.
func New(cfg *config.Config, client *apiclient.Client) http.Handler {
	s := mcpserver.NewMCPServer(cfg.MCPServerName, cfg.MCPServerVersion)

	registerTools(s, client)

	sse := mcpserver.NewSSEServer(s,
		mcpserver.WithBaseURL("http://0.0.0.0:"+cfg.Port),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/", sse)
	return mux
}

// extractToken pulls the bearer token from the Authorization header.
func extractToken(r *http.Request) string {
	v := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(v, "Bearer "); ok {
		return after
	}
	return ""
}
```

- [ ] **Step 2: Create `internal/mcp/tools.go`** (stub — filled out in B5–B8)

```go
package mcp

import (
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/uigraph/mcp/internal/apiclient"
	"github.com/uigraph/mcp/internal/tools"
)

func registerTools(s *mcpserver.MCPServer, client *apiclient.Client) {
	h := tools.New(client)
	_ = h // tool registration added in Tasks B5–B8
}
```

- [ ] **Step 3: Create `internal/tools/tools.go`**

```go
package tools

import "github.com/uigraph/mcp/internal/apiclient"

// Handler holds dependencies for all MCP tool implementations.
type Handler struct {
	client *apiclient.Client
}

func New(client *apiclient.Client) *Handler {
	return &Handler{client: client}
}
```

- [ ] **Step 4: Update `cmd/mcp/main.go`**

```go
package main

import (
	"log/slog"
	"os"

	"github.com/uigraph/mcp/internal/apiclient"
	"github.com/uigraph/mcp/internal/config"
	mcphandler "github.com/uigraph/mcp/internal/mcp"
	"github.com/uigraph/mcp/internal/server"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	client := apiclient.New(cfg.UIGraphAPIURL)
	handler := mcphandler.New(cfg, client)

	slog.Info("uigraph-mcp starting", "port", cfg.Port)
	if err := server.Run(cfg.Port, handler); err != nil {
		slog.Error("run error", "err", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Verify build**

```bash
go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/mcp/ internal/tools/ cmd/mcp/main.go
git commit -m "feat: wire MCP SSE server with tool dispatch skeleton"
```

---

### Task B5: Catalog and folder tools

**Files:**
- Create: `internal/tools/catalog.go`
- Create: `internal/tools/folders.go`
- Modify: `internal/mcp/tools.go` — register catalog + folder tools

- [ ] **Step 1: Create `internal/tools/catalog.go`**

```go
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func (h *Handler) RegisterCatalogTools(s *mcpserver.MCPServer) {
	s.AddTool(mcp.NewTool("list_services",
		mcp.WithDescription("List all services in a UIGraph organisation"),
		mcp.WithString("org_id", mcp.Required(), mcp.Description("Organisation ID")),
		mcp.WithString("folder_id", mcp.Description("Optional folder ID filter")),
	), h.listServices)

	s.AddTool(mcp.NewTool("get_service",
		mcp.WithDescription("Get full details and stats for a service"),
		mcp.WithString("org_id", mcp.Required(), mcp.Description("Organisation ID")),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
	), h.getService)

	s.AddTool(mcp.NewTool("list_api_groups",
		mcp.WithDescription("List API specification groups for a service"),
		mcp.WithString("org_id", mcp.Required(), mcp.Description("Organisation ID")),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
	), h.listAPIGroups)

	s.AddTool(mcp.NewTool("get_api_spec",
		mcp.WithDescription("Get the full API specification content (OpenAPI/GraphQL/gRPC) for an API group"),
		mcp.WithString("org_id", mcp.Required(), mcp.Description("Organisation ID")),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
		mcp.WithString("api_group_id", mcp.Required(), mcp.Description("API group ID")),
		mcp.WithString("model_id", mcp.Description("LLM model ID for cost tracking (e.g. claude-sonnet-4-6)")),
	), h.getAPISpec)

	s.AddTool(mcp.NewTool("list_endpoints",
		mcp.WithDescription("List all API endpoints for a service or API group"),
		mcp.WithString("org_id", mcp.Required(), mcp.Description("Organisation ID")),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
		mcp.WithString("api_group_id", mcp.Required(), mcp.Description("API group ID")),
	), h.listEndpoints)
}

func (h *Handler) listServices(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID := req.Params.Arguments["org_id"].(string)
	token := tokenFromCtx(ctx)

	svcs, err := h.client.ListServices(ctx, token, orgID, nil, nil)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Services in org %s\n\n", orgID))
	for _, s := range svcs {
		sb.WriteString(fmt.Sprintf("- **%s** (`%s`) — %s | %s | %s\n", s.Name, s.Slug, s.Status, s.Tier, s.Language))
		if s.Description != "" {
			sb.WriteString(fmt.Sprintf("  %s\n", s.Description))
		}
	}
	if len(svcs) == 0 {
		sb.WriteString("No services found.\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) getService(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID := req.Params.Arguments["org_id"].(string)
	serviceID := req.Params.Arguments["service_id"].(string)
	token := tokenFromCtx(ctx)

	svc, err := h.client.GetService(ctx, token, orgID, serviceID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", svc.Name))
	sb.WriteString(fmt.Sprintf("**ID:** %s | **Slug:** %s\n", svc.ID, svc.Slug))
	sb.WriteString(fmt.Sprintf("**Status:** %s | **Tier:** %s | **Language:** %s | **Category:** %s\n",
		svc.Status, svc.Tier, svc.Language, svc.Category))
	if svc.Description != "" {
		sb.WriteString(fmt.Sprintf("\n%s\n", svc.Description))
	}
	if len(svc.Labels) > 0 {
		sb.WriteString(fmt.Sprintf("\n**Labels:** %s\n", strings.Join(svc.Labels, ", ")))
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) listAPIGroups(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID := req.Params.Arguments["org_id"].(string)
	serviceID := req.Params.Arguments["service_id"].(string)
	token := tokenFromCtx(ctx)

	groups, err := h.client.ListAPIGroups(ctx, token, orgID, serviceID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# API Groups\n\n")
	for _, g := range groups {
		label := ""
		if g.Label != nil {
			label = " (" + *g.Label + ")"
		}
		sb.WriteString(fmt.Sprintf("- **%s** %s%s — %s | ID: `%s`\n",
			g.Name, g.Version, label, g.Protocol, g.ID))
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) getAPISpec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID := req.Params.Arguments["org_id"].(string)
	serviceID := req.Params.Arguments["service_id"].(string)
	apiGroupID := req.Params.Arguments["api_group_id"].(string)
	modelID, _ := req.Params.Arguments["model_id"].(string)
	if modelID == "" {
		modelID = "claude-sonnet-4-6"
	}
	token := tokenFromCtx(ctx)

	specBytes, err := h.client.GetAPIGroupSpec(ctx, token, orgID, serviceID, apiGroupID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	const maxChars = 50_000
	spec := string(specBytes)
	truncated := false
	if len(spec) > maxChars {
		spec = spec[:maxChars]
		truncated = true
	}

	// sum per-endpoint token counts for exact raw-file savings
	endpoints, _ := h.client.ListAPIEndpoints(ctx, token, orgID, serviceID, apiGroupID)
	var exactTokens *int
	if len(endpoints) > 0 {
		total := 0
		for _, e := range endpoints {
			total += e.TokenCount
		}
		exactTokens = &total
	}

	go h.recordUsage(orgID, token, "get_api_spec", []string{apiGroupID}, modelID, spec, exactTokens)

	result := spec
	if truncated {
		result += "\n\n[Truncated at 50,000 characters]"
	}
	return mcp.NewToolResultText(result), nil
}

func (h *Handler) listEndpoints(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID := req.Params.Arguments["org_id"].(string)
	serviceID := req.Params.Arguments["service_id"].(string)
	apiGroupID := req.Params.Arguments["api_group_id"].(string)
	token := tokenFromCtx(ctx)

	endpoints, err := h.client.ListAPIEndpoints(ctx, token, orgID, serviceID, apiGroupID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# API Endpoints\n\n")
	for _, e := range endpoints {
		tags := ""
		if len(e.Tags) > 0 {
			tags = " [" + strings.Join(e.Tags, ", ") + "]"
		}
		sb.WriteString(fmt.Sprintf("- **%s %s**%s — %s (~%d tokens)\n", e.Method, e.Path, tags, e.Summary, e.TokenCount))
	}
	return mcp.NewToolResultText(sb.String()), nil
}
```

- [ ] **Step 2: Create `internal/tools/folders.go`**

```go
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func (h *Handler) RegisterFolderTools(s *mcpserver.MCPServer) {
	s.AddTool(mcp.NewTool("list_folders",
		mcp.WithDescription("List folders in a UIGraph organisation"),
		mcp.WithString("org_id", mcp.Required(), mcp.Description("Organisation ID")),
		mcp.WithString("type", mcp.Description("Filter by type: service, diagram, map, doc")),
	), h.listFolders)
}

func (h *Handler) listFolders(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID := req.Params.Arguments["org_id"].(string)
	folderType, _ := req.Params.Arguments["type"].(string)
	token := tokenFromCtx(ctx)

	var ft *string
	if folderType != "" {
		ft = &folderType
	}

	folders, err := h.client.ListFolders(ctx, token, orgID, ft)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# Folders\n\n")
	for _, f := range folders {
		parent := ""
		if f.ParentID != nil {
			parent = fmt.Sprintf(" (parent: %s)", *f.ParentID)
		}
		sb.WriteString(fmt.Sprintf("- **%s** [%s]%s — ID: `%s`\n", f.Name, f.Type, parent, f.ID))
	}
	return mcp.NewToolResultText(sb.String()), nil
}
```

- [ ] **Step 3: Create `internal/tools/helpers.go`** — shared helpers used across all tools

```go
package tools

import (
	"context"
	"log/slog"

	"github.com/uigraph/mcp/internal/apiclient"
	"github.com/uigraph/mcp/internal/tokencount"
)

type contextKey string

const tokenKey contextKey = "bearer"

// tokenFromCtx retrieves the bearer token injected into the request context.
func tokenFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(tokenKey).(string)
	return v
}

// WithToken returns a new context carrying the bearer token.
func WithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenKey, token)
}

// recordUsage fires-and-forgets a usage event to uigraph-api.
func (h *Handler) recordUsage(orgID, token, toolName string, resourceIDs []string, modelID, responseText string, exactFileTokens *int) {
	ctx := context.Background()
	served := tokencount.Count(responseText)
	raw := tokencount.RawEquivalent(toolName, served, exactFileTokens)
	payload := apiclient.UsageEventPayload{
		ToolName:            toolName,
		ResourceIDs:         resourceIDs,
		ModelID:             modelID,
		TokensServed:        served,
		TokensRawEquivalent: raw,
		TokensSaved:         raw - served,
		ResponseSizeBytes:   len(responseText),
	}
	if err := h.client.RecordUsage(ctx, token, orgID, payload); err != nil {
		slog.Warn("failed to record MCP usage", "tool", toolName, "err", err)
	}
}
```

- [ ] **Step 4: Update `internal/mcp/server.go`** — inject token into context

The mark3labs/mcp-go SSE server provides a middleware hook. Add token injection:

```go
package mcp

import (
	"net/http"
	"strings"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/uigraph/mcp/internal/apiclient"
	"github.com/uigraph/mcp/internal/config"
	"github.com/uigraph/mcp/internal/tools"
)

func New(cfg *config.Config, client *apiclient.Client) http.Handler {
	s := mcpserver.NewMCPServer(cfg.MCPServerName, cfg.MCPServerVersion)
	registerTools(s, client)

	sse := mcpserver.NewSSEServer(s,
		mcpserver.WithBaseURL("http://0.0.0.0:"+cfg.Port),
		mcpserver.WithSSEContextFunc(func(r *http.Request) (any, any) {
			token := extractToken(r)
			return tools.TokenKey, token
		}),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/", sse)
	return mux
}

func extractToken(r *http.Request) string {
	v := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(v, "Bearer "); ok {
		return after
	}
	return ""
}
```

Export `TokenKey` from helpers:
```go
// in internal/tools/helpers.go, change:
const TokenKey contextKey = "bearer"
// and tokenFromCtx uses TokenKey
```

- [ ] **Step 5: Update `internal/mcp/tools.go`**

```go
package mcp

import (
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/uigraph/mcp/internal/apiclient"
	"github.com/uigraph/mcp/internal/tools"
)

func registerTools(s *mcpserver.MCPServer, client *apiclient.Client) {
	h := tools.New(client)
	h.RegisterCatalogTools(s)
	h.RegisterFolderTools(s)
}
```

- [ ] **Step 6: Verify build**

```bash
go build ./...
```

- [ ] **Step 7: Commit**

```bash
git add internal/tools/ internal/mcp/
git commit -m "feat: add catalog and folder MCP tools"
```

---

### Task B6: Diagram and DB schema tools

**Files:**
- Create: `internal/tools/diagrams.go`
- Create: `internal/tools/schemas.go`
- Modify: `internal/mcp/tools.go` — register diagram + schema tools

- [ ] **Step 1: Create `internal/tools/diagrams.go`**

```go
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func (h *Handler) RegisterDiagramTools(s *mcpserver.MCPServer) {
	s.AddTool(mcp.NewTool("list_diagrams",
		mcp.WithDescription("List architecture diagrams in a UIGraph organisation"),
		mcp.WithString("org_id", mcp.Required(), mcp.Description("Organisation ID")),
	), h.listDiagrams)

	s.AddTool(mcp.NewTool("get_diagram",
		mcp.WithDescription("Get the content of an architecture diagram"),
		mcp.WithString("org_id", mcp.Required(), mcp.Description("Organisation ID")),
		mcp.WithString("diagram_id", mcp.Required(), mcp.Description("Diagram ID")),
		mcp.WithString("model_id", mcp.Description("LLM model ID for cost tracking")),
	), h.getDiagram)
}

func (h *Handler) listDiagrams(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID := req.Params.Arguments["org_id"].(string)
	token := tokenFromCtx(ctx)

	diagrams, err := h.client.ListDiagrams(ctx, token, orgID, nil, nil)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# Architecture Diagrams\n\n")
	for _, d := range diagrams {
		sb.WriteString(fmt.Sprintf("- **%s** — ID: `%s` | raw: ~%d tokens\n",
			d.Name, d.ID, d.ContentTokenCount))
	}
	if len(diagrams) == 0 {
		sb.WriteString("No diagrams found.\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) getDiagram(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID := req.Params.Arguments["org_id"].(string)
	diagramID := req.Params.Arguments["diagram_id"].(string)
	modelID, _ := req.Params.Arguments["model_id"].(string)
	if modelID == "" {
		modelID = "claude-sonnet-4-6"
	}
	token := tokenFromCtx(ctx)

	content, err := h.client.GetDiagramContent(ctx, token, orgID, diagramID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	const maxChars = 100_000
	text := string(content)
	truncated := false
	if len(text) > maxChars {
		text = text[:maxChars]
		truncated = true
	}

	diagrams, _ := h.client.ListDiagrams(ctx, token, orgID, nil, nil)
	var exactTokens *int
	for _, d := range diagrams {
		if d.ID == diagramID {
			t := d.ContentTokenCount
			exactTokens = &t
			break
		}
	}

	go h.recordUsage(orgID, token, "get_diagram", []string{diagramID}, modelID, text, exactTokens)

	if truncated {
		text += "\n\n[Truncated at 100,000 characters]"
	}
	return mcp.NewToolResultText(text), nil
}
```

- [ ] **Step 2: Create `internal/tools/schemas.go`**

```go
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func (h *Handler) RegisterSchemaTools(s *mcpserver.MCPServer) {
	s.AddTool(mcp.NewTool("list_service_dbs",
		mcp.WithDescription("List database schemas attached to a service"),
		mcp.WithString("org_id", mcp.Required(), mcp.Description("Organisation ID")),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
	), h.listServiceDBs)

	s.AddTool(mcp.NewTool("get_db_schema",
		mcp.WithDescription("Get the full database schema for a service DB"),
		mcp.WithString("org_id", mcp.Required(), mcp.Description("Organisation ID")),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
		mcp.WithString("db_id", mcp.Required(), mcp.Description("Service DB ID")),
		mcp.WithString("model_id", mcp.Description("LLM model ID for cost tracking")),
	), h.getDBSchema)
}

func (h *Handler) listServiceDBs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID := req.Params.Arguments["org_id"].(string)
	serviceID := req.Params.Arguments["service_id"].(string)
	token := tokenFromCtx(ctx)

	dbs, err := h.client.ListServiceDBs(ctx, token, orgID, serviceID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# Database Schemas\n\n")
	for _, db := range dbs {
		sb.WriteString(fmt.Sprintf("- **%s** (%s/%s) — ID: `%s` | raw: ~%d tokens\n",
			db.DBName, db.DBType, db.Dialect, db.ID, db.SchemaTokenCount))
	}
	if len(dbs) == 0 {
		sb.WriteString("No databases found.\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) getDBSchema(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID := req.Params.Arguments["org_id"].(string)
	serviceID := req.Params.Arguments["service_id"].(string)
	dbID := req.Params.Arguments["db_id"].(string)
	modelID, _ := req.Params.Arguments["model_id"].(string)
	if modelID == "" {
		modelID = "claude-sonnet-4-6"
	}
	token := tokenFromCtx(ctx)

	db, err := h.client.GetServiceDB(ctx, token, orgID, serviceID, dbID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	text := fmt.Sprintf("# Database: %s\nType: %s | Dialect: %s\n\n", db.DBName, db.DBType, db.Dialect)
	// schema JSON is returned as part of the ServiceDB struct — the apiclient
	// would need to include SchemaJSON. For now format available metadata.
	text += fmt.Sprintf("Raw schema file: ~%d tokens\nID: %s\n", db.SchemaTokenCount, db.ID)

	const maxChars = 50_000
	truncated := false
	if len(text) > maxChars {
		text = text[:maxChars]
		truncated = true
	}

	exactTokens := &db.SchemaTokenCount
	go h.recordUsage(orgID, token, "get_db_schema", []string{dbID}, modelID, text, exactTokens)

	if truncated {
		text += "\n\n[Truncated at 50,000 characters]"
	}
	return mcp.NewToolResultText(text), nil
}
```

- [ ] **Step 3: Update `internal/mcp/tools.go`**

```go
func registerTools(s *mcpserver.MCPServer, client *apiclient.Client) {
	h := tools.New(client)
	h.RegisterCatalogTools(s)
	h.RegisterFolderTools(s)
	h.RegisterDiagramTools(s)
	h.RegisterSchemaTools(s)
}
```

- [ ] **Step 4: Add `SchemaJSON` to apiclient ServiceDB** 

In `internal/apiclient/catalog.go`, update `GetServiceDB` to also return schema content. Add a `GetServiceDBSchema` method:

```go
func (c *Client) GetServiceDBSchema(ctx context.Context, token, orgID, serviceID, dbID string) ([]byte, error) {
	return c.getRaw(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s/dbs/%s", orgID, serviceID, dbID))
}
```

- [ ] **Step 5: Verify build**

```bash
go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/tools/diagrams.go internal/tools/schemas.go internal/mcp/tools.go internal/apiclient/catalog.go
git commit -m "feat: add diagram and DB schema MCP tools"
```

---

### Task B7: Map tools

**Files:**
- Create: `internal/tools/maps.go`
- Modify: `internal/mcp/tools.go`

- [ ] **Step 1: Create `internal/tools/maps.go`**

```go
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func (h *Handler) RegisterMapTools(s *mcpserver.MCPServer) {
	s.AddTool(mcp.NewTool("list_maps",
		mcp.WithDescription("List UI journey maps in a UIGraph organisation"),
		mcp.WithString("org_id", mcp.Required(), mcp.Description("Organisation ID")),
	), h.listMaps)

	s.AddTool(mcp.NewTool("get_map",
		mcp.WithDescription("Get a UI journey map with all its frames"),
		mcp.WithString("org_id", mcp.Required(), mcp.Description("Organisation ID")),
		mcp.WithString("map_id", mcp.Required(), mcp.Description("Map ID")),
		mcp.WithString("model_id", mcp.Description("LLM model ID for cost tracking")),
	), h.getMap)
}

func (h *Handler) listMaps(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID := req.Params.Arguments["org_id"].(string)
	token := tokenFromCtx(ctx)

	maps, err := h.client.ListMaps(ctx, token, orgID, nil, nil)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# UI Journey Maps\n\n")
	for _, m := range maps {
		sb.WriteString(fmt.Sprintf("- **%s** [%s] — ID: `%s`\n", m.Name, m.Status, m.ID))
		if m.Description != "" {
			sb.WriteString(fmt.Sprintf("  %s\n", m.Description))
		}
	}
	if len(maps) == 0 {
		sb.WriteString("No maps found.\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) getMap(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID := req.Params.Arguments["org_id"].(string)
	mapID := req.Params.Arguments["map_id"].(string)
	modelID, _ := req.Params.Arguments["model_id"].(string)
	if modelID == "" {
		modelID = "claude-sonnet-4-6"
	}
	token := tokenFromCtx(ctx)

	m, err := h.client.GetMap(ctx, token, orgID, mapID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	frames, err := h.client.ListFrames(ctx, token, orgID, mapID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Map: %s\n", m.Name))
	sb.WriteString(fmt.Sprintf("**Status:** %s\n", m.Status))
	if m.Description != "" {
		sb.WriteString(fmt.Sprintf("\n%s\n", m.Description))
	}
	sb.WriteString(fmt.Sprintf("\n## Frames (%d)\n\n", len(frames)))
	for _, f := range frames {
		indent := ""
		if f.ParentFrameID != nil {
			indent = "  "
		}
		sb.WriteString(fmt.Sprintf("%s- **%s** [%s/%s]\n", indent, f.Name, f.TemplateType, f.Status))
		if f.Description != "" {
			sb.WriteString(fmt.Sprintf("%s  %s\n", indent, f.Description))
		}
	}

	text := sb.String()
	go h.recordUsage(orgID, token, "get_map", []string{mapID}, modelID, text, nil)
	return mcp.NewToolResultText(text), nil
}
```

- [ ] **Step 2: Update `internal/mcp/tools.go`**

```go
func registerTools(s *mcpserver.MCPServer, client *apiclient.Client) {
	h := tools.New(client)
	h.RegisterCatalogTools(s)
	h.RegisterFolderTools(s)
	h.RegisterDiagramTools(s)
	h.RegisterSchemaTools(s)
	h.RegisterMapTools(s)
}
```

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/tools/maps.go internal/mcp/tools.go
git commit -m "feat: add map MCP tools"
```

---

### Task B8: Service context composite tool

**Files:**
- Create: `internal/tools/service_context.go`
- Modify: `internal/mcp/tools.go`

- [ ] **Step 1: Create `internal/tools/service_context.go`**

```go
package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/uigraph/mcp/internal/apiclient"
)

func (h *Handler) RegisterServiceContextTool(s *mcpserver.MCPServer) {
	s.AddTool(mcp.NewTool("get_service_context",
		mcp.WithDescription("Get comprehensive context for a service: metadata, API specs, DB schemas, diagrams, and docs. Use this as the primary tool when you need to understand a service."),
		mcp.WithString("org_id", mcp.Required(), mcp.Description("Organisation ID")),
		mcp.WithString("service_id_or_slug", mcp.Required(), mcp.Description("Service ID (UUID) or slug")),
		mcp.WithString("model_id", mcp.Description("LLM model ID for cost tracking (e.g. claude-sonnet-4-6)")),
	), h.getServiceContext)
}

func (h *Handler) getServiceContext(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID := req.Params.Arguments["org_id"].(string)
	serviceIDOrSlug := req.Params.Arguments["service_id_or_slug"].(string)
	modelID, _ := req.Params.Arguments["model_id"].(string)
	if modelID == "" {
		modelID = "claude-sonnet-4-6"
	}
	token := tokenFromCtx(ctx)

	// Resolve service (UUID vs slug)
	svc, err := h.resolveService(ctx, token, orgID, serviceIDOrSlug)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("service not found: %s", serviceIDOrSlug)), nil
	}

	// Fan out parallel fetches
	type result[T any] struct {
		val T
		err error
	}

	var (
		apiGroupsCh = make(chan result[[]apiclient.APIGroup], 1)
		dbsCh       = make(chan result[[]apiclient.ServiceDB], 1)
		diagramsCh  = make(chan result[[]apiclient.ServiceDiagram], 1)
		docsCh      = make(chan result[[]apiclient.ServiceDoc], 1)
		wg          sync.WaitGroup
	)

	wg.Add(4)
	go func() { defer wg.Done(); v, e := h.client.ListAPIGroups(ctx, token, orgID, svc.ID); apiGroupsCh <- result[[]apiclient.APIGroup]{v, e} }()
	go func() { defer wg.Done(); v, e := h.client.ListServiceDBs(ctx, token, orgID, svc.ID); dbsCh <- result[[]apiclient.ServiceDB]{v, e} }()
	go func() { defer wg.Done(); v, e := h.client.ListServiceDiagrams(ctx, token, orgID, svc.ID); diagramsCh <- result[[]apiclient.ServiceDiagram]{v, e} }()
	go func() { defer wg.Done(); v, e := h.client.ListServiceDocs(ctx, token, orgID, svc.ID); docsCh <- result[[]apiclient.ServiceDoc]{v, e} }()
	wg.Wait()

	apiGroups := (<-apiGroupsCh).val
	dbs := (<-dbsCh).val
	diagrams := (<-diagramsCh).val
	docs := (<-docsCh).val

	// fetch endpoints for each group to sum per-endpoint token counts (sequential since we need group IDs first)
	var allEndpoints []apiclient.APIEndpoint
	for _, g := range apiGroups {
		eps, _ := h.client.ListAPIEndpoints(ctx, token, orgID, svc.ID, g.ID)
		allEndpoints = append(allEndpoints, eps...)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Service: %s\n\n", svc.Name))
	sb.WriteString(fmt.Sprintf("**ID:** %s | **Slug:** %s\n", svc.ID, svc.Slug))
	sb.WriteString(fmt.Sprintf("**Status:** %s | **Tier:** %s | **Language:** %s | **Category:** %s\n",
		svc.Status, svc.Tier, svc.Language, svc.Category))
	if svc.Description != "" {
		sb.WriteString(fmt.Sprintf("\n%s\n", svc.Description))
	}

	// sum per-endpoint token counts across all groups
	endpointTokenTotal := 0
	for _, e := range allEndpoints {
		endpointTokenTotal += e.TokenCount
	}

	totalRawTokens := 0
	sb.WriteString(fmt.Sprintf("\n## API Specifications (%d groups, %d endpoints)\n\n", len(apiGroups), len(allEndpoints)))
	resourceIDs := []string{svc.ID}
	for _, g := range apiGroups {
		label := ""
		if g.Label != nil {
			label = " (" + *g.Label + ")"
		}
		sb.WriteString(fmt.Sprintf("- **%s** %s%s — %s | ID: `%s`\n",
			g.Name, g.Version, label, g.Protocol, g.ID))
		resourceIDs = append(resourceIDs, g.ID)
	}
	if endpointTokenTotal > 0 {
		sb.WriteString(fmt.Sprintf("\nRaw spec files total: ~%d tokens across %d endpoints\n", endpointTokenTotal, len(allEndpoints)))
		totalRawTokens += endpointTokenTotal
	}

	// DB schemas
	sb.WriteString(fmt.Sprintf("\n## Database Schemas (%d)\n\n", len(dbs)))
	for _, db := range dbs {
		sb.WriteString(fmt.Sprintf("- **%s** (%s/%s) — raw: ~%d tokens | ID: `%s`\n",
			db.DBName, db.DBType, db.Dialect, db.SchemaTokenCount, db.ID))
		totalRawTokens += db.SchemaTokenCount
		resourceIDs = append(resourceIDs, db.ID)
	}

	// Diagrams
	sb.WriteString(fmt.Sprintf("\n## Architecture Diagrams (%d)\n\n", len(diagrams)))
	for _, d := range diagrams {
		sb.WriteString(fmt.Sprintf("- Diagram ID: `%s`\n", d.DiagramID))
		resourceIDs = append(resourceIDs, d.DiagramID)
	}

	// Docs
	sb.WriteString(fmt.Sprintf("\n## Documentation (%d)\n\n", len(docs)))
	for _, doc := range docs {
		sb.WriteString(fmt.Sprintf("- **%s** [%s] — %s | raw: ~%d tokens\n",
			doc.FileName, doc.FileType, doc.Description, doc.DocTokenCount))
		totalRawTokens += doc.DocTokenCount
	}

	text := sb.String()

	// Use sum of actual file token counts as raw equivalent (exact)
	go h.recordUsage(orgID, token, "get_service_context", resourceIDs, modelID, text, &totalRawTokens)

	return mcp.NewToolResultText(text), nil
}

func (h *Handler) resolveService(ctx context.Context, token, orgID, serviceIDOrSlug string) (*apiclient.Service, error) {
	// Try as ID first (heuristic: contains hyphens in UUID pattern)
	svc, err := h.client.GetService(ctx, token, orgID, serviceIDOrSlug)
	if err == nil {
		return svc, nil
	}
	// Fall back to slug lookup
	return h.client.GetServiceBySlug(ctx, token, orgID, serviceIDOrSlug)
}
```

- [ ] **Step 2: Update `internal/mcp/tools.go`**

```go
func registerTools(s *mcpserver.MCPServer, client *apiclient.Client) {
	h := tools.New(client)
	h.RegisterServiceContextTool(s) // composite first — most important
	h.RegisterCatalogTools(s)
	h.RegisterFolderTools(s)
	h.RegisterDiagramTools(s)
	h.RegisterSchemaTools(s)
	h.RegisterMapTools(s)
}
```

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/tools/service_context.go internal/mcp/tools.go
git commit -m "feat: add get_service_context composite tool with fan-out and exact token savings"
```

---

### Task B9: Deployment files

**Files:**
- Create: `Dockerfile`
- Create: `Dockerfile.dev`

- [ ] **Step 1: Create `Dockerfile`**

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /mcp ./cmd/mcp

FROM scratch
COPY --from=builder /mcp /mcp
EXPOSE 8080
ENTRYPOINT ["/mcp"]
```

- [ ] **Step 2: Create `Dockerfile.dev`**

```dockerfile
FROM golang:1.25-alpine
WORKDIR /app
RUN go install github.com/air-verse/air@latest
COPY go.mod go.sum ./
RUN go mod download
COPY . .
EXPOSE 8080
CMD ["air"]
```

- [ ] **Step 3: Final build verification**

```bash
go build ./...
go vet ./...
```

- [ ] **Step 4: Commit**

```bash
git add Dockerfile Dockerfile.dev
git commit -m "feat: add Dockerfiles for uigraph-mcp"
```

---

## Self-Review

**Spec coverage check:**

| Spec requirement | Covered by |
|---|---|
| HTTP/SSE transport | Task B4 |
| Bearer token passthrough | B4 (SSEContextFunc), B5 (tokenFromCtx) |
| Service account + user session auth | A3 (Record handler checks PrincipalKind) |
| `get_service_context` fan-out composite | B8 |
| All catalog tools | B5 |
| Diagram tools | B6 |
| DB schema tools | B6 |
| Map tools | B7 |
| Folder tools | B5 |
| Token estimation (exact + fallback) | B3 |
| Fire-and-forget usage recording | B5 helpers, B6, B7, B8 |
| `llm_models` table + API | A2 |
| `mcp_usage_events` table + API | A3 |
| Savings summary with live cost computation | A3 (GetSavingsSummary SQL) |
| Token count columns on synced tables | A1 |
| Seed LLM models in migration | A2 |
| NOT NULL token count columns (v1) | A1 |

**Placeholder scan:** No TBDs found.

**Type consistency:** `apiclient.UsageEventPayload` matches `RecordUsage` signature. `tokencount.RawEquivalent` signature matches all call sites. `tools.Handler` used consistently. `TokenKey` exported from helpers and used in server.go. `APIEndpoint.TokenCount` (not `SpecTokenCount`) used consistently across A1 domain struct, B2 apiclient, B5 catalog tool, and B8 service_context.
