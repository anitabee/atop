package ui

import (
	"strings"
	"testing"
)

func TestRelativePct(t *testing.T) {
	tests := []struct {
		v, maxv, want float64
	}{
		{0, 100, 0},
		{50, 100, 50},
		{100, 100, 100},
		{30, 60, 50},
		{0, 0, 0},
		{50, 0, 0},
	}
	for _, tc := range tests {
		got := relativePct(tc.v, tc.maxv)
		if got != tc.want {
			t.Errorf("relativePct(%v, %v) = %v, want %v", tc.v, tc.maxv, got, tc.want)
		}
	}
}

func TestAvgSlice(t *testing.T) {
	tests := []struct {
		vals []float64
		want float64
	}{
		{nil, 0},
		{[]float64{}, 0},
		{[]float64{100}, 100},
		{[]float64{25, 75}, 50},
		{[]float64{10, 20, 30}, 20},
	}
	for _, tc := range tests {
		got := avgSlice(tc.vals)
		if got != tc.want {
			t.Errorf("avgSlice(%v) = %v, want %v", tc.vals, got, tc.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello w…"},
		{"ab", 2, "ab"},
		{"abc", 2, "a…"},
	}
	for _, tc := range tests {
		got := truncate(tc.s, tc.maxLen)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.s, tc.maxLen, got, tc.want)
		}
	}
}

func TestRenderBarDoesNotPanic(t *testing.T) {
	inputs := []struct {
		pct   float64
		width int
	}{
		{0, 20},
		{50, 20},
		{100, 20},
		{-10, 20},  // clamped to 0
		{150, 20},  // clamped to 100
		{50, 0},    // zero width returns ""
		{50, 1},
	}
	for _, tc := range inputs {
		_ = renderBar(tc.pct, tc.width)
	}
}

func TestRenderBarZeroWidth(t *testing.T) {
	if got := renderBar(50, 0); got != "" {
		t.Errorf("renderBar with zero width should return empty string, got %q", got)
	}
}

func TestRenderBarContainsFilledAndEmpty(t *testing.T) {
	bar := renderBar(50, 10)
	// Strip ANSI escape codes (simple approach: check the raw string contains block chars)
	if !strings.Contains(bar, "█") {
		t.Error("renderBar(50, 10): expected filled blocks '█'")
	}
	if !strings.Contains(bar, "░") {
		t.Error("renderBar(50, 10): expected empty blocks '░'")
	}
}

func TestRenderBarFullyFilled(t *testing.T) {
	bar := renderBar(100, 10)
	if strings.Contains(bar, "░") {
		t.Error("renderBar(100, 10): expected no empty blocks '░'")
	}
}

func TestRenderBarEmpty(t *testing.T) {
	bar := renderBar(0, 10)
	if strings.Contains(bar, "█") {
		t.Error("renderBar(0, 10): expected no filled blocks '█'")
	}
}
