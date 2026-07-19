package mcp

import (
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/uigraph/mcp/internal/apiclient"
	"github.com/uigraph/mcp/internal/tools"
)

func registerTools(s *mcpserver.MCPServer, client *apiclient.Client) {
	h := tools.New(client)
	h.RegisterUserTool(s)
	h.RegisterCatalogTools(s)
	h.RegisterDependencyTools(s)
	h.RegisterFolderTools(s)
	h.RegisterDiagramTools(s)
	h.RegisterSchemaTools(s)
	h.RegisterMapTools(s)
	h.RegisterTestTools(s)
	h.RegisterDocTools(s)
	h.RegisterTeamTools(s)
}
