package tools

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/uigraph/mcp/internal/apiclient"
	"github.com/uigraph/mcp/internal/tokencount"
)

func requireUUID(req mcp.CallToolRequest, name string) (string, error) {
	v, err := req.RequireString(name)
	if err != nil {
		return "", err
	}
	if err := uuid.Validate(v); err != nil {
		return "", fmt.Errorf("%s must be a valid UUID, got %q", name, v)
	}
	return v, nil
}

func optionalUUID(req mcp.CallToolRequest, name string) (*string, error) {
	v := req.GetString(name, "")
	if v == "" {
		return nil, nil
	}
	if err := uuid.Validate(v); err != nil {
		return nil, fmt.Errorf("%s must be a valid UUID, got %q", name, v)
	}
	return &v, nil
}

type contextKey string

// TokenKey is the context key under which the inbound bearer token is stored.
// Exported so internal/mcp can inject it from the SSE context func.
const TokenKey contextKey = "bearer"

const OrgKey contextKey = "org"

const ClientNameKey contextKey = "clientName"

const ClientVersionKey contextKey = "clientVersion"

const StartKey contextKey = "start"

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

func WithStart(ctx context.Context, start time.Time) context.Context {
	return context.WithValue(ctx, StartKey, start)
}

func startFromCtx(ctx context.Context) (time.Time, bool) {
	start, ok := ctx.Value(StartKey).(time.Time)
	return start, ok
}

func durationMsFromCtx(ctx context.Context) int {
	start, ok := startFromCtx(ctx)
	if !ok {
		return 0
	}
	return int(time.Since(start).Milliseconds())
}

func ClientFromCtx(ctx context.Context) (string, string) {
	return clientFromCtx(ctx)
}

func (h *Handler) orgID(ctx context.Context) (string, error) {
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
		DurationMs:          durationMsFromCtx(reqCtx),
	}
	slog.Info("recording MCP usage",
		"tool", toolName, "org", orgID, "client", clientName,
		"tokensServed", served, "tokensSaved", raw-served)
	if err := h.client.RecordUsage(ctx, token, orgID, payload); err != nil {
		slog.Error("failed to record MCP usage", "tool", toolName, "org", orgID, "err", err)
		return
	}
	slog.Info("recorded MCP usage", "tool", toolName, "org", orgID)
}

var selfRecordingTools = map[string]bool{
	"get_service":   true,
	"get_api_spec":  true,
	"get_db_schema": true,
	"get_diagram":   true,
	"get_map":       true,
}

func RecordToolCall(reqCtx context.Context, client *apiclient.Client, toolName, responseText string) {
	if selfRecordingTools[toolName] {
		return
	}
	orgID := orgFromCtx(reqCtx)
	if orgID == "" {
		slog.Error("skip recording MCP usage: missing org", "tool", toolName)
		return
	}
	token := tokenFromCtx(reqCtx)
	clientName, clientVersion := clientFromCtx(reqCtx)
	ctx := context.WithoutCancel(reqCtx)
	served := tokencount.Count(responseText)
	raw := tokencount.RawEquivalent(toolName, served, nil)
	payload := apiclient.UsageEventPayload{
		ToolName:            toolName,
		TokensServed:        served,
		TokensRawEquivalent: raw,
		TokensSaved:         raw - served,
		ResponseSizeBytes:   len(responseText),
		ClientName:          clientName,
		ClientVersion:       clientVersion,
		DurationMs:          durationMsFromCtx(reqCtx),
	}
	slog.Info("recording MCP usage",
		"tool", toolName, "org", orgID, "client", clientName,
		"tokensServed", served, "tokensSaved", raw-served)
	if err := client.RecordUsage(ctx, token, orgID, payload); err != nil {
		slog.Error("failed to record MCP usage", "tool", toolName, "org", orgID, "err", err)
		return
	}
	slog.Info("recorded MCP usage", "tool", toolName, "org", orgID)
}
