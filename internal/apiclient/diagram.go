package apiclient

import (
	"context"
	"fmt"
	"time"
)

type Diagram struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	ContentTokenCount int       `json:"contentTokenCount"`
	PreviewAssetID    *string   `json:"previewAssetId,omitempty"`
	PreviewStatus     string    `json:"previewStatus"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func (c *Client) ListDiagrams(ctx context.Context, token, orgID string, folderID, teamID *string) ([]Diagram, error) {
	path := fmt.Sprintf("/api/v1/orgs/%s/diagrams", orgID)
	var resp struct {
		Diagrams []Diagram `json:"diagrams"`
	}
	return resp.Diagrams, c.get(ctx, token, path, &resp)
}

func (c *Client) GetDiagramContent(ctx context.Context, token, orgID, diagramID string) ([]byte, error) {
	var resp struct {
		Content string `json:"content"`
	}
	if err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/diagrams/%s/content", orgID, diagramID), &resp); err != nil {
		return nil, err
	}
	return []byte(resp.Content), nil
}

func (c *Client) GetDiagram(ctx context.Context, token, orgID, diagramID string) (*Diagram, error) {
	var d Diagram
	if err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/diagrams/%s", orgID, diagramID), &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (c *Client) ConvertDiagramToMermaid(ctx context.Context, token, content string) (string, error) {
	req := struct {
		Content string `json:"content"`
	}{Content: content}
	var resp struct {
		Mermaid string `json:"mermaid"`
	}
	if err := c.postGateway(ctx, token, "/v1/sync/diagrams/to-mermaid", req, &resp); err != nil {
		return "", err
	}
	return resp.Mermaid, nil
}

func (c *Client) ResolveAssetURL(ctx context.Context, token, orgID, assetID string) (string, error) {
	var resp struct {
		URLs map[string]string `json:"urls"`
	}
	if err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/assets/urls?ids=%s", orgID, assetID), &resp); err != nil {
		return "", err
	}
	return resp.URLs[assetID], nil
}
