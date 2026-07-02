package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func setAuth(req *http.Request, token string) {
	if strings.HasPrefix(token, "uig_") {
		req.Header.Set("X-API-Key", token)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
}

func (c *Client) get(ctx context.Context, token, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("apiclient: build request: %w", err)
	}
	setAuth(req, token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("apiclient: do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("apiclient: %s %s → %d: %s", http.MethodGet, path, resp.StatusCode, body)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) getRaw(ctx context.Context, token, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("apiclient: build request: %w", err)
	}
	setAuth(req, token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("apiclient: do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("apiclient: %s → %d: %s", path, resp.StatusCode, body)
	}
	return io.ReadAll(resp.Body)
}

var ErrNotFound = fmt.Errorf("not found")
