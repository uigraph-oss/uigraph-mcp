package mcp

import (
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/uigraph/mcp/internal/apiclient"
	"github.com/uigraph/mcp/internal/tools"
)

func registerTools(s *mcpserver.MCPServer, client *apiclient.Client) {
	h := tools.New(client)
	h.RegisterCatalogTools(s)
	h.RegisterFolderTools(s)
}
