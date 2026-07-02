package apiclient

import (
	"context"
	"fmt"
	"time"
)

type Map struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Frame struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	TemplateType  string  `json:"templateType"`
	Status        string  `json:"status"`
	ParentFrameID *string `json:"parentFrameId,omitempty"`
	Order         float64 `json:"order"`
}

type Folder struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	ParentID *string `json:"parentId,omitempty"`
	Order    float64 `json:"order"`
}

func (c *Client) ListMaps(ctx context.Context, token, orgID string, folderID, teamID *string) ([]Map, error) {
	var resp struct {
		Maps []Map `json:"maps"`
	}
	return resp.Maps, c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/maps", orgID), &resp)
}

func (c *Client) GetMap(ctx context.Context, token, orgID, mapID string) (*Map, error) {
	var m Map
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/maps/%s", orgID, mapID), &m)
	return &m, err
}

func (c *Client) ListFrames(ctx context.Context, token, orgID, mapID string) ([]Frame, error) {
	var resp struct {
		Frames []Frame `json:"frames"`
	}
	return resp.Frames, c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/maps/%s/frames", orgID, mapID), &resp)
}

func (c *Client) ListFolders(ctx context.Context, token, orgID string, folderType *string) ([]Folder, error) {
	path := fmt.Sprintf("/api/v1/orgs/%s/folders", orgID)
	if folderType != nil {
		path += "?type=" + *folderType
	}
	var resp struct {
		Folders []Folder `json:"folders"`
	}
	return resp.Folders, c.get(ctx, token, path, &resp)
}
