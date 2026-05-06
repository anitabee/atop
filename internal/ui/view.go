package ui

import (
	"atop/internal/metrics"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.snap == nil && m.err == nil {
		return "\n  " + rainbowText("Collecting metrics…") + "\n\n  Press q to quit"
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
	parts = append(parts, renderMemory(m.snap, w), renderDiskNet(m.snap, w), stDim.Render("  q to quit"))
	return strings.Join(parts, "\n")
}

// ── Header ────────────────────────────────────────────────────────────────────

func renderHeader(s *metrics.Snapshot, w int) string {
	hostname, _ := os.Hostname()
	logo := rainbowText(" atop")
	right := stDim.Render(fmt.Sprintf("%s  %s ", hostname, time.Now().Format("15:04:05")))

	// Calculate raw (non-ANSI) widths for gap arithmetic.
	rawLeft := len(" atop")
	rawRight := lipgloss.Width(right)
	gap := max(0, w-rawLeft-rawRight)

	header := logo + strings.Repeat(" ", gap) + right
	return lipgloss.NewStyle().
		Background(lipgloss.Color("#0D0D1A")).
		Padding(0, 0).
		Width(w).
		Render(header)
}

// ── CPU ───────────────────────────────────────────────────────────────────────

func renderCPU(s *metrics.Snapshot, w int) string {
	innerW := w - 4

	label := "CPU"
	if s.CPUModel != "" {
		label = "CPU: " + truncate(s.CPUModel, innerW/2)
	}
	loadStr := fmt.Sprintf("Load: %.2f  %.2f  %.2f", s.Load1, s.Load5, s.Load15)
	gap := max(0, innerW-len(label)-lipgloss.Width(loadStr))
	header := sectionTitle(label, colCPU) + strings.Repeat(" ", gap) + stDim.Render(loadStr)

	barW := max(6, innerW-18)
	totalLine := fmt.Sprintf("  Total:  %s  %5.1f%%",
		progressBar(s.CPUTotal, barW, barColor(s.CPUTotal, colCPU)), s.CPUTotal)

	content := header + "\n" + totalLine

	// Apple Silicon P-core / E-core breakdown
	if s.IsAppleSi && s.NumPCores > 0 && s.NumECores > 0 && len(s.CPUCores) > 0 {
		n := len(s.CPUCores)
		pEnd := min(s.NumPCores, n)
		eStart := pEnd
		eEnd := min(eStart+s.NumECores, n)

		pAvg := avgSlice(s.CPUCores[:pEnd])
		eAvg := avgSlice(s.CPUCores[eStart:eEnd])

		content += "\n" + fmt.Sprintf("  P-Core: %s  %5.1f%%",
			progressBar(pAvg, barW, barColor(pAvg, colCPU)), pAvg)
		content += "\n" + fmt.Sprintf("  E-Core: %s  %5.1f%%",
			progressBar(eAvg, barW, barColor(eAvg, rainbowAt(1))), eAvg)
	}

	// Per-core grid
	if len(s.CPUCores) > 0 {
		content += "\n"
		for _, line := range coreGrid(s.CPUCores, innerW) {
			content += "\n" + line
		}
	}

	return sectionBox(colCPU).Width(innerW).Render(content)
}

// coreGrid renders individual core bars cycling through rainbow colours.
func coreGrid(cores []float64, innerW int) []string {
	const cellW = 18
	cols := max(1, innerW/cellW)

	var lines []string
	var row strings.Builder
	for i, p := range cores {
		// Normal colour cycles through the rainbow; overrides at warn/danger levels.
		col := barColor(p, rainbowAt(i))
		bar := progressBar(p, 6, col)
		cell := fmt.Sprintf(" %s %s %4.0f%%",
			stDim.Render(fmt.Sprintf("C%02d:", i)), bar, p)
		row.WriteString(cell)
		if (i+1)%cols == 0 || i == len(cores)-1 {
			lines = append(lines, row.String())
			row.Reset()
		}
	}
	return lines
}

// ── GPU ───────────────────────────────────────────────────────────────────────

func renderGPU(s *metrics.Snapshot, w int) string {
	if len(s.GPUs) == 0 {
		return ""
	}
	innerW := w - 4
	barW := max(6, innerW-18)

	var content string
	for i, g := range s.GPUs {
		label := g.Name
		if len(s.GPUs) > 1 {
			label = fmt.Sprintf("GPU %d: %s", i+1, g.Name)
		} else {
			label = "GPU: " + g.Name
		}
		if i == 0 {
			content = sectionTitle(label, colGPU)
		} else {
			content += "\n" + sectionTitle(label, colGPU)
		}

		content += "\n" + fmt.Sprintf("  Device:   %s  %5.1f%%",
			progressBar(g.DeviceUtil, barW, barColor(g.DeviceUtil, colGPU)), g.DeviceUtil)
		content += "\n" + fmt.Sprintf("  Renderer: %s  %5.1f%%",
			progressBar(g.RendererUtil, barW, barColor(g.RendererUtil, rainbowAt(6))), g.RendererUtil)
		content += "\n" + fmt.Sprintf("  Tiler:    %s  %5.1f%%",
			progressBar(g.TilerUtil, barW, barColor(g.TilerUtil, rainbowAt(1))), g.TilerUtil)
		if g.MemInUse > 0 {
			content += "\n" + fmt.Sprintf("  VRAM:     %s in use", fmtSize(g.MemInUse))
		}
	}

	return sectionBox(colGPU).Width(innerW).Render(content)
}

// ── Memory ────────────────────────────────────────────────────────────────────

func renderMemory(s *metrics.Snapshot, w int) string {
	innerW := w - 4
	barW := max(6, innerW-32)

	ramLine := fmt.Sprintf("  RAM:  %s  %s / %s  %5.1f%%",
		progressBar(s.MemPercent, barW, barColor(s.MemPercent, colMem)),
		fmtSize(s.MemUsed), fmtSize(s.MemTotal), s.MemPercent)
	swapLine := fmt.Sprintf("  Swap: %s  %s / %s  %5.1f%%",
		progressBar(s.SwapPercent, barW, barColor(s.SwapPercent, colMem)),
		fmtSize(s.SwapUsed), fmtSize(s.SwapTotal), s.SwapPercent)

	content := sectionTitle("Memory", colMem) + "\n" + ramLine + "\n" + swapLine
	return sectionBox(colMem).Width(innerW).Render(content)
}

// ── Disk + Network (side by side) ────────────────────────────────────────────

func renderDiskNet(s *metrics.Snapshot, w int) string {
	halfOuter := (w - 1) / 2
	halfInner := halfOuter - 4
	barW := max(4, halfInner-22)

	diskBox := renderDisk(s, halfInner, barW)
	netBox := renderNet(s, halfInner, barW)
	return lipgloss.JoinHorizontal(lipgloss.Top, diskBox, " ", netBox)
}

func renderDisk(s *metrics.Snapshot, innerW, barW int) string {
	maxVal := math.Max(s.DiskReadPS, s.DiskWritePS)
	readPct := relativePct(s.DiskReadPS, maxVal)
	writePct := relativePct(s.DiskWritePS, maxVal)

	content := sectionTitle("Disk I/O", colDisk) + "\n" +
		fmt.Sprintf("  Read:  %s  %s", progressBar(readPct, barW, colDisk), fmtRate(s.DiskReadPS)) + "\n" +
		fmt.Sprintf("  Write: %s  %s", progressBar(writePct, barW, rainbowAt(4)), fmtRate(s.DiskWritePS))
	return sectionBox(colDisk).Width(innerW).Render(content)
}

func renderNet(s *metrics.Snapshot, innerW, barW int) string {
	maxVal := math.Max(s.NetUpPS, s.NetDownPS)
	upPct := relativePct(s.NetUpPS, maxVal)
	downPct := relativePct(s.NetDownPS, maxVal)

	content := sectionTitle("Network", colNet) + "\n" +
		fmt.Sprintf("  ↑ Up:   %s  %s", progressBar(upPct, barW, colNet), fmtRate(s.NetUpPS)) + "\n" +
		fmt.Sprintf("  ↓ Down: %s  %s", progressBar(downPct, barW, rainbowAt(5)), fmtRate(s.NetDownPS))
	return sectionBox(colNet).Width(innerW).Render(content)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func progressBar(pct float64, width int, color lipgloss.Color) string {
	if width <= 0 {
		return ""
	}
	pct = clamp(pct, 0, 100)
	filled := clampInt(int(math.Round(float64(width)*pct/100)), 0, width)
	empty := width - filled
	return lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(blendWithBg(color, 0.20)).Render(strings.Repeat("░", empty))
}

// blendWithBg blends col toward the dark background at the given opacity (0–1).
// opacity=0.20 means 80% transparent: col contributes 20%, background 80%.
func blendWithBg(col lipgloss.Color, opacity float64) lipgloss.Color {
	s := string(col)
	if len(s) != 7 || s[0] != '#' {
		return colEmpty
	}
	r, e1 := strconv.ParseInt(s[1:3], 16, 64)
	g, e2 := strconv.ParseInt(s[3:5], 16, 64)
	b, e3 := strconv.ParseInt(s[5:7], 16, 64)
	if e1 != nil || e2 != nil || e3 != nil {
		return colEmpty
	}
	// colEmpty background is #1C1C2C
	const bgR, bgG, bgB = 0x1C, 0x1C, 0x2C
	nr := int(float64(r)*opacity + float64(bgR)*(1-opacity))
	ng := int(float64(g)*opacity + float64(bgG)*(1-opacity))
	nb := int(float64(b)*opacity + float64(bgB)*(1-opacity))
	return lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", nr, ng, nb))
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

func fmtSize(b uint64) string {
	const (
		gb = 1 << 30
		mb = 1 << 20
		kb = 1 << 10
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1fGB", float64(b)/gb)
	case b >= mb:
		return fmt.Sprintf("%.0fMB", float64(b)/mb)
	case b >= kb:
		return fmt.Sprintf("%.0fKB", float64(b)/kb)
	default:
		return fmt.Sprintf("%dB", b)
	}
}

func fmtRate(bps float64) string {
	const (
		gb = 1 << 30
		mb = 1 << 20
		kb = 1 << 10
	)
	switch {
	case bps >= gb:
		return fmt.Sprintf("%.1f GB/s", bps/gb)
	case bps >= mb:
		return fmt.Sprintf("%.1f MB/s", bps/mb)
	case bps >= kb:
		return fmt.Sprintf("%.1f KB/s", bps/kb)
	default:
		return fmt.Sprintf("%.0f B/s", bps)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
