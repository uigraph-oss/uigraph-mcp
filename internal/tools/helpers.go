package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/uigraph/mcp/internal/apiclient"
	"github.com/uigraph/mcp/internal/tokencount"
)

type contextKey string

// TokenKey is the context key under which the inbound bearer token is stored.
// Exported so internal/mcp can inject it from the SSE context func.
const TokenKey contextKey = "bearer"

const OrgKey contextKey = "org"

const ClientNameKey contextKey = "clientName"

const ClientVersionKey contextKey = "clientVersion"

// tokenFromCtx retrieves the bearer token injected into the request context.
func tokenFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(TokenKey).(string)
	return v
}

// WithToken returns a new context carrying the bearer token.
func WithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, TokenKey, token)
}

func orgFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(OrgKey).(string)
	return v
}

func WithOrg(ctx context.Context, org string) context.Context {
	return context.WithValue(ctx, OrgKey, org)
}

func WithClient(ctx context.Context, name, version string) context.Context {
	ctx = context.WithValue(ctx, ClientNameKey, name)
	ctx = context.WithValue(ctx, ClientVersionKey, version)
	return ctx
}

func clientFromCtx(ctx context.Context) (string, string) {
	name, _ := ctx.Value(ClientNameKey).(string)
	version, _ := ctx.Value(ClientVersionKey).(string)
	return name, version
}

func (h *Handler) orgID(ctx context.Context, req mcp.CallToolRequest) (string, error) {
	if v := req.GetString("org_id", ""); v != "" {
		return v, nil
	}
	if v := orgFromCtx(ctx); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("org_id is required")
}

// recordUsage fires-and-forgets a usage event to uigraph-api. It reads the
// agentic tool's identity from reqCtx, then detaches from it for the outbound
// call so the request's cancellation does not abort the report.
func (h *Handler) recordUsage(reqCtx context.Context, orgID, token, toolName string, resourceIDs []string, responseText string, exactFileTokens *int) {
	clientName, clientVersion := clientFromCtx(reqCtx)
	ctx := context.WithoutCancel(reqCtx)
	served := tokencount.Count(responseText)
	raw := tokencount.RawEquivalent(toolName, served, exactFileTokens)
	payload := apiclient.UsageEventPayload{
		ToolName:            toolName,
		ResourceIDs:         resourceIDs,
		TokensServed:        served,
		TokensRawEquivalent: raw,
		TokensSaved:         raw - served,
		ResponseSizeBytes:   len(responseText),
		ClientName:          clientName,
		ClientVersion:       clientVersion,
	}
	if err := h.client.RecordUsage(ctx, token, orgID, payload); err != nil {
		slog.Warn("failed to record MCP usage", "tool", toolName, "err", err)
	}
}
