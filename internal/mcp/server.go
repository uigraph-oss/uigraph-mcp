package mcp

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

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
	hooks.AddAfterCallTool(func(ctx context.Context, id any, req *mcp.CallToolRequest, result any) {
		res, ok := result.(*mcp.CallToolResult)
		if !ok {
			return
		}
		if res.IsError {
			name, version := tools.ClientFromCtx(ctx)
			slog.Error("tool call failed",
				"tool", req.Params.Name, "client", name, "clientVersion", version,
				"error", toolResultText(res))
			return
		}
		go tools.RecordToolCall(ctx, client, req.Params.Name, toolResultText(res))
	})
	hooks.AddOnError(func(ctx context.Context, id any, method mcp.MCPMethod, message any, err error) {
		slog.Error("mcp error", "method", method, "err", err)
	})

	s := mcpserver.NewMCPServer(cfg.MCPServerName, cfg.MCPServerVersion, mcpserver.WithHooks(hooks))

	registerTools(s, client)

	streamable := mcpserver.NewStreamableHTTPServer(s,
		mcpserver.WithStateLess(true),
		mcpserver.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			token, scheme := extractCredential(r)
			ctx = tools.WithToken(ctx, token)
			ctx = apiclient.WithScheme(ctx, scheme)
			ctx = tools.WithOrg(ctx, r.Header.Get("X-UIGraph-Org-Id"))
			ctx = tools.WithClient(ctx, r.Header.Get("X-UIGraph-Client-Name"), r.Header.Get("X-UIGraph-Client-Version"))
			ctx = tools.WithStart(ctx, time.Now())
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

func toolResultText(res *mcp.CallToolResult) string {
	var sb strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	return sb.String()
}

func extractCredential(r *http.Request) (string, apiclient.Scheme) {
	if key := r.Header.Get("X-API-Key"); key != "" {
		return key, apiclient.SchemeAPIKey
	}
	if after, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
		return after, apiclient.SchemeBearer
	}
	return "", ""
}
