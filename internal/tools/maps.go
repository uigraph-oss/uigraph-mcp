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
		mcp.WithString("search_by_name", mcp.Description("Optional filter matching map name")),
	), h.listMaps)

	s.AddTool(mcp.NewTool("get_map",
		mcp.WithDescription("Get a UI journey map with all its frames"),
		mcp.WithString("map_id", mcp.Required(), mcp.Description("Map ID (UUID)")),
	), h.getMap)
}

func (h *Handler) listMaps(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	var search *string
	if s := req.GetString("search_by_name", ""); s != "" {
		search = &s
	}

	maps, err := h.client.ListMaps(ctx, token, orgID, nil, nil, search)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# UI journey maps\n\n")
	for _, m := range maps {
		sb.WriteString(fmt.Sprintf("- **MapID:** `%s`\n", m.ID))
		sb.WriteString(fmt.Sprintf("  - **Name:** %s\n", m.Name))
		sb.WriteString(fmt.Sprintf("  - **Status:** %s\n", m.Status))
		if m.Description != "" {
			sb.WriteString(fmt.Sprintf("  - **Description:** %s\n", m.Description))
		}
		sb.WriteString("\n")
	}
	if len(maps) == 0 {
		sb.WriteString("No maps found.\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) getMap(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	mapID, err := requireUUID(req, "map_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
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
	sb.WriteString(fmt.Sprintf("- **MapID:** `%s`\n", m.ID))
	sb.WriteString(fmt.Sprintf("- **Name:** %s\n", m.Name))
	sb.WriteString(fmt.Sprintf("- **Status:** %s\n", m.Status))
	if m.Description != "" {
		sb.WriteString(fmt.Sprintf("- **Description:** %s\n", m.Description))
	}
	sb.WriteString(fmt.Sprintf("\n## Frames (%d)\n\n", len(frames)))
	for _, f := range frames {
		indent := ""
		if f.ParentFrameID != nil {
			indent = "  "
		}
		sb.WriteString(fmt.Sprintf("%s- **FrameID:** `%s`\n", indent, f.ID))
		sb.WriteString(fmt.Sprintf("%s  - **Name:** %s\n", indent, f.Name))
		sb.WriteString(fmt.Sprintf("%s  - **Template:** %s\n", indent, f.TemplateType))
		sb.WriteString(fmt.Sprintf("%s  - **Status:** %s\n", indent, f.Status))
		if f.ParentFrameID != nil {
			sb.WriteString(fmt.Sprintf("%s  - **ParentFrameID:** `%s`\n", indent, *f.ParentFrameID))
		}
		if f.Description != "" {
			sb.WriteString(fmt.Sprintf("%s  - **Description:** %s\n", indent, f.Description))
		}
		sb.WriteString("\n")
	}
	if len(frames) == 0 {
		sb.WriteString("No frames found.\n")
	}

	text := sb.String()
	go h.recordUsage(ctx, orgID, token, "get_map", []string{mapID}, text, nil)
	return mcp.NewToolResultText(text), nil
}
