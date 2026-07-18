package tools

import (
	"testing"
	"time"
)

func TestFormatResult(t *testing.T) {
	tests := []struct {
		name   string
		result *Result
	}{
		{
			name: "verified success",
			result: &Result{
				Success:  true,
				Verified: true,
				ExitCode: 0,
				Content:  "wallpaper set",
				Duration: 100 * time.Millisecond,
			},
		},
		{
			name: "success not verified",
			result: &Result{
				Success:  true,
				Verified: false,
				ExitCode: 0,
				Content:  "moved 3 files",
				Duration: 50 * time.Millisecond,
			},
		},
		{
			name: "failure",
			result: &Result{
				Success:  false,
				ExitCode: 1,
				Content:  "permission denied",
				Error:    "permission denied",
				Duration: 20 * time.Millisecond,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := FormatResult(tt.result)
			if len(s) == 0 {
				t.Error("FormatResult returned empty string")
			}
		})
	}
}

func TestStringSet(t *testing.T) {
	s := stringSet([]string{"a", "b", "c"})
	if len(s) != 3 {
		t.Errorf("expected 3 elements, got %d", len(s))
	}
	if !s["b"] {
		t.Error("expected b to be in set")
	}
}

func TestSubtract(t *testing.T) {
	a := stringSet([]string{"a", "b", "c"})
	b := stringSet([]string{"b"})
	result := subtract(a, b)
	if len(result) != 2 {
		t.Errorf("expected 2 elements, got %d", len(result))
	}
	if !result["a"] || !result["c"] {
		t.Error("expected a and c in result")
	}
	if result["b"] {
		t.Error("b should not be in result")
	}
}

func TestStringSetEqual(t *testing.T) {
	a := stringSet([]string{"a", "b"})
	b := stringSet([]string{"a", "b"})
	c := stringSet([]string{"a", "c"})

	if !stringSetEqual(a, b) {
		t.Error("a and b should be equal")
	}
	if stringSetEqual(a, c) {
		t.Error("a and c should not be equal")
	}
}