package ui

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m Model) View() tea.View {
	v := tea.NewView(m.viewString())
	v.AltScreen = true
	return v
}

func (m Model) viewString() string {
	if m.snap == nil && m.err == nil {
		return "\n  " + lipgloss.NewStyle().Foreground(colorPurple).Render("Collecting metrics…") +
			"\n\n  Press q to quit"
	}
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press q to quit", m.err)
	}

	w := m.width
	if w < 60 {
		w = 60
	}

	parts := []string{m.renderHeader(w), m.renderCPU(w)}
	if gpu := m.renderGPU(w); gpu != "" {
		parts = append(parts, gpu)
	}
	parts = append(parts,
		m.renderMemory(w),
		m.renderDiskNet(w),
		lipgloss.NewStyle().Foreground(colorGray).Render("  q to quit"),
	)
	return strings.Join(parts, "\n")
}

// LIPGLOSS THEME
func (m Model) renderHeader(w int) string {
	hostname, _ := os.Hostname()
	logo := lipgloss.NewStyle().Bold(true).Foreground(colorPurple).Render(" atop")
	rightStr := fmt.Sprintf("%s  %s ", hostname, time.Now().Format("15:04:05"))
	ts := lipgloss.NewStyle().Foreground(colorGray).Render(rightStr)
	gap := max(0, w-5-len(rightStr))
	return logo + strings.Repeat(" ", gap) + ts
}

// LIPGLOSS THEME
func (m Model) renderCPU(w int) string {
	s := m.snap
	innerW := w - 4

	label := "CPU"
	if s.CPUModel != "" {
		label = "CPU: " + truncate(s.CPUModel, innerW/2)
	}
	loadStr := fmt.Sprintf("Load: %.2f  %.2f  %.2f", s.Load1, s.Load5, s.Load15)
	gap := max(0, innerW-len(label)-len(loadStr))
	header := sectionTitle(label) + strings.Repeat(" ", gap) + stSecondary.Render(loadStr)

	content := header + "\n" +
		"  " + stLabel.Render("Total:    ") + m.cpuBar.ViewAs(s.CPUTotal/100) + "  " + styledPct(s.CPUTotal)

	// Apple Silicon P-core / E-core breakdown
	if s.IsAppleSi && s.NumPCores > 0 && s.NumECores > 0 && len(s.CPUCores) > 0 {
		n := len(s.CPUCores)
		pEnd := min(s.NumPCores, n)
		eStart := pEnd
		eEnd := min(eStart+s.NumECores, n)

		pAvg := avgSlice(s.CPUCores[:pEnd])
		eAvg := avgSlice(s.CPUCores[eStart:eEnd])

		content += "\n" + "  " + stLabel.Render("P-Core:   ") + m.cpuPBar.ViewAs(pAvg/100) + "  " + styledPct(pAvg)
		content += "\n" + "  " + stLabel.Render("E-Core:   ") + m.cpuEBar.ViewAs(eAvg/100) + "  " + styledPct(eAvg)
	}

	// Per-core grid
	if len(s.CPUCores) > 0 {
		content += "\n"
		for _, line := range m.coreGrid(innerW) {
			content += "\n" + line
		}
	}

	return panelStyle.Width(w).Render(content)
}

// LIPGLOSS THEME
// coreGrid uses per-core progress bars so each cell follows the green→yellow→red gradient.
func (m Model) coreGrid(innerW int) []string {
	const cellW = 18
	cols := max(1, innerW/cellW)

	var lines []string
	var row strings.Builder
	for i, p := range m.snap.CPUCores {
		var bar string
		if i < len(m.coreBars) {
			bar = m.coreBars[i].ViewAs(p / 100)
		} else {
			bar = renderBar(p, 6)
		}
		cell := fmt.Sprintf(" %s %s %4.0f%%",
			stMuted.Render(fmt.Sprintf("C%02d:", i)), bar, p)
		row.WriteString(cell)
		if (i+1)%cols == 0 || i == len(m.snap.CPUCores)-1 {
			lines = append(lines, row.String())
			row.Reset()
		}
	}
	return lines
}

// LIPGLOSS THEME
func (m Model) renderGPU(w int) string {
	s := m.snap
	if len(s.GPUs) == 0 {
		return ""
	}

	var content string
	for i, g := range s.GPUs {
		label := "GPU: " + g.Name
		if len(s.GPUs) > 1 {
			label = fmt.Sprintf("GPU %d: %s", i+1, g.Name)
		}
		if i == 0 {
			content = sectionTitle(label)
		} else {
			content += "\n" + sectionTitle(label)
		}
		devBar, renBar, tilBar := renderBar(g.DeviceUtil, 6), renderBar(g.RendererUtil, 6), renderBar(g.TilerUtil, 6)
		if i < len(m.gpuDevBars) {
			devBar = m.gpuDevBars[i].ViewAs(g.DeviceUtil / 100)
			renBar = m.gpuRenBars[i].ViewAs(g.RendererUtil / 100)
			tilBar = m.gpuTilBars[i].ViewAs(g.TilerUtil / 100)
		}
		content += "\n" + "  " + stLabel.Render("Device:   ") + devBar + "  " + styledPct(g.DeviceUtil)
		content += "\n" + "  " + stLabel.Render("Renderer: ") + renBar + "  " + styledPct(g.RendererUtil)
		content += "\n" + "  " + stLabel.Render("Tiler:    ") + tilBar + "  " + styledPct(g.TilerUtil)
		if g.MemInUse > 0 {
			content += "\n" + "  " + stLabel.Render("VRAM:     ") + styledSize(g.MemInUse) + stMuted.Render(" in use")
		}
	}

	return panelStyle.Width(w).Render(content)
}

