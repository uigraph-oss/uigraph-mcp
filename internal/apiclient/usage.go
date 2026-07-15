package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type UsageEventPayload struct {
	ToolName            string   `json:"toolName"`
	ResourceIDs         []string `json:"resourceIds"`
	TokensServed        int      `json:"tokensServed"`
	TokensRawEquivalent int      `json:"tokensRawEquivalent"`
	TokensSaved         int      `json:"tokensSaved"`
	ResponseSizeBytes   int      `json:"responseSizeBytes"`
	ClientName          string   `json:"clientName,omitempty"`
	ClientVersion       string   `json:"clientVersion,omitempty"`
	DurationMs          int      `json:"durationMs"`
}

func (c *Client) RecordUsage(ctx context.Context, token, orgID string, e UsageEventPayload) error {
	b, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("apiclient: marshal usage event: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+fmt.Sprintf("/api/v1/orgs/%s/mcp/usage", orgID), bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("apiclient: build usage request: %w", err)
	}
	setAuth(ctx, req, token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("apiclient: record usage: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("apiclient: record usage → %d: %s", resp.StatusCode, body)
	}
	return nil
}
