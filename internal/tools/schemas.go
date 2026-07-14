package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func (h *Handler) RegisterSchemaTools(s *mcpserver.MCPServer) {
	s.AddTool(mcp.NewTool("list_service_dbs",
		mcp.WithDescription("List database schemas attached to a service"),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
	), h.listServiceDBs)

	s.AddTool(mcp.NewTool("get_db_schema",
		mcp.WithDescription("Get the full database schema for a service DB"),
		mcp.WithString("service_id", mcp.Required(), mcp.Description("Service ID")),
		mcp.WithString("db_id", mcp.Required(), mcp.Description("Service DB ID")),
	), h.getDBSchema)
}

func (h *Handler) listServiceDBs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	serviceID, err := req.RequireString("service_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	dbs, err := h.client.ListServiceDBs(ctx, token, orgID, serviceID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("# Database schemas\n\n")
	for _, db := range dbs {
		sb.WriteString(fmt.Sprintf("- **DatabaseID:** `%s`\n", db.ID))
		sb.WriteString(fmt.Sprintf("  - **Name:** %s\n", db.DBName))
		sb.WriteString(fmt.Sprintf("  - **Type:** %s\n", db.DBType))
		sb.WriteString(fmt.Sprintf("  - **Dialect:** %s\n", db.Dialect))
		sb.WriteString(fmt.Sprintf("  - **Tokens:** ~%d\n", db.SchemaTokenCount))
		sb.WriteString("\n")
	}
	if len(dbs) == 0 {
		sb.WriteString("No databases found.\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (h *Handler) getDBSchema(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgID, err := h.orgID(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	serviceID, err := req.RequireString("service_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	dbID, err := req.RequireString("db_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	token := tokenFromCtx(ctx)

	db, err := h.client.GetServiceDB(ctx, token, orgID, serviceID, dbID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	text := fmt.Sprintf("- **DatabaseID:** `%s`\n", db.ID)
	text += fmt.Sprintf("- **Name:** %s\n", db.DBName)
	text += fmt.Sprintf("- **Type:** %s\n", db.DBType)
	text += fmt.Sprintf("- **Dialect:** %s\n", db.Dialect)
	text += formatDBSchema(db.SchemaJSON)

	const maxChars = 50_000
	truncated := false
	if len(text) > maxChars {
		text = text[:maxChars]
		truncated = true
	}

	exactTokens := &db.SchemaTokenCount
	go h.recordUsage(ctx, orgID, token, "get_db_schema", []string{dbID}, text, exactTokens)

	if truncated {
		text += "\n\n[Truncated at 50,000 characters]"
	}
	return mcp.NewToolResultText(text), nil
}

func formatDBSchema(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	data := []byte(raw)
	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		data = []byte(asString)
	}
	var schema struct {
		Tables []struct {
			Name    string `json:"name"`
			Columns []struct {
				Name          string  `json:"name"`
				Type          string  `json:"type"`
				Nullable      *bool   `json:"nullable"`
				IsPrimaryKey  *bool   `json:"isPrimaryKey"`
				Unique        *bool   `json:"unique"`
				AutoIncrement *bool   `json:"autoIncrement"`
				DefaultValue  *string `json:"defaultValue"`
				ForeignKey    *string `json:"foreignKey"`
				Description   *string `json:"description"`
			} `json:"columns"`
			Indexes []struct {
				Name   string   `json:"name"`
				Type   string   `json:"type"`
				Fields []string `json:"fields"`
			} `json:"indexes"`
		} `json:"tables"`
		NoSQLSchema json.RawMessage `json:"noSQLSchema"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		return ""
	}

	var sb strings.Builder

	for _, t := range schema.Tables {
		sb.WriteString(fmt.Sprintf("\n## `%s`\n\n", t.Name))
		sb.WriteString("| Column | Type | Nullable | Key | Default | Description |\n")
		sb.WriteString("| --- | --- | --- | --- | --- | --- |\n")
		for _, c := range t.Columns {
			var keys []string
			if c.IsPrimaryKey != nil && *c.IsPrimaryKey {
				keys = append(keys, "PK")
			}
			if c.Unique != nil && *c.Unique {
				keys = append(keys, "unique")
			}
			if c.AutoIncrement != nil && *c.AutoIncrement {
				keys = append(keys, "auto")
			}
			if c.ForeignKey != nil && *c.ForeignKey != "" {
				keys = append(keys, "FK→"+*c.ForeignKey)
			}
			nullable := ""
			if c.Nullable != nil && *c.Nullable {
				nullable = "yes"
			}
			if c.Nullable != nil && !*c.Nullable {
				nullable = "no"
			}
			def := ""
			if c.DefaultValue != nil {
				def = *c.DefaultValue
			}
			desc := ""
			if c.Description != nil {
				desc = *c.Description
			}
			sb.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s | %s | %s |\n",
				mdCell(c.Name), mdCell(c.Type), nullable,
				mdCell(strings.Join(keys, ", ")), mdCell(def), mdCell(desc)))
		}
		if len(t.Indexes) > 0 {
			sb.WriteString("\n**Indexes:**\n\n")
			for _, idx := range t.Indexes {
				line := fmt.Sprintf("- `%s`", idx.Name)
				if idx.Type != "" {
					line += fmt.Sprintf(" (%s)", idx.Type)
				}
				if len(idx.Fields) > 0 {
					line += ": " + strings.Join(idx.Fields, ", ")
				}
				sb.WriteString(line + "\n")
			}
		}
	}

	nosql := strings.TrimSpace(string(schema.NoSQLSchema))
	if nosql != "" && nosql != "null" && nosql != "{}" {
		sb.WriteString("\n## NoSQL Schema\n\n```json\n")
		sb.WriteString(nosql)
		sb.WriteString("\n```\n")
	}

	return sb.String()
}

func mdCell(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	return s
}
