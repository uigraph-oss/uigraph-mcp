package mcp

import (
	"context"
	"net/http"
	"strings"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/uigraph/mcp/internal/apiclient"
	"github.com/uigraph/mcp/internal/auth"
	"github.com/uigraph/mcp/internal/config"
	"github.com/uigraph/mcp/internal/tools"
)

// New builds the MCP HTTP/SSE handler with all tools registered.
func New(cfg *config.Config, client *apiclient.Client) http.Handler {
	s := mcpserver.NewMCPServer(cfg.MCPServerName, cfg.MCPServerVersion)

	registerTools(s, client)

	streamable := mcpserver.NewStreamableHTTPServer(s,
		mcpserver.WithStateLess(true),
		mcpserver.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			ctx = tools.WithToken(ctx, extractToken(r))
			ctx = tools.WithOrg(ctx, r.Header.Get("X-UIGraph-Org-Id"))
			return ctx
		}),
	)

	authH := auth.New(cfg, client)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /auth/login", authH.Login)
	mux.HandleFunc("GET /auth/callback", authH.Callback)
	mux.HandleFunc("GET /auth/me", authH.Me)
	mux.Handle("/", streamable)
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
