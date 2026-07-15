package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func (h *Handler) RegisterTeamTools(s *mcpserver.MCPServer) {
	s.AddTool(mcp.NewTool("list_teams",
		mcp.WithDescription("List teams in a UIGraph organisation"),
	), h.listTeams)

	s.AddTool(mcp.NewTool("get_team",
		mcp.WithDescription("Get a team with its members"),
		mcp.WithString("team_id", mcp.Required(), mcp.Description("Team ID")),
	), h.getTeam)
}

func (h *Handler) listTeams(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	teams, err := h.client.ListTeams(ctx, token, orgID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# Teams\n\n")
	for _, t := range teams {
		sb.WriteString(fmt.Sprintf("- **TeamID:** `%s`\n", t.ID))
		sb.WriteString(fmt.Sprintf("  - **Name:** %s\n", t.Name))
		if t.Email != "" {
			sb.WriteString(fmt.Sprintf("  - **Email:** %s\n", t.Email))
		}
		if t.ExternalID != "" {
			sb.WriteString(fmt.Sprintf("  - **ExternalID:** %s\n", t.ExternalID))
		}
		sb.WriteString(fmt.Sprintf("  - **MemberCount:** %d\n", t.MemberCount))
		sb.WriteString("\n")
	}
	if len(teams) == 0 {
		sb.WriteString("No teams found.\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) getTeam(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	teamID, err := req.RequireString("team_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	t, err := h.client.GetTeam(ctx, token, orgID, teamID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("- **TeamID:** `%s`\n", t.ID))
	sb.WriteString(fmt.Sprintf("- **Name:** %s\n", t.Name))
	if t.Email != "" {
		sb.WriteString(fmt.Sprintf("- **Email:** %s\n", t.Email))
	}
	if t.ExternalID != "" {
		sb.WriteString(fmt.Sprintf("- **ExternalID:** %s\n", t.ExternalID))
	}
	sb.WriteString(fmt.Sprintf("- **MemberCount:** %d\n", t.MemberCount))

	members, err := h.client.ListTeamMembers(ctx, token, orgID, teamID)
	if err == nil {
		sb.WriteString(fmt.Sprintf("\n## Members (%d)\n\n", len(members)))
		for _, m := range members {
			sb.WriteString(fmt.Sprintf("- **UserID:** `%s` — **Permission:** %s\n", m.UserID, m.Permission))
		}
	}
	return mcp.NewToolResultText(sb.String()), nil
}