// LIPGLOSS THEME
func (m Model) renderMemory(w int) string {
	s := m.snap

	ramLine := "  " + stLabel.Render("RAM:      ") + m.ramBar.ViewAs(s.MemPercent/100) +
		"  " + styledSize(s.MemUsed) + stMuted.Render(" / ") + styledSize(s.MemTotal) +
		"  " + styledPct(s.MemPercent)
	swapLine := "  " + stLabel.Render("Swap:     ") + m.swapBar.ViewAs(s.SwapPercent/100) +
		"  " + styledSize(s.SwapUsed) + stMuted.Render(" / ") + styledSize(s.SwapTotal) +
		"  " + styledPct(s.SwapPercent)

	content := sectionTitle("Memory") + "\n" + ramLine + "\n" + swapLine
	return panelStyle.Width(w).Render(content)
}

// LIPGLOSS THEME
func (m Model) renderDiskNet(w int) string {
	leftOuter := (w - 1) / 2
	rightOuter := w - 1 - leftOuter
	leftInner := leftOuter - 4
	rightInner := rightOuter - 4

	return lipgloss.JoinHorizontal(lipgloss.Top,
		m.renderDisk(leftInner),
		" ",
		m.renderNet(rightInner),
	)
}

// LIPGLOSS THEME
func (m Model) renderDisk(innerW int) string {
	s := m.snap
	maxDisk := math.Max(s.DiskReadPS, s.DiskWritePS)
	content := sectionTitle("Disk I/O") + "\n" +
		"  " + stNetLabel.Render("Read:     ") + m.diskReadBar.ViewAs(relativePct(s.DiskReadPS, maxDisk)/100) + "  " + styledRate(s.DiskReadPS) + "\n" +
		"  " + stNetLabel.Render("Write:    ") + m.diskWriteBar.ViewAs(relativePct(s.DiskWritePS, maxDisk)/100) + "  " + styledRate(s.DiskWritePS)
	return panelStyle.Width(innerW + 4).Render(content)
}

// LIPGLOSS THEME
func (m Model) renderNet(innerW int) string {
	s := m.snap
	maxNet := math.Max(s.NetUpPS, s.NetDownPS)
	content := sectionTitle("Network") + "\n" +
		"  " + stNetLabel.Render("↑ Up:     ") + m.netUpBar.ViewAs(relativePct(s.NetUpPS, maxNet)/100) + "  " + styledRate(s.NetUpPS) + "\n" +
		"  " + stNetLabel.Render("↓ Down:   ") + m.netDownBar.ViewAs(relativePct(s.NetDownPS, maxNet)/100) + "  " + styledRate(s.NetDownPS)
	return panelStyle.Width(innerW + 4).Render(content)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func styledPct(pct float64) string {
	return stValue.Render(fmt.Sprintf("%5.1f", pct)) + stUnit.Render("%")
}

func styledSize(b uint64) string {
	const (
		gb = 1 << 30
		mb = 1 << 20
		kb = 1 << 10
	)
	switch {
	case b >= gb:
		return stValue.Render(fmt.Sprintf("%.1f", float64(b)/gb)) + stUnit.Render("GB")
	case b >= mb:
		return stValue.Render(fmt.Sprintf("%.0f", float64(b)/mb)) + stUnit.Render("MB")
	case b >= kb:
		return stValue.Render(fmt.Sprintf("%.0f", float64(b)/kb)) + stUnit.Render("KB")
	default:
		return stValue.Render(fmt.Sprintf("%d", b)) + stUnit.Render("B")
	}
}

func styledRate(bps float64) string {
	const (
		gb = 1 << 30
		mb = 1 << 20
		kb = 1 << 10
	)
	switch {
	case bps >= gb:
		return stValue.Render(fmt.Sprintf("%.1f", bps/gb)) + stUnit.Render(" GB/s")
	case bps >= mb:
		return stValue.Render(fmt.Sprintf("%.1f", bps/mb)) + stUnit.Render(" MB/s")
	case bps >= kb:
		return stValue.Render(fmt.Sprintf("%.1f", bps/kb)) + stUnit.Render(" KB/s")
	default:
		return stValue.Render(fmt.Sprintf("%.0f", bps)) + stUnit.Render(" B/s")
	}
}

func relativePct(v, maxv float64) float64 {
	if maxv <= 0 {
		return 0
	}
	return v / maxv * 100
}

func avgSlice(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}
