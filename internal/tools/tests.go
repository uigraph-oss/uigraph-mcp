package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func (h *Handler) RegisterTestTools(s *mcpserver.MCPServer) {
	s.AddTool(mcp.NewTool("list_test_packs",
		mcp.WithDescription("List test packs for a service"),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
	), h.listTestPacks)

	s.AddTool(mcp.NewTool("get_test_pack",
		mcp.WithDescription("Get a single test pack by ID"),
		mcp.WithString("test_pack_id", mcp.Required(), mcp.Description("Test pack ID")),
	), h.getTestPack)

	s.AddTool(mcp.NewTool("list_test_cases",
		mcp.WithDescription("List test cases for a service, optionally filtered by test pack"),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
		mcp.WithString("test_pack_id", mcp.Description("Optional test pack ID filter")),
	), h.listTestCases)

	s.AddTool(mcp.NewTool("list_test_runs",
		mcp.WithDescription("List test runs for a service, optionally filtered by test pack"),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
		mcp.WithString("test_pack_id", mcp.Description("Optional test pack ID filter")),
	), h.listTestRuns)

	s.AddTool(mcp.NewTool("get_test_run",
		mcp.WithDescription("Get a single test run by ID"),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
		mcp.WithString("test_run_id", mcp.Required(), mcp.Description("Test run ID")),
	), h.getTestRun)

	s.AddTool(mcp.NewTool("list_test_runs_summary",
		mcp.WithDescription("List test runs for a service with aggregated pass/fail/skip/block counts"),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
		mcp.WithString("test_pack_id", mcp.Description("Optional test pack ID filter")),
		mcp.WithString("environment", mcp.Description("Optional environment filter")),
		mcp.WithString("status", mcp.Description("Optional status filter")),
	), h.listTestRunsSummary)

	s.AddTool(mcp.NewTool("list_test_run_results",
		mcp.WithDescription("List per-test-case results for a test run"),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
		mcp.WithString("test_run_id", mcp.Required(), mcp.Description("Test run ID")),
	), h.listTestRunResults)
}

