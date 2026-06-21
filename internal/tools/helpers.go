package tools

import (
	"context"
	"log/slog"

	"github.com/uigraph/mcp/internal/apiclient"
	"github.com/uigraph/mcp/internal/tokencount"
)

type contextKey string

// TokenKey is the context key under which the inbound bearer token is stored.
// Exported so internal/mcp can inject it from the SSE context func.
const TokenKey contextKey = "bearer"

// tokenFromCtx retrieves the bearer token injected into the request context.
func tokenFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(TokenKey).(string)
	return v
}

// WithToken returns a new context carrying the bearer token.
func WithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, TokenKey, token)
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
