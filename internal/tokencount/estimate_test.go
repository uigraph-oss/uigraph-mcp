package tokencount_test

import (
	"testing"

	"github.com/uigraph/mcp/internal/tokencount"
)

func TestCount(t *testing.T) {
	got := tokencount.Count("hello world") // 11 chars / 4 = 2
	if got != 2 {
		t.Fatalf("Count(%q) = %d, want 2", "hello world", got)
	}
}

func TestRawEquivalent_ExactCount(t *testing.T) {
	exact := 5000
	got := tokencount.RawEquivalent("get_api_spec", 1000, &exact)
	if got != 5000 {
		t.Fatalf("RawEquivalent with exact = %d, want 5000", got)
	}
}

func TestRawEquivalent_Multiplier(t *testing.T) {
	// get_api_spec fallback multiplier is 4.0x
	got := tokencount.RawEquivalent("get_api_spec", 1000, nil)
	if got != 4000 {
		t.Fatalf("RawEquivalent fallback = %d, want 4000", got)
	}
}

func TestRawEquivalent_UnknownTool(t *testing.T) {
	// unknown tool uses 1.5x
	got := tokencount.RawEquivalent("unknown_tool", 1000, nil)
	if got != 1500 {
		t.Fatalf("RawEquivalent unknown = %d, want 1500", got)
	}
}