func (h *Handler) listTestPacks(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	serviceID, err := req.RequireString("service_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	packs, err := h.client.ListTestPacks(ctx, token, orgID, serviceID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# Test packs\n\n")
	for _, p := range packs {
		sb.WriteString(fmt.Sprintf("- **TestPackID:** `%s`\n", p.ID))
		sb.WriteString(fmt.Sprintf("  - **Name:** %s\n", p.Name))
		sb.WriteString(fmt.Sprintf("  - **Type:** %s\n", p.Type))
		sb.WriteString("\n")
	}
	if len(packs) == 0 {
		sb.WriteString("No test packs found.\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) getTestPack(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	testPackID, err := req.RequireString("test_pack_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	p, err := h.client.GetTestPack(ctx, token, orgID, testPackID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("- **TestPackID:** `%s`\n", p.ID))
	sb.WriteString(fmt.Sprintf("- **ServiceID:** `%s`\n", p.ServiceID))
	sb.WriteString(fmt.Sprintf("- **Name:** %s\n", p.Name))
	sb.WriteString(fmt.Sprintf("- **Type:** %s\n", p.Type))
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) listTestCases(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	serviceID, err := req.RequireString("service_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	var testPackID *string
	if v := req.GetString("test_pack_id", ""); v != "" {
		testPackID = &v
	}

	cases, err := h.client.ListTestCases(ctx, token, orgID, serviceID, testPackID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# Test cases\n\n")
	for _, c := range cases {
		sb.WriteString(fmt.Sprintf("- **TestCaseID:** `%s`\n", c.ID))
		sb.WriteString(fmt.Sprintf("  - **Title:** %s\n", c.Title))
		sb.WriteString(fmt.Sprintf("  - **Type:** %s\n", c.Type))
		sb.WriteString(fmt.Sprintf("  - **Status:** %s\n", c.Status))
		sb.WriteString(fmt.Sprintf("  - **TestPackID:** `%s`\n", c.TestPackID))
		if c.Priority != nil {
			sb.WriteString(fmt.Sprintf("  - **Priority:** %s\n", *c.Priority))
		}
		if c.IsCritical {
			sb.WriteString("  - **Critical:** yes\n")
		}
		if len(c.Labels) > 0 {
			sb.WriteString(fmt.Sprintf("  - **Labels:** %s\n", strings.Join(c.Labels, ", ")))
		}
		sb.WriteString("\n")
	}
	if len(cases) == 0 {
		sb.WriteString("No test cases found.\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) listTestRuns(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	serviceID, err := req.RequireString("service_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	var testPackID *string
	if v := req.GetString("test_pack_id", ""); v != "" {
		testPackID = &v
	}

	runs, err := h.client.ListTestRuns(ctx, token, orgID, serviceID, testPackID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# Test runs\n\n")
	for _, r := range runs {
		sb.WriteString(fmt.Sprintf("- **TestRunID:** `%s`\n", r.ID))
		sb.WriteString(fmt.Sprintf("  - **TestPackID:** `%s`\n", r.TestPackID))
		sb.WriteString(fmt.Sprintf("  - **Environment:** %s\n", r.Environment))
		sb.WriteString(fmt.Sprintf("  - **Status:** %s\n", r.Status))
		sb.WriteString(fmt.Sprintf("  - **OverallStatus:** %s\n", r.OverallStatus))
		if r.ReleaseLabel != nil {
			sb.WriteString(fmt.Sprintf("  - **ReleaseLabel:** %s\n", *r.ReleaseLabel))
		}
		sb.WriteString("\n")
	}
	if len(runs) == 0 {
		sb.WriteString("No test runs found.\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) getTestRun(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	serviceID, err := req.RequireString("service_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	testRunID, err := req.RequireString("test_run_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	r, err := h.client.GetTestRun(ctx, token, orgID, serviceID, testRunID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("- **TestRunID:** `%s`\n", r.ID))
	sb.WriteString(fmt.Sprintf("- **TestPackID:** `%s`\n", r.TestPackID))
	sb.WriteString(fmt.Sprintf("- **Environment:** %s\n", r.Environment))
	sb.WriteString(fmt.Sprintf("- **Status:** %s\n", r.Status))
	sb.WriteString(fmt.Sprintf("- **OverallStatus:** %s\n", r.OverallStatus))
	if r.ReleaseLabel != nil {
		sb.WriteString(fmt.Sprintf("- **ReleaseLabel:** %s\n", *r.ReleaseLabel))
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) listTestRunsSummary(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	serviceID, err := req.RequireString("service_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	var testPackID, environment, status *string
	if v := req.GetString("test_pack_id", ""); v != "" {
		testPackID = &v
	}
	if v := req.GetString("environment", ""); v != "" {
		environment = &v
	}
	if v := req.GetString("status", ""); v != "" {
		status = &v
	}

	summaries, err := h.client.ListTestRunsSummary(ctx, token, orgID, serviceID, testPackID, environment, status)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# Test runs summary\n\n")
	for _, s := range summaries {
		sb.WriteString(fmt.Sprintf("- **TestRunID:** `%s`\n", s.TestRunID))
		sb.WriteString(fmt.Sprintf("  - **Environment:** %s\n", s.Environment))
		sb.WriteString(fmt.Sprintf("  - **Status:** %s\n", s.Status))
		sb.WriteString(fmt.Sprintf("  - **OverallStatus:** %s\n", s.OverallStatus))
		sb.WriteString(fmt.Sprintf("  - **Passed:** %d, **Failed:** %d, **Skipped:** %d, **Blocked:** %d\n",
			s.PassedCount, s.FailedCount, s.SkippedCount, s.BlockedCount))
		sb.WriteString("\n")
	}
	if len(summaries) == 0 {
		sb.WriteString("No test runs found.\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) listTestRunResults(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	serviceID, err := req.RequireString("service_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	testRunID, err := req.RequireString("test_run_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	results, err := h.client.ListTestRunResults(ctx, token, orgID, serviceID, testRunID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# Test run results\n\n")
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("- **TestRunResultID:** `%s`\n", r.ID))
		sb.WriteString(fmt.Sprintf("  - **TestCaseID:** `%s`\n", r.TestCaseID))
		sb.WriteString(fmt.Sprintf("  - **Status:** %s\n", r.Status))
		if r.ResponseStatus != nil {
			sb.WriteString(fmt.Sprintf("  - **ResponseStatus:** %d\n", *r.ResponseStatus))
		}
		if r.ResponseTimeMs != nil {
			sb.WriteString(fmt.Sprintf("  - **ResponseTimeMs:** %d\n", *r.ResponseTimeMs))
		}
		if r.BlockedReason != nil {
			sb.WriteString(fmt.Sprintf("  - **BlockedReason:** %s\n", *r.BlockedReason))
		}
		if r.Notes != nil {
			sb.WriteString(fmt.Sprintf("  - **Notes:** %s\n", *r.Notes))
		}
		sb.WriteString("\n")
	}
	if len(results) == 0 {
		sb.WriteString("No test run results found.\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}
