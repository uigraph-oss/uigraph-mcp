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

// OrgKey is the context key under which the inbound default org id is stored.
// Exported so internal/mcp can inject it from the SSE context func.
const OrgKey contextKey = "org"

// tokenFromCtx retrieves the bearer token injected into the request context.
func tokenFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(TokenKey).(string)
	return v
}

// WithToken returns a new context carrying the bearer token.
func WithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, TokenKey, token)
}

// orgFromCtx retrieves the default org id injected into the request context.
func orgFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(OrgKey).(string)
	return v
}

// WithOrg returns a new context carrying the default org id.
func WithOrg(ctx context.Context, org string) context.Context {
	return context.WithValue(ctx, OrgKey, org)
}

// orgID resolves the org for a tool call: the explicit org_id argument when
// provided, otherwise the default org injected from the request header. The
// server is strict — it never resolves or guesses an org. The client is
// responsible for sending an explicit org.
func (h *Handler) orgID(ctx context.Context, req mcp.CallToolRequest) (string, error) {
	if v := req.GetString("org_id", ""); v != "" {
		return v, nil
	}
	if v := orgFromCtx(ctx); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("org_id is required")
}

// recordUsage fires-and-forgets a usage event to uigraph-api.
func (h *Handler) recordUsage(orgID, token, toolName string, resourceIDs []string, modelID, responseText string, exactFileTokens *int) {
	ctx := context.Background()
	served := tokencount.Count(responseText)
	raw := tokencount.RawEquivalent(toolName, served, exactFileTokens)
	payload := apiclient.UsageEventPayload{
		ToolName:            toolName,
		ResourceIDs:         resourceIDs,
		ModelID:             modelID,
		TokensServed:        served,
		TokensRawEquivalent: raw,
		TokensSaved:         raw - served,
		ResponseSizeBytes:   len(responseText),
	}
	if err := h.client.RecordUsage(ctx, token, orgID, payload); err != nil {
		slog.Warn("failed to record MCP usage", "tool", toolName, "err", err)
	}
}
