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
		mcp.WithString("folder_id", mcp.Description("Optional folder ID filter")),
	), h.listServices)

	s.AddTool(mcp.NewTool("get_service",
		mcp.WithDescription("Get full details and stats for a service"),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
	), h.getService)

	s.AddTool(mcp.NewTool("list_api_groups",
		mcp.WithDescription("List API specification groups for a service"),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
	), h.listAPIGroups)

	s.AddTool(mcp.NewTool("get_api_spec",
		mcp.WithDescription("Get the full API specification content (OpenAPI/GraphQL/gRPC) for an API group"),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
		mcp.WithString("api_group_id", mcp.Required(), mcp.Description("API group ID")),
	), h.getAPISpec)

	s.AddTool(mcp.NewTool("list_endpoints",
		mcp.WithDescription("List all API endpoints for a service or API group"),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
		mcp.WithString("api_group_id", mcp.Required(), mcp.Description("API group ID")),
	), h.listEndpoints)
}

func (h *Handler) listServices(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
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
	sb.WriteString(fmt.Sprintf("# Services in org `%s`\n\n", orgID))
	for _, s := range svcs {
		sb.WriteString(fmt.Sprintf("- **ServiceID:** `%s`\n", s.ID))
		sb.WriteString(fmt.Sprintf("  - **Name:** %s\n", s.Name))
		sb.WriteString(fmt.Sprintf("  - **Status:** %s\n", s.Status))
		sb.WriteString(fmt.Sprintf("  - **Tier:** %s\n", s.Tier))
		sb.WriteString(fmt.Sprintf("  - **Language:** %s\n", s.Language))
		if s.Description != "" {
			sb.WriteString(fmt.Sprintf("  - **Description:** %s\n", s.Description))
		}
		sb.WriteString("\n")
	}
	if len(svcs) == 0 {
		sb.WriteString("No services found.\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) getService(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
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
	sb.WriteString(fmt.Sprintf("- **ServiceID:** `%s`\n", svc.ID))
	sb.WriteString(fmt.Sprintf("- **Name:** %s\n", svc.Name))
	sb.WriteString(fmt.Sprintf("- **Status:** %s\n", svc.Status))
	sb.WriteString(fmt.Sprintf("- **Tier:** %s\n", svc.Tier))
	sb.WriteString(fmt.Sprintf("- **Language:** %s\n", svc.Language))
	sb.WriteString(fmt.Sprintf("- **Category:** %s\n", svc.Category))
	if len(svc.Labels) > 0 {
		sb.WriteString(fmt.Sprintf("- **Labels:** %s\n", strings.Join(svc.Labels, ", ")))
	}
	if svc.Description != "" {
		sb.WriteString(fmt.Sprintf("- **Description:** %s\n", svc.Description))
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) listAPIGroups(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
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
	sb.WriteString("# API groups\n\n")
	for _, g := range groups {
		sb.WriteString(fmt.Sprintf("- **APIGroupID:** `%s`\n", g.ID))
		sb.WriteString(fmt.Sprintf("  - **Name:** %s\n", g.Name))
		sb.WriteString(fmt.Sprintf("  - **Version:** %s\n", g.Version))
		sb.WriteString(fmt.Sprintf("  - **Protocol:** %s\n", g.Protocol))
		if g.Label != nil {
			sb.WriteString(fmt.Sprintf("  - **Label:** %s\n", *g.Label))
		}
		sb.WriteString("\n")
	}
	if len(groups) == 0 {
		sb.WriteString("No API groups found.\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) getAPISpec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
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

	go h.recordUsage(ctx, orgID, token, "get_api_spec", []string{apiGroupID}, spec, exactTokens)

	result := spec
	if truncated {
		result += "\n\n[Truncated at 50,000 characters]"
	}
	return mcp.NewToolResultText(result), nil
}

func (h *Handler) listEndpoints(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
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
	sb.WriteString("# API endpoints\n\n")
	for _, e := range endpoints {
		sb.WriteString(fmt.Sprintf("- **EndpointID:** `%s`\n", e.ID))
		sb.WriteString(fmt.Sprintf("  - **Method:** %s\n", e.Method))
		sb.WriteString(fmt.Sprintf("  - **Path:** %s\n", e.Path))
		if e.Summary != "" {
			sb.WriteString(fmt.Sprintf("  - **Summary:** %s\n", e.Summary))
		}
		if len(e.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("  - **Tags:** %s\n", strings.Join(e.Tags, ", ")))
		}
		sb.WriteString(fmt.Sprintf("  - **Tokens:** ~%d\n", e.TokenCount))
		sb.WriteString("\n")
	}
	if len(endpoints) == 0 {
		sb.WriteString("No endpoints found.\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}
