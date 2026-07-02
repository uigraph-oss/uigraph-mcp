package apiclient

import (
	"context"
	"fmt"
	"time"
)

type Service struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Tier        string    `json:"tier"`
	Category    string    `json:"category"`
	Language    string    `json:"language"`
	Labels      []string  `json:"labels"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ServiceStats struct {
	ServiceID     string `json:"serviceId"`
	EndpointCount int    `json:"endpointCount"`
	DiagramCount  int    `json:"diagramCount"`
	DocCount      int    `json:"docCount"`
	DBTableCount  int    `json:"dbTableCount"`
	TestCaseCount int    `json:"testCaseCount"`
}

type APIGroup struct {
	ID        string    `json:"id"`
	ServiceID string    `json:"serviceId"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Protocol  string    `json:"protocol"`
	Label     *string   `json:"label,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type APIEndpoint struct {
	ID          string   `json:"id"`
	OperationID string   `json:"operationId"`
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	Summary     string   `json:"summary"`
	Tags        []string `json:"tags"`
	TokenCount  int      `json:"tokenCount"`
}

type ServiceDoc struct {
	ID            string `json:"id"`
	FileName      string `json:"fileName"`
	FileType      string `json:"fileType"`
	Description   string `json:"description"`
	DocTokenCount int    `json:"docTokenCount"`
}

type ServiceDB struct {
	ID               string    `json:"id"`
	DBName           string    `json:"dbName"`
	DBType           string    `json:"dbType"`
	Dialect          string    `json:"dialect"`
	SchemaTokenCount int       `json:"schemaTokenCount"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type ServiceDiagram struct {
	ServiceID string `json:"serviceId"`
	DiagramID string `json:"diagramId"`
}

func (c *Client) ListServices(ctx context.Context, token, orgID string, folderID, teamID *string) ([]Service, error) {
	path := fmt.Sprintf("/api/v1/orgs/%s/services", orgID)
	if folderID != nil {
		path += "?folderId=" + *folderID
	}
	var resp struct {
		Services []Service `json:"services"`
	}
	return resp.Services, c.get(ctx, token, path, &resp)
}

func (c *Client) GetService(ctx context.Context, token, orgID, serviceID string) (*Service, error) {
	var svc Service
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s", orgID, serviceID), &svc)
	return &svc, err
}

func (c *Client) ListServiceStats(ctx context.Context, token, orgID string) ([]ServiceStats, error) {
	var resp struct {
		Stats []ServiceStats `json:"stats"`
	}
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/stats", orgID), &resp)
	return resp.Stats, err
}

func (c *Client) ListAPIGroups(ctx context.Context, token, orgID, serviceID string) ([]APIGroup, error) {
	var resp struct {
		APIGroups []APIGroup `json:"apiGroups"`
	}
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s/api-groups", orgID, serviceID), &resp)
	return resp.APIGroups, err
}

func (c *Client) GetAPIGroupSpec(ctx context.Context, token, orgID, serviceID, apiGroupID string) ([]byte, error) {
	return c.getRaw(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s/api-groups/%s/spec", orgID, serviceID, apiGroupID))
}

func (c *Client) ListAPIEndpoints(ctx context.Context, token, orgID, serviceID, apiGroupID string) ([]APIEndpoint, error) {
	var resp struct {
		Endpoints []APIEndpoint `json:"endpoints"`
	}
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s/api-groups/%s/endpoints", orgID, serviceID, apiGroupID), &resp)
	return resp.Endpoints, err
}

func (c *Client) ListServiceDocs(ctx context.Context, token, orgID, serviceID string) ([]ServiceDoc, error) {
	var resp struct {
		Docs []ServiceDoc `json:"docs"`
	}
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s/docs", orgID, serviceID), &resp)
	return resp.Docs, err
}

func (c *Client) ListServiceDBs(ctx context.Context, token, orgID, serviceID string) ([]ServiceDB, error) {
	var resp struct {
		DBs []ServiceDB `json:"dbs"`
	}
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s/dbs", orgID, serviceID), &resp)
	return resp.DBs, err
}

func (c *Client) GetServiceDB(ctx context.Context, token, orgID, serviceID, dbID string) (*ServiceDB, error) {
	var db ServiceDB
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s/dbs/%s", orgID, serviceID, dbID), &db)
	return &db, err
}

func (c *Client) GetServiceDBSchema(ctx context.Context, token, orgID, serviceID, dbID string) ([]byte, error) {
	return c.getRaw(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s/dbs/%s", orgID, serviceID, dbID))
}

func (c *Client) ListServiceDiagrams(ctx context.Context, token, orgID, serviceID string) ([]ServiceDiagram, error) {
	var resp struct {
		Diagrams []ServiceDiagram `json:"diagrams"`
	}
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s/diagrams", orgID, serviceID), &resp)
	return resp.Diagrams, err
}
