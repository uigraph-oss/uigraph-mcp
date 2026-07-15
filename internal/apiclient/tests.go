package apiclient

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

type TestPack struct {
	ID        string    `json:"testPackId"`
	ServiceID string    `json:"serviceId"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type TestCase struct {
	ID          string   `json:"testCaseId"`
	TestPackID  string   `json:"testPackId"`
	Title       string   `json:"title"`
	Type        string   `json:"type"`
	Order       float64  `json:"order"`
	Priority    *string  `json:"priority,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Status      string   `json:"status"`
	Version     int      `json:"version"`
	IsCritical  bool     `json:"isCritical"`
	Description *string  `json:"description,omitempty"`
}

type TestRun struct {
	ID            string     `json:"testRunId"`
	TestPackID    string     `json:"testPackId"`
	Environment   string     `json:"environment"`
	ReleaseLabel  *string    `json:"releaseLabel,omitempty"`
	Status        string     `json:"status"`
	OverallStatus string     `json:"overallStatus"`
	StartedAt     *time.Time `json:"startedAt,omitempty"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
	ExecutedBy    string     `json:"executedBy"`
	ExecutedAt    time.Time  `json:"executedAt"`
}

type TestRunSummary struct {
	TestRunID     string    `json:"testRunId"`
	TestPackID    string    `json:"testPackId"`
	Environment   string    `json:"environment"`
	Status        string    `json:"status"`
	OverallStatus string    `json:"overallStatus"`
	ExecutedBy    string    `json:"executedBy"`
	ExecutedAt    time.Time `json:"executedAt"`
	PassedCount   int       `json:"passedCount"`
	FailedCount   int       `json:"failedCount"`
	SkippedCount  int       `json:"skippedCount"`
	BlockedCount  int       `json:"blockedCount"`
}

type TestRunResult struct {
	ID             string  `json:"testRunResultId"`
	TestRunID      string  `json:"testRunId"`
	TestCaseID     string  `json:"testCaseId"`
	Status         string  `json:"status"`
	BlockedReason  *string `json:"blockedReason,omitempty"`
	ResponseStatus *int    `json:"responseStatus,omitempty"`
	ResponseTimeMs *int64  `json:"responseTimeMs,omitempty"`
	Notes          *string `json:"notes,omitempty"`
}

func (c *Client) ListTestPacks(ctx context.Context, token, orgID, serviceID string) ([]TestPack, error) {
	var resp struct {
		TestPacks []TestPack `json:"testPacks"`
	}
	err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s/test-packs", orgID, serviceID), &resp)
	return resp.TestPacks, err
}

func (c *Client) GetTestPack(ctx context.Context, token, orgID, testPackID string) (*TestPack, error) {
	var p TestPack
	if err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/test-packs/%s", orgID, testPackID), &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *Client) ListTestCases(ctx context.Context, token, orgID, serviceID string, testPackID *string) ([]TestCase, error) {
	path := fmt.Sprintf("/api/v1/orgs/%s/services/%s/test-cases", orgID, serviceID)
	if testPackID != nil {
		q := url.Values{}
		q.Set("testPackId", *testPackID)
		path += "?" + q.Encode()
	}
	var resp struct {
		TestCases []TestCase `json:"testCases"`
	}
	err := c.get(ctx, token, path, &resp)
	return resp.TestCases, err
}

func (c *Client) ListTestRuns(ctx context.Context, token, orgID, serviceID string, testPackID *string) ([]TestRun, error) {
	path := fmt.Sprintf("/api/v1/orgs/%s/services/%s/test-runs", orgID, serviceID)
	if testPackID != nil {
		q := url.Values{}
		q.Set("testPackId", *testPackID)
		path += "?" + q.Encode()
	}
	var resp struct {
		TestRuns []TestRun `json:"testRuns"`
	}
	err := c.get(ctx, token, path, &resp)
	return resp.TestRuns, err
}

func (c *Client) GetTestRun(ctx context.Context, token, orgID, serviceID, testRunID string) (*TestRun, error) {
	var tr TestRun
	if err := c.get(ctx, token, fmt.Sprintf("/api/v1/orgs/%s/services/%s/test-run/%s", orgID, serviceID, testRunID), &tr); err != nil {
		return nil, err
	}
	return &tr, nil
}

func (c *Client) ListTestRunsSummary(ctx context.Context, token, orgID, serviceID string, testPackID, environment, status *string) ([]TestRunSummary, error) {
	path := fmt.Sprintf("/api/v1/orgs/%s/services/%s/test-runs-summary", orgID, serviceID)
	q := url.Values{}
	if testPackID != nil {
		q.Set("testPackId", *testPackID)
	}
	if environment != nil {
		q.Set("environment", *environment)
	}
	if status != nil {
		q.Set("status", *status)
	}
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var resp struct {
		TestRunsSummary []TestRunSummary `json:"testRunsSummary"`
	}
	err := c.get(ctx, token, path, &resp)
	return resp.TestRunsSummary, err
}

func (c *Client) ListTestRunResults(ctx context.Context, token, orgID, serviceID, testRunID string) ([]TestRunResult, error) {
	path := fmt.Sprintf("/api/v1/orgs/%s/services/%s/test-run-results", orgID, serviceID)
	q := url.Values{}
	q.Set("testRunId", testRunID)
	path += "?" + q.Encode()
	var resp struct {
		TestRunResults []TestRunResult `json:"testRunResults"`
	}
	err := c.get(ctx, token, path, &resp)
	return resp.TestRunResults, err
}
