package apiclient

import "context"

type Me struct {
	UserID       string `json:"userId"`
	OrgID        string `json:"orgId,omitempty"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	Login        string `json:"login"`
	Kind         string `json:"kind"`
	Role         string `json:"role"`
	AuthProvider string `json:"authProvider"`
}

type Org struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

func (c *Client) GetMe(ctx context.Context, token string) (*Me, error) {
	var me Me
	err := c.get(ctx, token, "/api/v1/auth/me", &me)
	return &me, err
}

func (c *Client) GetMyOrgs(ctx context.Context, token string) ([]Org, error) {
	var resp struct {
		Orgs []Org `json:"orgs"`
	}
	return resp.Orgs, c.get(ctx, token, "/api/v1/auth/orgs", &resp)
}
