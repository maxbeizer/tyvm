package main

import (
	"strings"
	"testing"
)

func TestSparkline(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		wantSVG  bool
		contains []string
	}{
		{"empty", nil, false, nil},
		{"single", []float64{1.0}, false, nil},
		{"flat", []float64{2, 2, 2}, true, []string{"<svg", "polyline", "0.0,24.0"}},
		{"rising", []float64{0, 1, 2}, true, []string{"<svg", `width="80"`, `height="24"`}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sparkline(tc.values, 80, 24)
			if tc.wantSVG && got == "" {
				t.Fatalf("expected SVG, got empty string")
			}
			if !tc.wantSVG && got != "" {
				t.Fatalf("expected empty string, got %q", got)
			}
			for _, s := range tc.contains {
				if !strings.Contains(got, s) {
					t.Errorf("expected output to contain %q, got %q", s, got)
				}
			}
		})
	}
}

func TestSparklineCoordinates(t *testing.T) {
	// Two points: should produce polyline at endpoints with y inverted (max→0, min→height).
	got := sparkline([]float64{0, 10}, 100, 50)
	for _, want := range []string{"0.0,50.0", "100.0,0.0"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected point %q in %q", want, got)
		}
	}
}
