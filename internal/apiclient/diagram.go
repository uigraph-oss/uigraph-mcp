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
	return c.getRaw(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/diagrams/%s/content", orgID, diagramID))
}
