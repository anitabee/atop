package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Vivid rainbow palette cycling through the spectrum.
var rainbow = []lipgloss.Color{
	"#FF355E", // radical red
	"#FF6037", // outrageous orange
	"#FFCC33", // sunglow yellow
	"#66FF66", // screamin' green
	"#50BFE6", // blizzard blue
	"#BF5FFF", // purple pizzazz
	"#FF6EFF", // ultra pink
}

// Per-section accent colors drawn from the rainbow.
var (
	colCPU  = lipgloss.Color("#FF355E") // red    – CPU
	colGPU  = lipgloss.Color("#FF6EFF") // pink   – GPU
	colMem  = lipgloss.Color("#66FF66") // green  – memory
	colDisk = lipgloss.Color("#50BFE6") // blue   – disk
	colNet  = lipgloss.Color("#BF5FFF") // violet – network

	colWarn   = lipgloss.Color("#FFCC33") // yellow  – 70–90%
	colDanger = lipgloss.Color("#FF3300") // hot red – 90%+
	colEmpty  = lipgloss.Color("#1C1C2C") // dark bg for empty bar portions

	colText    = lipgloss.Color("#FFFFFF")
	colSubtext = lipgloss.Color("#AAAAAA")

	stDim = lipgloss.NewStyle().Foreground(colSubtext)
)

// rainbowAt returns a colour from the palette by index (wraps around).
func rainbowAt(i int) lipgloss.Color {
	return rainbow[i%len(rainbow)]
}

// rainbowText renders each rune of s in a cycling rainbow colour.
func rainbowText(s string) string {
	var b strings.Builder
	for i, ch := range s {
		b.WriteString(
			lipgloss.NewStyle().
				Foreground(rainbowAt(i)).
				Bold(true).
				Render(string(ch)),
		)
	}
	return b.String()
}

// sectionTitle renders a title in the section's accent colour.
func sectionTitle(text string, col lipgloss.Color) string {
	return lipgloss.NewStyle().Bold(true).Foreground(col).Render(text)
}

// sectionBox returns a rounded box whose border matches the section colour.
func sectionBox(col lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(col).
		Padding(0, 1)
}

// barColor returns the section colour at normal usage, yellow at ≥70%, red at ≥90%.
func barColor(pct float64, normal lipgloss.Color) lipgloss.Color {
	switch {
	case pct >= 90:
		return colDanger
	case pct >= 70:
		return colWarn
	default:
		return normal
	}
}
