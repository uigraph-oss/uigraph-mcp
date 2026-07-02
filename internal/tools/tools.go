package tools

import "github.com/uigraph/mcp/internal/apiclient"

// Handler holds dependencies for all MCP tool implementations.
type Handler struct {
	client *apiclient.Client
}

func New(client *apiclient.Client) *Handler {
	return &Handler{client: client}
}
