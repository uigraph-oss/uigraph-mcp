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
	orgID, err := req.RequireString("org_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	folderType := req.GetString("type", "")
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
