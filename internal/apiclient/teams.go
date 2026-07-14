package apiclient

import (
	"context"
	"fmt"
)

type Team struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email,omitempty"`
	ExternalID  string `json:"externalId,omitempty"`
	MemberCount int    `json:"memberCount"`
}

type TeamMember struct {
	TeamID     string `json:"teamId"`
	UserID     string `json:"userId"`
	Permission string `json:"permission"`
}

func (c *Client) ListTeams(ctx context.Context, token, orgID string) ([]Team, error) {
	var resp struct {
		Teams []Team `json:"teams"`
	}
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/teams", orgID), &resp)
	return resp.Teams, err
}

func (c *Client) GetTeam(ctx context.Context, token, orgID, teamID string) (*Team, error) {
	var t Team
	if err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/teams/%s", orgID, teamID), &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (c *Client) ListTeamMembers(ctx context.Context, token, orgID, teamID string) ([]TeamMember, error) {
	var resp struct {
		Members []TeamMember `json:"members"`
	}
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/teams/%s/members", orgID, teamID), &resp)
	return resp.Members, err
}
