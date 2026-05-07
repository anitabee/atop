package ui

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
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

// LIPGLOSS THEME — gradient palette, computed once at startup
var barGradient [101]color.Color

func init() {
	type rgb struct{ r, g, b float64 }
	lerp := func(a, b rgb, t float64) rgb {
		return rgb{a.r + (b.r-a.r)*t, a.g + (b.g-a.g)*t, a.b + (b.b-a.b)*t}
	}
	green  := rgb{0x04, 0xB5, 0x75} // #04B575 — lipgloss neon green
	yellow := rgb{0xED, 0xFF, 0x82} // #EDFF82 — lipgloss yellow
	red    := rgb{0xEB, 0x42, 0x68} // #EB4268 — lipgloss hot pink/red
	for i := 0; i <= 100; i++ {
		t := float64(i) / 100.0
		var c rgb
		if t <= 0.5 {
			c = lerp(green, yellow, t*2)
		} else {
			c = lerp(yellow, red, (t-0.5)*2)
		}
		barGradient[i] = lipgloss.Color(fmt.Sprintf("#%02X%02X%02X",
			int(math.Round(c.r)),
			int(math.Round(c.g)),
			int(math.Round(c.b)),
		))
	}
}

func barColorForPct(pct float64) color.Color {
	idx := int(math.Round(pct))
	if idx < 0 {
		idx = 0
	}
	if idx > 100 {
		idx = 100
	}
	return barGradient[idx]
}

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

// renderBar draws a progress bar using the perceptual green→yellow→red gradient.
// Empty segments use colorDarkGray.
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

	c := barColorForPct(pct)
	bar := lipgloss.NewStyle().Foreground(c).Render(strings.Repeat("█", filled))
	bar += lipgloss.NewStyle().Foreground(colorDarkGray).Render(strings.Repeat("░", empty))
	return bar
}
