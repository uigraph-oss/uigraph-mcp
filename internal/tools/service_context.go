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
	orgID, err := req.RequireString("org_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	serviceIDOrSlug, err := req.RequireString("service_id_or_slug")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	modelID := req.GetString("model_id", "")
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
