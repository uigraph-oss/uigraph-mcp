package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/uigraph/mcp/internal/apiclient"
)

func (h *Handler) RegisterDependencyTools(s *mcpserver.MCPServer) {
	s.AddTool(mcp.NewTool(
		"get_dependency_graph",
		mcp.WithDescription("Get the organisation-wide service dependency graph as a Mermaid diagram."),
	), h.getDependencyGraph)
}

func (h *Handler) getDependencyGraph(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	edges, err := h.client.DependencyGraph(ctx, token, orgID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(dependencyMermaid(edges)), nil
}

func dependencyMermaid(edges []apiclient.ServiceDependency) string {
	consumerID := func(edge apiclient.ServiceDependency) string {
		if edge.Consumer != nil {
			return edge.Consumer.ID
		}
		return edge.ID
	}
	providerID := func(edge apiclient.ServiceDependency) string {
		if edge.Provider != nil {
			return edge.Provider.ID
		}
		return "ghost:" + edge.ProviderName
	}

	labels := map[string]string{}
	for _, edge := range edges {
		if edge.Consumer != nil {
			labels[consumerID(edge)] = edge.Consumer.Name
		} else if _, ok := labels[consumerID(edge)]; !ok {
			labels[consumerID(edge)] = consumerID(edge)
		}
		if edge.Provider != nil {
			labels[providerID(edge)] = edge.Provider.Name
		} else if _, ok := labels[providerID(edge)]; !ok {
			labels[providerID(edge)] = edge.ProviderName
		}
	}

	ids := make([]string, 0, len(labels))
	for id := range labels {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	mermaidID := map[string]string{}
	for i, id := range ids {
		mermaidID[id] = fmt.Sprintf("n%d", i)
	}

	var sb strings.Builder
	sb.WriteString("flowchart LR\n")
	for _, id := range ids {
		sb.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", mermaidID[id], escapeMermaidLabel(labels[id])))
	}

	sorted := make([]apiclient.ServiceDependency, len(edges))
	copy(sorted, edges)
	sort.Slice(sorted, func(i, j int) bool {
		if consumerID(sorted[i]) != consumerID(sorted[j]) {
			return consumerID(sorted[i]) < consumerID(sorted[j])
		}
		if providerID(sorted[i]) != providerID(sorted[j]) {
			return providerID(sorted[i]) < providerID(sorted[j])
		}
		return sorted[i].Name < sorted[j].Name
	})

	for _, edge := range sorted {
		label := dependencyEdgeLabel(edge)
		if label == "" {
			sb.WriteString(fmt.Sprintf("    %s --> %s\n", mermaidID[consumerID(edge)], mermaidID[providerID(edge)]))
			continue
		}
		sb.WriteString(fmt.Sprintf("    %s -->|%s| %s\n", mermaidID[consumerID(edge)], escapeMermaidLabel(label), mermaidID[providerID(edge)]))
	}

	return sb.String()
}

func dependencyEdgeLabel(edge apiclient.ServiceDependency) string {
	parts := []string{}
	if edge.Criticality != "" {
		parts = append(parts, edge.Criticality)
	}
	if edge.Type != "" {
		parts = append(parts, edge.Type)
	}
	return strings.Join(parts, " · ")
}

func escapeMermaidLabel(label string) string {
	return strings.ReplaceAll(label, "\"", "#quot;")
}
