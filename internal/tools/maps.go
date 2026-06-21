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
	orgID, err := req.RequireString("org_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
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
	orgID, err := req.RequireString("org_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	mapID, err := req.RequireString("map_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	modelID := req.GetString("model_id", "")
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
