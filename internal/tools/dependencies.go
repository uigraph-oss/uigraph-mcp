package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func (h *Handler) RegisterDependencyTools(s *mcpserver.MCPServer) {
	s.AddTool(mcp.NewTool(
		"get_dependency_graph",
		mcp.WithDescription("Get the organisation-wide service dependency graph as a Mermaid diagram."),
	), h.getDependencyGraph)
}

func (h *Handler) getDependencyGraph(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	graph, err := h.client.DependencyGraph(ctx, token, orgID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(graph)), nil
}
