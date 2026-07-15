package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Scheme string

const (
	SchemeAPIKey Scheme = "api-key"
	SchemeBearer Scheme = "bearer"
)

type schemeKey struct{}

func WithScheme(ctx context.Context, scheme Scheme) context.Context {
	return context.WithValue(ctx, schemeKey{}, scheme)
}

func schemeFromCtx(ctx context.Context) Scheme {
	v, _ := ctx.Value(schemeKey{}).(Scheme)
	return v
}

type Client struct {
	baseURL    string
	gatewayURL string
	httpClient *http.Client
}

func New(baseURL, gatewayURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		gatewayURL: gatewayURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func setAuth(ctx context.Context, req *http.Request, token string) {
	scheme := schemeFromCtx(ctx)
	if scheme == SchemeAPIKey {
		req.Header.Set("X-API-Key", token)
		return
	}
	if scheme == SchemeBearer {
		req.Header.Set("Authorization", "Bearer "+token)
		return
	}
}

func setGatewayAuth(ctx context.Context, req *http.Request, token string) {
	scheme := schemeFromCtx(ctx)
	if scheme == SchemeAPIKey {
		req.Header.Set("X-API-Token", token)
		return
	}
	if scheme == SchemeBearer {
		req.Header.Set("Authorization", "Bearer "+token)
		return
	}
}

func (c *Client) get(ctx context.Context, token, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("apiclient: build request: %w", err)
	}
	setAuth(ctx, req, token)
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

func (c *Client) postGateway(ctx context.Context, token, path string, in, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("apiclient: marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.gatewayURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("apiclient: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	setGatewayAuth(ctx, req, token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("apiclient: do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("apiclient: %s %s → %d: %s", http.MethodPost, path, resp.StatusCode, b)
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
	setAuth(ctx, req, token)
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

func (c *Client) URLExists(ctx context.Context, url string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

var ErrNotFound = fmt.Errorf("not found")
