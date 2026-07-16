package apiclient

import (
	"context"
	"fmt"
)

type ServiceDependency struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	ProviderName     string   `json:"providerName"`
	Type             string   `json:"type"`
	Criticality      string   `json:"criticality"`
	Description      string   `json:"description"`
	API              *string  `json:"api,omitempty"`
	Operations       []string `json:"operations"`
	OnboardingStatus string   `json:"onboardingStatus"`
	Direction        string   `json:"direction"`
	Consumer         *Service `json:"consumer,omitempty"`
	Provider         *Service `json:"provider,omitempty"`
}

func (c *Client) ListServiceDependencies(ctx context.Context, token, orgID, serviceID string) ([]ServiceDependency, error) {
	var resp struct {
		Edges []ServiceDependency `json:"edges"`
	}
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s/dependencies", orgID, serviceID), &resp)
	return resp.Edges, err
}
