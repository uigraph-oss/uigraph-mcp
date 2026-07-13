package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func (h *Handler) RegisterDiagramTools(s *mcpserver.MCPServer) {
	s.AddTool(mcp.NewTool("list_diagrams",
		mcp.WithDescription("List architecture diagrams in a UIGraph organisation"),
		mcp.WithString("org_id", mcp.Description("Organisation ID (defaults to the configured default org)")),
	), h.listDiagrams)

	s.AddTool(mcp.NewTool("get_diagram",
		mcp.WithDescription("Get the content of an architecture diagram"),
		mcp.WithString("org_id", mcp.Description("Organisation ID (defaults to the configured default org)")),
		mcp.WithString("diagram_id", mcp.Required(), mcp.Description("Diagram ID")),
		mcp.WithBoolean("include_thumbnail", mcp.Description("Include a thumbnailURL for the diagram's preview image, if one exists")),
	), h.getDiagram)
}

func (h *Handler) listDiagrams(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	diagrams, err := h.client.ListDiagrams(ctx, token, orgID, nil, nil)
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
	orgID, err := h.orgID(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	diagramID, err := req.RequireString("diagram_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	content, err := h.client.GetDiagramContent(ctx, token, orgID, diagramID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	const maxChars = 100_000
	text := string(content)
	truncated := false
	if len(text) > maxChars {
		text = text[:maxChars]
		truncated = true
	}

	diagrams, _ := h.client.ListDiagrams(ctx, token, orgID, nil, nil)
	var exactTokens *int
	for _, d := range diagrams {
		if d.ID == diagramID {
			t := d.ContentTokenCount
			exactTokens = &t
			break
		}
	}

	go h.recordUsage(ctx, orgID, token, "get_diagram", []string{diagramID}, text, exactTokens)

	if truncated {
		text += "\n\n[Truncated at 100,000 characters]"
	}

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
