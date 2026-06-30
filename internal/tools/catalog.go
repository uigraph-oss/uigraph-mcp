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
		mcp.WithString("org_id", mcp.Description("Organisation ID (defaults to the configured default org)")),
		mcp.WithString("folder_id", mcp.Description("Optional folder ID filter")),
	), h.listServices)

	s.AddTool(mcp.NewTool("get_service",
		mcp.WithDescription("Get full details and stats for a service"),
		mcp.WithString("org_id", mcp.Description("Organisation ID (defaults to the configured default org)")),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
	), h.getService)

	s.AddTool(mcp.NewTool("list_api_groups",
		mcp.WithDescription("List API specification groups for a service"),
		mcp.WithString("org_id", mcp.Description("Organisation ID (defaults to the configured default org)")),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
	), h.listAPIGroups)

	s.AddTool(mcp.NewTool("get_api_spec",
		mcp.WithDescription("Get the full API specification content (OpenAPI/GraphQL/gRPC) for an API group"),
		mcp.WithString("org_id", mcp.Description("Organisation ID (defaults to the configured default org)")),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
		mcp.WithString("api_group_id", mcp.Required(), mcp.Description("API group ID")),
		mcp.WithString("model_id", mcp.Description("LLM model ID for cost tracking (e.g. claude-sonnet-4-6)")),
	), h.getAPISpec)

	s.AddTool(mcp.NewTool("list_endpoints",
		mcp.WithDescription("List all API endpoints for a service or API group"),
		mcp.WithString("org_id", mcp.Description("Organisation ID (defaults to the configured default org)")),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
		mcp.WithString("api_group_id", mcp.Required(), mcp.Description("API group ID")),
	), h.listEndpoints)
}

func (h *Handler) listServices(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	var folderID *string
	if fid := req.GetString("folder_id", ""); fid != "" {
		folderID = &fid
	}

	svcs, err := h.client.ListServices(ctx, token, orgID, folderID, nil)
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
	orgID, err := h.orgID(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	serviceID, err := req.RequireString("service_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
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
	orgID, err := h.orgID(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	serviceID, err := req.RequireString("service_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	apiGroupID, err := req.RequireString("api_group_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	modelID := req.GetString("model_id", "")
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
	orgID, err := h.orgID(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	serviceID, err := req.RequireString("service_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	apiGroupID, err := req.RequireString("api_group_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
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
