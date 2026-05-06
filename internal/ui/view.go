package ui

import (
	"atop/internal/metrics"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
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

	parts := []string{renderHeader(m.snap, w), renderCPU(m.snap, w)}
	if gpu := renderGPU(m.snap, w); gpu != "" {
		parts = append(parts, gpu)
	}
	parts = append(parts,
		renderMemory(m.snap, w),
		renderDiskNet(m.snap, w),
		lipgloss.NewStyle().Foreground(colorGray).Render("  q to quit"),
	)
	return strings.Join(parts, "\n")
}

// LIPGLOSS THEME
func renderHeader(s *metrics.Snapshot, w int) string {
	hostname, _ := os.Hostname()
	logo := lipgloss.NewStyle().Bold(true).Foreground(colorPurple).Render(" atop")
	rightStr := fmt.Sprintf("%s  %s ", hostname, time.Now().Format("15:04:05"))
	ts := lipgloss.NewStyle().Foreground(colorGray).Render(rightStr)
	gap := max(0, w-5-len(rightStr))
	return logo + strings.Repeat(" ", gap) + ts
}

// LIPGLOSS THEME
func renderCPU(s *metrics.Snapshot, w int) string {
	innerW := w - 4

	label := "CPU"
	if s.CPUModel != "" {
		label = "CPU: " + truncate(s.CPUModel, innerW/2)
	}
	loadStr := fmt.Sprintf("Load: %.2f  %.2f  %.2f", s.Load1, s.Load5, s.Load15)
	gap := max(0, innerW-len(label)-len(loadStr))
	header := sectionTitle(label) + strings.Repeat(" ", gap) + stSecondary.Render(loadStr)

	barW := max(6, innerW-18)
	content := header + "\n" +
		"  " + stLabel.Render("Total:  ") + renderBar(s.CPUTotal, barW) + "  " + styledPct(s.CPUTotal)

	// Apple Silicon P-core / E-core breakdown
	if s.IsAppleSi && s.NumPCores > 0 && s.NumECores > 0 && len(s.CPUCores) > 0 {
		n := len(s.CPUCores)
		pEnd := min(s.NumPCores, n)
		eStart := pEnd
		eEnd := min(eStart+s.NumECores, n)

		pAvg := avgSlice(s.CPUCores[:pEnd])
		eAvg := avgSlice(s.CPUCores[eStart:eEnd])

		content += "\n" + "  " + stLabel.Render("P-Core: ") + renderBar(pAvg, barW) + "  " + styledPct(pAvg)
		content += "\n" + "  " + stLabel.Render("E-Core: ") + renderBar(eAvg, barW) + "  " + styledPct(eAvg)
	}

	// Per-core grid
	if len(s.CPUCores) > 0 {
		content += "\n"
		for _, line := range coreGrid(s.CPUCores, innerW) {
			content += "\n" + line
		}
	}

	return panelStyle.Width(innerW).Render(content)
}

// LIPGLOSS THEME
// coreGrid uses renderBar so per-core colours follow the green/yellow/pink thresholds.
func coreGrid(cores []float64, innerW int) []string {
	const cellW = 18
	cols := max(1, innerW/cellW)

	var lines []string
	var row strings.Builder
	for i, p := range cores {
		bar := renderBar(p, 6)
		cell := fmt.Sprintf(" %s %s %4.0f%%",
			stMuted.Render(fmt.Sprintf("C%02d:", i)), bar, p)
		row.WriteString(cell)
		if (i+1)%cols == 0 || i == len(cores)-1 {
			lines = append(lines, row.String())
			row.Reset()
		}
	}
	return lines
}

// LIPGLOSS THEME
func renderGPU(s *metrics.Snapshot, w int) string {
	if len(s.GPUs) == 0 {
		return ""
	}
	innerW := w - 4
	barW := max(6, innerW-18)

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
		content += "\n" + "  " + stLabel.Render("Device:   ") + renderBar(g.DeviceUtil, barW) + "  " + styledPct(g.DeviceUtil)
		content += "\n" + "  " + stLabel.Render("Renderer: ") + renderBar(g.RendererUtil, barW) + "  " + styledPct(g.RendererUtil)
		content += "\n" + "  " + stLabel.Render("Tiler:    ") + renderBar(g.TilerUtil, barW) + "  " + styledPct(g.TilerUtil)
		if g.MemInUse > 0 {
			content += "\n" + "  " + stLabel.Render("VRAM:     ") + styledSize(g.MemInUse) + stMuted.Render(" in use")
		}
	}

	return panelStyle.Width(innerW).Render(content)
}

// LIPGLOSS THEME
func renderMemory(s *metrics.Snapshot, w int) string {
	innerW := w - 4
	barW := max(6, innerW-32)

	ramLine := "  " + stLabel.Render("RAM:  ") + renderBar(s.MemPercent, barW) +
		"  " + styledSize(s.MemUsed) + stMuted.Render(" / ") + styledSize(s.MemTotal) +
		"  " + styledPct(s.MemPercent)
	swapLine := "  " + stLabel.Render("Swap: ") + renderBar(s.SwapPercent, barW) +
		"  " + styledSize(s.SwapUsed) + stMuted.Render(" / ") + styledSize(s.SwapTotal) +
		"  " + styledPct(s.SwapPercent)

	content := sectionTitle("Memory") + "\n" + ramLine + "\n" + swapLine
	return panelStyle.Width(innerW).Render(content)
}

// LIPGLOSS THEME
func renderDiskNet(s *metrics.Snapshot, w int) string {
	// leftOuter + 1 (separator) + rightOuter = w exactly, so the Network
	// box's right edge aligns with the full-width panels above.
	leftOuter := (w - 1) / 2
	rightOuter := w - 1 - leftOuter
	leftInner := leftOuter - 4
	rightInner := rightOuter - 4

	return lipgloss.JoinHorizontal(lipgloss.Top,
		renderDisk(s, leftInner, max(4, leftInner-22)),
		" ",
		renderNet(s, rightInner, max(4, rightInner-22)),
	)
}

// LIPGLOSS THEME
func renderDisk(s *metrics.Snapshot, innerW, barW int) string {
	maxVal := math.Max(s.DiskReadPS, s.DiskWritePS)
	readPct := relativePct(s.DiskReadPS, maxVal)
	writePct := relativePct(s.DiskWritePS, maxVal)

	content := sectionTitle("Disk I/O") + "\n" +
		"  " + stNetLabel.Render("Read:  ") + renderBar(readPct, barW) + "  " + styledRate(s.DiskReadPS) + "\n" +
		"  " + stNetLabel.Render("Write: ") + renderBar(writePct, barW) + "  " + styledRate(s.DiskWritePS)
	return panelStyle.Width(innerW).Render(content)
}

// LIPGLOSS THEME
func renderNet(s *metrics.Snapshot, innerW, barW int) string {
	maxVal := math.Max(s.NetUpPS, s.NetDownPS)
	upPct := relativePct(s.NetUpPS, maxVal)
	downPct := relativePct(s.NetDownPS, maxVal)

	content := sectionTitle("Network") + "\n" +
		"  " + stNetLabel.Render("↑ Up:   ") + renderBar(upPct, barW) + "  " + styledRate(s.NetUpPS) + "\n" +
		"  " + stNetLabel.Render("↓ Down: ") + renderBar(downPct, barW) + "  " + styledRate(s.NetDownPS)
	return panelStyle.Width(innerW).Render(content)
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
