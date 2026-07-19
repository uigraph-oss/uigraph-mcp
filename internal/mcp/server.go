package mcp

import (
	"context"
	"fmt"
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
	mux.Handle("/", authenticateMCP(client, streamable))
	return mux
}

func authenticateMCP(client *apiclient.Client, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, scheme := extractCredential(r)
		if token == "" {
			http.Error(w, "missing credentials", http.StatusUnauthorized)
			return
		}

		ctx := apiclient.WithScheme(r.Context(), scheme)
		me, err := client.GetMe(ctx, token)
		if err != nil {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		if scheme == apiclient.SchemeAPIKey && me.Kind != "service_account" {
			http.Error(w, "X-API-Key credentials must authenticate as a service account", http.StatusForbidden)
			return
		}
		if scheme == apiclient.SchemeBearer && me.Kind != "user" {
			http.Error(w, "Bearer credentials must authenticate as a user account", http.StatusForbidden)
			return
		}

		orgID, err := authenticatedOrg(ctx, client, token, me, r.Header.Get("X-UIGraph-Org-Id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		ctx = tools.WithOrg(ctx, orgID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func authenticatedOrg(ctx context.Context, client *apiclient.Client, token string, me *apiclient.Me, requestedOrgID string) (string, error) {
	if me.Kind == "service_account" {
		if requestedOrgID != "" {
			return "", fmt.Errorf("service accounts must not send X-UIGraph-Org-Id")
		}
		if me.OrgID == "" {
			return "", fmt.Errorf("service account has no organization")
		}
		return me.OrgID, nil
	}

	if me.Kind != "user" {
		return "", fmt.Errorf("unsupported authenticated identity kind")
	}

	if requestedOrgID == "" {
		return "", fmt.Errorf("X-UIGraph-Org-Id is required for user accounts")
	}

	orgs, err := client.GetMyOrgs(ctx, token)
	if err != nil {
		return "", fmt.Errorf("failed to resolve user organizations")
	}
	for _, org := range orgs {
		if org.ID == requestedOrgID {
			return requestedOrgID, nil
		}
	}
	return "", fmt.Errorf("user does not have access to organization")
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
