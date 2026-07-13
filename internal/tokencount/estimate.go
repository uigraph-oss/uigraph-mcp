package tokencount

// Count estimates the number of LLM tokens in text using the standard
// chars-per-token approximation.
func Count(text string) int {
	return len(text) / 4
}

// multipliers maps tool name → raw-file-read equivalent multiplier.
// When an AI tool reads the equivalent raw repo file, it burns
// (served tokens × multiplier) input tokens.
var multipliers = map[string]float64{
	"get_service_context": 3.0,
	"get_api_spec":        2.5,
	"list_endpoints":      2.0,
	"get_db_schema":       2.25,
	"get_diagram":         1.5,
	"get_map":             1.5,
	"list_services":       1.2,
	"get_service":         1.2,
	"list_api_groups":     1.2,
	"list_service_dbs":    1.2,
	"list_diagrams":       1.2,
	"list_maps":           1.2,
	"list_folders":        1.2,
}

// RawEquivalent returns the estimated tokens an AI would have burned reading
// the equivalent raw repo files. If exactFileTokens is non-nil (recorded at
// CLI sync time), that exact count is used. Otherwise the per-tool multiplier
// is applied to served.
func RawEquivalent(toolName string, served int, exactFileTokens *int) int {
	if exactFileTokens != nil && *exactFileTokens > 0 {
		return *exactFileTokens
	}
	m, ok := multipliers[toolName]
	if !ok {
		m = 1
	}
	return int(float64(served) * m)
}
