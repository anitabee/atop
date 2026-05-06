package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// LIPGLOSS THEME — every color comes from the official lipgloss README/examples.
var (
	colorPurple    = lipgloss.Color("#7D56F4") // primary accent / headers
	colorGreen     = lipgloss.Color("#04B575") // healthy / normal values
	colorDarkGray  = lipgloss.Color("#3C3C3C") // muted / background fill
	colorPink      = lipgloss.Color("#EB4268") // warning / high utilization
	colorWhite     = lipgloss.Color("#FAFAFA") // primary text
	colorYellow    = lipgloss.Color("#EDFF82") // mid-range gradient color
	colorLightPurp = lipgloss.Color("#C5ADF9") // soft accent / secondary
	colorTeal      = lipgloss.Color("#37CD96") // alternate green (adaptive)
	colorViolet    = lipgloss.Color("#864EFF") // alternate purple (adaptive)

	// ANSI 256 colors from the lipgloss README:
	colorAqua      = lipgloss.Color("86")  // #5FD7D7 — secondary info / load
	colorBorder    = lipgloss.Color("63")  // #5F5FAF — panel borders
	colorItemLabel = lipgloss.Color("212") // #FF87D7 — disk / network labels
	colorPurple256 = lipgloss.Color("99")  // headers in table example
	colorGray      = lipgloss.Color("245") // secondary text / units
	colorLightGray = lipgloss.Color("241") // tertiary text
)

// Reusable style shortcuts.
var (
	stLabel    = lipgloss.NewStyle().Foreground(colorWhite)
	stValue    = lipgloss.NewStyle().Bold(true).Foreground(colorWhite)
	stUnit     = lipgloss.NewStyle().Foreground(colorGray)
	stSecondary = lipgloss.NewStyle().Foreground(colorAqua)
	stNetLabel = lipgloss.NewStyle().Foreground(colorItemLabel)
	stMuted    = lipgloss.NewStyle().Foreground(colorDarkGray)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)
)

// sectionTitle renders a bold purple section heading.
func sectionTitle(text string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(colorPurple).Render(text)
}

// LIPGLOSS THEME
// renderBar draws a progress bar using only the canonical lipgloss palette.
//   0–60%  → colorGreen
//  60–85%  → colorYellow
//  85–100% → colorPink
//  empty   → colorDarkGray
func renderBar(pct float64, width int) string {
	if width <= 0 {
		return ""
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(pct / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled

	var barColor lipgloss.Color
	switch {
	case pct >= 85:
		barColor = colorPink
	case pct >= 60:
		barColor = colorYellow
	default:
		barColor = colorGreen
	}

	bar := lipgloss.NewStyle().Foreground(barColor).Render(strings.Repeat("█", filled))
	bar += lipgloss.NewStyle().Foreground(colorDarkGray).Render(strings.Repeat("░", empty))
	return bar
}
