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
		mcp.WithString("type", mcp.Description("Filter by type: service, diagram, map, doc")),
	), h.listFolders)
}

func (h *Handler) listFolders(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
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
		sb.WriteString(fmt.Sprintf("- **FolderID:** `%s`\n", f.ID))
		sb.WriteString(fmt.Sprintf("  - **Name:** %s\n", f.Name))
		sb.WriteString(fmt.Sprintf("  - **Type:** %s\n", f.Type))
		if f.ParentID != nil {
			sb.WriteString(fmt.Sprintf("  - **ParentID:** `%s`\n", *f.ParentID))
		}
		sb.WriteString("\n")
	}
	if len(folders) == 0 {
		sb.WriteString("No folders found.\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}
