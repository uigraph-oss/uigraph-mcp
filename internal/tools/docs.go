package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func (h *Handler) RegisterDocTools(s *mcpserver.MCPServer) {
	s.AddTool(mcp.NewTool(
		"list_docs",
		mcp.WithDescription("List documents in a UIGraph organisation, or documents attached to a service when service_id is provided"),
		mcp.WithString("search_by_name", mcp.Description("Optional filter matching document file name or description")),
		mcp.WithString("service_id", mcp.Description("Optional service ID (UUID) to list only documents attached to that service")),
	), h.listDocs)

	s.AddTool(mcp.NewTool(
		"get_doc",
		mcp.WithDescription("Get a document's metadata plus its content inlined for common text and image types"),
		mcp.WithString("doc_id", mcp.Required(), mcp.Description("Document ID (UUID)")),
	), h.getDoc)
}

func (h *Handler) listDocs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	serviceID, err := optionalUUID(req, "service_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if serviceID != nil {
		docs, err := h.client.ListServiceDocs(ctx, token, orgID, *serviceID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var sb strings.Builder
		sb.WriteString("# Service documents\n\n")
		for _, d := range docs {
			sb.WriteString(fmt.Sprintf("- **DocID:** `%s`\n", d.ID))
			sb.WriteString(fmt.Sprintf("  - **FileName:** %s\n", d.FileName))
			sb.WriteString(fmt.Sprintf("  - **FileType:** %s\n", d.FileType))
			if d.Description != "" {
				sb.WriteString(fmt.Sprintf("  - **Description:** %s\n", d.Description))
			}
			sb.WriteString("\n")
		}
		if len(docs) == 0 {
			sb.WriteString("No documents found.\n")
		}
		return mcp.NewToolResultText(sb.String()), nil
	}

	var search *string
	if s := req.GetString("search_by_name", ""); s != "" {
		search = &s
	}

	docs, err := h.client.ListDocs(ctx, token, orgID, search)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# Documents\n\n")
	for _, d := range docs {
		sb.WriteString(fmt.Sprintf("- **DocID:** `%s`\n", d.ID))
		sb.WriteString(fmt.Sprintf("  - **FileName:** %s\n", d.FileName))
		sb.WriteString(fmt.Sprintf("  - **FileType:** %s\n", d.FileType))
		if d.Description != "" {
			sb.WriteString(fmt.Sprintf("  - **Description:** %s\n", d.Description))
		}
		sb.WriteString("\n")
	}
	if len(docs) == 0 {
		sb.WriteString("No documents found.\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) getDoc(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	docID, err := requireUUID(req, "doc_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	doc, err := h.client.GetDoc(ctx, token, orgID, docID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("- **DocID:** `%s`\n", doc.ID))
	sb.WriteString(fmt.Sprintf("- **FileName:** %s\n", doc.FileName))
	sb.WriteString(fmt.Sprintf("- **FileType:** %s\n", doc.FileType))
	if doc.Description != "" {
		sb.WriteString(fmt.Sprintf("- **Description:** %s\n", doc.Description))
	}

	content, err := h.client.GetDocContent(ctx, token, orgID, docID)
	if err != nil {
		sb.WriteString(fmt.Sprintf("\n[Content unavailable: %s]\n", err.Error()))
		return mcp.NewToolResultText(sb.String()), nil
	}

	if isImageType(doc.FileType) {
		sb.WriteString("\n## Content\n\n[image rendered below]\n")
		return mcp.NewToolResultImage(sb.String(), base64.StdEncoding.EncodeToString(content), doc.FileType), nil
	}
	if isTextType(doc.FileType) {
		sb.WriteString("\n## Content\n\n")
		sb.WriteString(string(content))
		sb.WriteString("\n")
		return mcp.NewToolResultText(sb.String()), nil
	}

	sb.WriteString(fmt.Sprintf("\n## Content\n\n[binary content (%d bytes) not rendered for type %s]\n", len(content), doc.FileType))
	return mcp.NewToolResultText(sb.String()), nil
}

func isImageType(fileType string) bool {
	t := strings.ToLower(fileType)
	if t == "image/svg+xml" {
		return false
	}
	return strings.HasPrefix(t, "image/")
}

func isTextType(fileType string) bool {
	t := strings.ToLower(fileType)
	if strings.HasPrefix(t, "text/") {
		return true
	}
	for _, marker := range []string{"json", "yaml", "xml", "markdown", "javascript", "csv", "html", "svg", "plain"} {
		if strings.Contains(t, marker) {
			return true
		}
	}
	return false
}
