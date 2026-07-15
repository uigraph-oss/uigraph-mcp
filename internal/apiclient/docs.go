package apiclient

import (
	"context"
	"fmt"
	"net/url"
)

type Doc struct {
	ID            string  `json:"id"`
	FileName      string  `json:"fileName"`
	FileType      string  `json:"fileType"`
	Description   string  `json:"description"`
	DocTokenCount int     `json:"docTokenCount"`
	FolderID      *string `json:"folderId,omitempty"`
	TeamID        *string `json:"teamId,omitempty"`
}

func (c *Client) ListDocs(ctx context.Context, token, orgID string, search *string) ([]Doc, error) {
	path := fmt.Sprintf("/api/v1/orgs/%s/docs", orgID)
	q := url.Values{}
	if search != nil {
		q.Set("search", *search)
	}
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var resp struct {
		Docs []Doc `json:"docs"`
	}
	return resp.Docs, c.get(ctx, token, path, &resp)
}

func (c *Client) GetDoc(ctx context.Context, token, orgID, docID string) (*Doc, error) {
	var d Doc
	if err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/docs/%s", orgID, docID), &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (c *Client) GetDocContent(ctx context.Context, token, orgID, docID string) ([]byte, error) {
	return c.getRaw(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/docs/%s/content", orgID, docID))
}
