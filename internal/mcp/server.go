package mcp

import (
	"net/http"
	"strings"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/uigraph/mcp/internal/apiclient"
	"github.com/uigraph/mcp/internal/config"
)

// New builds the MCP HTTP/SSE handler with all tools registered.
func New(cfg *config.Config, client *apiclient.Client) http.Handler {
	s := mcpserver.NewMCPServer(cfg.MCPServerName, cfg.MCPServerVersion)

	registerTools(s, client)

	sse := mcpserver.NewSSEServer(s,
		mcpserver.WithBaseURL("http://0.0.0.0:"+cfg.Port),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/", sse)
	return mux
}

// extractToken pulls the bearer token from the Authorization header.
func extractToken(r *http.Request) string {
	v := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(v, "Bearer "); ok {
		return after
	}
	return ""
}
