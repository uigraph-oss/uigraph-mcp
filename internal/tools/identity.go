package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func (h *Handler) RegisterUserTool(s *mcpserver.MCPServer) {
	s.AddTool(mcp.NewTool("get_current_user",
		mcp.WithDescription("Get the currently authenticated uigraph identity: user (or service account) profile and the organisations they belong to. Use this to find out who is logged in and which org IDs are available."),
	), h.getCurrentUser)
}

func (h *Handler) getCurrentUser(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	token := tokenFromCtx(ctx)
	if token == "" {
		return mcp.NewToolResultError("not authenticated"), nil
	}

	me, err := h.client.GetMe(ctx, token)
	if err != nil {
		return mcp.NewToolResultError("failed to resolve identity"), nil
	}

	orgs, err := h.client.GetMyOrgs(ctx, token)
	if err != nil {
		return mcp.NewToolResultError("failed to resolve organisations"), nil
	}

	var sb strings.Builder
	sb.WriteString("# Current User\n\n")
	sb.WriteString(fmt.Sprintf("- **UserID:** `%s`\n", me.UserID))
	sb.WriteString(fmt.Sprintf("- **Name:** %s\n", me.Name))
	sb.WriteString(fmt.Sprintf("- **Email:** %s\n", me.Email))
	sb.WriteString(fmt.Sprintf("- **Login:** %s\n", me.Login))
	sb.WriteString(fmt.Sprintf("- **Kind:** %s\n", me.Kind))
	sb.WriteString(fmt.Sprintf("- **Role:** %s\n", me.Role))
	sb.WriteString(fmt.Sprintf("- **AuthProvider:** %s\n", me.AuthProvider))
	if me.OrgID != "" {
		sb.WriteString(fmt.Sprintf("- **CurrentOrgID:** `%s`\n", me.OrgID))
	}

	sb.WriteString(fmt.Sprintf("\n## Organisations (%d)\n\n", len(orgs)))
	for _, o := range orgs {
		sb.WriteString(fmt.Sprintf("- **OrgID:** `%s`\n", o.ID))
		sb.WriteString(fmt.Sprintf("  - **Name:** %s\n", o.Name))
		sb.WriteString(fmt.Sprintf("  - **Role:** %s\n", o.Role))
	}

	return mcp.NewToolResultText(sb.String()), nil
}
