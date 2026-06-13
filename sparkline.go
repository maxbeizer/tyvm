package main

import (
	"fmt"
	"strings"
)

// sparkline renders the given values as an inline SVG polyline. Returns an
// empty string when fewer than two points are supplied.
func sparkline(values []float64, width, height int) string {
	if len(values) < 2 {
		return ""
	}
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	if max == min {
		max = min + 1
	}

	points := make([]string, 0, len(values))
	for i, v := range values {
		x := float64(i) / float64(len(values)-1) * float64(width)
		y := float64(height) - (v-min)/(max-min)*float64(height)
		points = append(points, fmt.Sprintf("%.1f,%.1f", x, y))
	}

	return fmt.Sprintf(
		`<svg width="%d" height="%d" viewBox="0 0 %d %d" class="sparkline"><polyline points="%s" fill="none" stroke="#0e7490" stroke-width="1.5"/></svg>`,
		width, height, width, height, strings.Join(points, " "),
	)
}
