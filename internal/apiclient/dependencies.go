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
	APIGroupName     *string  `json:"apiGroupName,omitempty"`
	APIEndpointNames []string `json:"apiEndpointNames"`
	DatabaseName     *string  `json:"databaseName,omitempty"`
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

func (c *Client) DependencyGraph(ctx context.Context, token, orgID string) ([]ServiceDependency, error) {
	var resp struct {
		Edges []ServiceDependency `json:"edges"`
	}
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/dependency-graph", orgID), &resp)
	return resp.Edges, err
}
