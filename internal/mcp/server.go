package mcp

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/uigraph/mcp/internal/apiclient"
	"github.com/uigraph/mcp/internal/auth"
	"github.com/uigraph/mcp/internal/config"
	"github.com/uigraph/mcp/internal/tools"
)

func New(cfg *config.Config, client *apiclient.Client) http.Handler {
	hooks := &mcpserver.Hooks{}
	hooks.AddBeforeCallTool(func(ctx context.Context, id any, req *mcp.CallToolRequest) {
		name, version := tools.ClientFromCtx(ctx)
		slog.Info("tool call", "tool", req.Params.Name, "client", name, "clientVersion", version)
	})
	hooks.AddOnError(func(ctx context.Context, id any, method mcp.MCPMethod, message any, err error) {
		slog.Error("mcp error", "method", method, "err", err)
	})

	s := mcpserver.NewMCPServer(cfg.MCPServerName, cfg.MCPServerVersion, mcpserver.WithHooks(hooks))

	registerTools(s, client)

	streamable := mcpserver.NewStreamableHTTPServer(s,
		mcpserver.WithStateLess(true),
		mcpserver.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			ctx = tools.WithToken(ctx, extractToken(r))
			ctx = tools.WithOrg(ctx, r.Header.Get("X-UIGraph-Org-Id"))
			ctx = tools.WithClient(ctx, r.Header.Get("X-UIGraph-Client-Name"), r.Header.Get("X-UIGraph-Client-Version"))
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

func extractToken(r *http.Request) string {
	v := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(v, "Bearer "); ok {
		return after
	}
	return ""
}
