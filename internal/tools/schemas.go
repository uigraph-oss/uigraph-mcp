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
		mcp.WithString("org_id", mcp.Description("Organisation ID (defaults to the configured default org)")),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
	), h.listServiceDBs)

	s.AddTool(mcp.NewTool("get_db_schema",
		mcp.WithDescription("Get the full database schema for a service DB"),
		mcp.WithString("org_id", mcp.Description("Organisation ID (defaults to the configured default org)")),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
		mcp.WithString("db_id", mcp.Required(), mcp.Description("Service DB ID")),
		mcp.WithString("model_id", mcp.Description("LLM model ID for cost tracking")),
	), h.getDBSchema)
}

func (h *Handler) listServiceDBs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	serviceID, err := req.RequireString("service_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	dbs, err := h.client.ListServiceDBs(ctx, token, orgID, serviceID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# Database schemas\n\n")
	for _, db := range dbs {
		sb.WriteString(fmt.Sprintf("- **DatabaseID:** `%s`\n", db.ID))
		sb.WriteString(fmt.Sprintf("  - **Name:** %s\n", db.DBName))
		sb.WriteString(fmt.Sprintf("  - **Type:** %s\n", db.DBType))
		sb.WriteString(fmt.Sprintf("  - **Dialect:** %s\n", db.Dialect))
		sb.WriteString(fmt.Sprintf("  - **Tokens:** ~%d\n", db.SchemaTokenCount))
		sb.WriteString("\n")
	}
	if len(dbs) == 0 {
		sb.WriteString("No databases found.\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) getDBSchema(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	serviceID, err := req.RequireString("service_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	dbID, err := req.RequireString("db_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	modelID := req.GetString("model_id", "")
	if modelID == "" {
		modelID = "claude-sonnet-4-6"
	}
	token := tokenFromCtx(ctx)

	db, err := h.client.GetServiceDB(ctx, token, orgID, serviceID, dbID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	text := fmt.Sprintf("- **DatabaseID:** `%s`\n", db.ID)
	text += fmt.Sprintf("- **Name:** %s\n", db.DBName)
	text += fmt.Sprintf("- **Type:** %s\n", db.DBType)
	text += fmt.Sprintf("- **Dialect:** %s\n", db.Dialect)
	// schema JSON is returned as part of the ServiceDB struct — the apiclient
	// would need to include SchemaJSON. For now format available metadata.
	text += fmt.Sprintf("- **Tokens:** ~%d\n", db.SchemaTokenCount)

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
