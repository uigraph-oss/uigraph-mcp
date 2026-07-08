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
		mcp.WithString("org_id", mcp.Description("Organisation ID (defaults to the configured default org)")),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID (UUID)")),
	), h.getServiceContext)
}

func (h *Handler) getServiceContext(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	serviceID, err := req.RequireString("service_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	svc, err := h.client.GetService(ctx, token, orgID, serviceID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("service not found: %s", serviceID)), nil
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
	sb.WriteString(fmt.Sprintf("- **ServiceID:** `%s`\n", svc.ID))
	sb.WriteString(fmt.Sprintf("- **Name:** %s\n", svc.Name))
	sb.WriteString(fmt.Sprintf("- **Status:** %s\n", svc.Status))
	sb.WriteString(fmt.Sprintf("- **Tier:** %s\n", svc.Tier))
	sb.WriteString(fmt.Sprintf("- **Language:** %s\n", svc.Language))
	sb.WriteString(fmt.Sprintf("- **Category:** %s\n", svc.Category))
	if svc.Description != "" {
		sb.WriteString(fmt.Sprintf("- **Description:** %s\n", svc.Description))
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
		sb.WriteString(fmt.Sprintf("- **APIGroupID:** `%s`\n", g.ID))
		sb.WriteString(fmt.Sprintf("  - **Name:** %s\n", g.Name))
		sb.WriteString(fmt.Sprintf("  - **Version:** %s\n", g.Version))
		sb.WriteString(fmt.Sprintf("  - **Protocol:** %s\n", g.Protocol))
		if g.Label != nil {
			sb.WriteString(fmt.Sprintf("  - **Label:** %s\n", *g.Label))
		}
		sb.WriteString("\n")
		resourceIDs = append(resourceIDs, g.ID)
	}
	if endpointTokenTotal > 0 {
		sb.WriteString(fmt.Sprintf("\nRaw spec files total: ~%d tokens across %d endpoints\n", endpointTokenTotal, len(allEndpoints)))
		totalRawTokens += endpointTokenTotal
	}

	// DB schemas
	sb.WriteString(fmt.Sprintf("\n## Database Schemas (%d)\n\n", len(dbs)))
	for _, db := range dbs {
		sb.WriteString(fmt.Sprintf("- **DatabaseID:** `%s`\n", db.ID))
		sb.WriteString(fmt.Sprintf("  - **Name:** %s\n", db.DBName))
		sb.WriteString(fmt.Sprintf("  - **Type:** %s\n", db.DBType))
		sb.WriteString(fmt.Sprintf("  - **Dialect:** %s\n", db.Dialect))
		sb.WriteString(fmt.Sprintf("  - **Tokens:** ~%d\n", db.SchemaTokenCount))
		sb.WriteString("\n")
		totalRawTokens += db.SchemaTokenCount
		resourceIDs = append(resourceIDs, db.ID)
	}

	// Diagrams
	sb.WriteString(fmt.Sprintf("\n## Architecture Diagrams (%d)\n\n", len(diagrams)))
	for _, d := range diagrams {
		sb.WriteString(fmt.Sprintf("- **DiagramID:** `%s`\n", d.DiagramID))
		resourceIDs = append(resourceIDs, d.DiagramID)
	}

	// Docs
	sb.WriteString(fmt.Sprintf("\n## Documentation (%d)\n\n", len(docs)))
	for _, doc := range docs {
		sb.WriteString(fmt.Sprintf("- **DocID:** `%s`\n", doc.ID))
		sb.WriteString(fmt.Sprintf("  - **FileName:** %s\n", doc.FileName))
		sb.WriteString(fmt.Sprintf("  - **FileType:** %s\n", doc.FileType))
		if doc.Description != "" {
			sb.WriteString(fmt.Sprintf("  - **Description:** %s\n", doc.Description))
		}
		sb.WriteString(fmt.Sprintf("  - **Tokens:** ~%d\n", doc.DocTokenCount))
		sb.WriteString("\n")
		totalRawTokens += doc.DocTokenCount
	}

	text := sb.String()

	// Use sum of actual file token counts as raw equivalent (exact)
	go h.recordUsage(ctx, orgID, token, "get_service_context", resourceIDs, text, &totalRawTokens)

	return mcp.NewToolResultText(text), nil
}
