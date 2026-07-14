package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func (h *Handler) RegisterDiagramTools(s *mcpserver.MCPServer) {
	s.AddTool(mcp.NewTool("list_diagrams",
		mcp.WithDescription("List architecture diagrams in a UIGraph organisation"),
		mcp.WithString("search_by_name", mcp.Description("Optional filter matching diagram name")),
	), h.listDiagrams)

	s.AddTool(mcp.NewTool("get_diagram",
		mcp.WithDescription("Get an architecture diagram's details and a mermaid rendering of it"),
		mcp.WithString("diagram_id", mcp.Required(), mcp.Description("Diagram ID")),
		mcp.WithBoolean("include_content", mcp.Description("Include the raw ReactFlow diagram content in addition to the mermaid rendering")),
		mcp.WithBoolean("include_thumbnail", mcp.Description("Include a thumbnailURL for the diagram's preview image, if one exists")),
	), h.getDiagram)
}

func (h *Handler) listDiagrams(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	var search *string
	if s := req.GetString("search_by_name", ""); s != "" {
		search = &s
	}

	diagrams, err := h.client.ListDiagrams(ctx, token, orgID, nil, nil, search)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# Architecture diagrams\n\n")
	for _, d := range diagrams {
		sb.WriteString(fmt.Sprintf("- **DiagramID:** `%s`\n", d.ID))
		sb.WriteString(fmt.Sprintf("  - **Name:** %s\n", d.Name))
		sb.WriteString(fmt.Sprintf("  - **Tokens:** ~%d\n", d.ContentTokenCount))
		sb.WriteString("\n")
	}
	if len(diagrams) == 0 {
		sb.WriteString("No diagrams found.\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) getDiagram(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	diagramID, err := req.RequireString("diagram_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	dg, err := h.client.GetDiagram(ctx, token, orgID, diagramID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	content, err := h.client.GetDiagramContent(ctx, token, orgID, diagramID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	mermaid, err := h.client.ConvertDiagramToMermaid(ctx, token, string(content))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", dg.Name))
	sb.WriteString(fmt.Sprintf("- **DiagramID:** `%s`\n", dg.ID))
	sb.WriteString(fmt.Sprintf("- **Tokens:** ~%d\n", dg.ContentTokenCount))
	sb.WriteString(fmt.Sprintf("- **Updated:** %s\n\n", dg.UpdatedAt.Format(time.RFC3339)))
	sb.WriteString("## Mermaid\n\n")
	sb.WriteString("```mermaid\n")
	sb.WriteString(mermaid)
	sb.WriteString("\n```\n")

	if req.GetBool("include_content", false) {
		const maxChars = 100_000
		raw := string(content)
		truncated := false
		if len(raw) > maxChars {
			raw = raw[:maxChars]
			truncated = true
		}
		sb.WriteString("\n## Raw content\n\n")
		sb.WriteString("```json\n")
		sb.WriteString(raw)
		sb.WriteString("\n```\n")
		if truncated {
			sb.WriteString("\n[Truncated at 100,000 characters]\n")
		}
	}

	text := sb.String()

	exactTokens := dg.ContentTokenCount
	go h.recordUsage(ctx, orgID, token, "get_diagram", []string{diagramID}, text, &exactTokens)

	if req.GetBool("include_thumbnail", false) {
		if url := h.thumbnailURL(ctx, token, orgID, diagramID); url != "" {
			text = fmt.Sprintf("**thumbnailURL:** %s\n\n%s", url, text)
		}
	}

	return mcp.NewToolResultText(text), nil
}

func (h *Handler) thumbnailURL(ctx context.Context, token, orgID, diagramID string) string {
	dg, err := h.client.GetDiagram(ctx, token, orgID, diagramID)
	if err != nil || dg == nil || dg.PreviewStatus != "success" || dg.PreviewAssetID == nil {
		return ""
	}
	url, err := h.client.ResolveAssetURL(ctx, token, orgID, *dg.PreviewAssetID)
	if err != nil || url == "" {
		return ""
	}
	if !h.client.URLExists(ctx, url) {
		return ""
	}
	return url
}
