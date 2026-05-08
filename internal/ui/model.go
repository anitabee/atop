package ui

import (
	"atop/internal/metrics"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/progress"
)

// Progress bars are rendered with ViewAs(pct) — no SetPercent, no animation,
// no FrameMsg chains. Bars only need width configured; percentages are passed
// directly at render time.

// Model is the root bubbletea model.
type Model struct {
	collector *metrics.Collector
	snap      *metrics.Snapshot
	interval  time.Duration
	width     int
	height    int
	err       error

	// LIPGLOSS THEME — bubbles progress bars
	cpuBar      progress.Model
	cpuPBar     progress.Model
	cpuEBar     progress.Model
	coreBars    []progress.Model
	gpuDevBars  []progress.Model
	gpuRenBars  []progress.Model
	gpuTilBars  []progress.Model
	ramBar      progress.Model
	swapBar     progress.Model
	diskReadBar progress.Model
	diskWriteBar progress.Model
	netUpBar    progress.Model
	netDownBar  progress.Model
}

func newBar() progress.Model {
	return progress.New(
		progress.WithDefaultBlend(),
		progress.WithoutPercentage(),
	)
}

type resultMsg struct {
	snap *metrics.Snapshot
	err  error
}

// New creates a Model with the given refresh interval.
func New(interval time.Duration) Model {
	m := Model{
		collector: metrics.New(),
		interval:  interval,

		// LIPGLOSS THEME — bubbles progress bars
		cpuBar:       newBar(),
		cpuPBar:      newBar(),
		cpuEBar:      newBar(),
		ramBar:       newBar(),
		swapBar:      newBar(),
		diskReadBar:  newBar(),
		diskWriteBar: newBar(),
		netUpBar:     newBar(),
		netDownBar:   newBar(),
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return m.collectNow()
}

func (m Model) collectNow() tea.Cmd {
	return func() tea.Msg {
		snap, err := m.collector.Collect()
		return resultMsg{snap: snap, err: err}
	}
}

func (m Model) scheduleNext() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(m.interval)
		snap, err := m.collector.Collect()
		return resultMsg{snap: snap, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateBarWidths()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case resultMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.snap = msg.snap
			m.err = nil
		}
		m.ensureBarSlices()
		return m, m.scheduleNext()
	}

	return m, nil
}

// updateBarWidths sets each bar's pixel width based on the current terminal size.
func (m *Model) updateBarWidths() {
	w := m.width
	if w < 60 {
		w = 60
	}
	innerW := w - 4
	cpuBarW := max(6, innerW-(2+labelColW+2+6))        // indent + label + gap + pct
	memBarW := max(6, innerW-(2+labelColW+2+6+3+6+2+6)) // indent + label + gap + size + " / " + size + gap + pct

	leftOuter := (w - 1) / 2
	rightOuter := w - 1 - leftOuter
	leftInner := leftOuter - 4
	rightInner := rightOuter - 4
	diskBarW := max(4, leftInner-(2+labelColW+2+11)) // indent + label + gap + rate (≤10 chars + 1 spare)
	netBarW := max(4, rightInner-(2+labelColW+2+10)) // indent + label + gap + rate (≤10 chars)

	m.cpuBar.SetWidth(cpuBarW)
	m.cpuPBar.SetWidth(cpuBarW)
	m.cpuEBar.SetWidth(cpuBarW)
	for i := range m.coreBars {
		m.coreBars[i].SetWidth(6)
	}
	for i := range m.gpuDevBars {
		m.gpuDevBars[i].SetWidth(cpuBarW)
		m.gpuRenBars[i].SetWidth(cpuBarW)
		m.gpuTilBars[i].SetWidth(cpuBarW)
	}
	m.ramBar.SetWidth(memBarW)
	m.swapBar.SetWidth(memBarW)
	m.diskReadBar.SetWidth(diskBarW)
	m.diskWriteBar.SetWidth(diskBarW)
	m.netUpBar.SetWidth(netBarW)
	m.netDownBar.SetWidth(netBarW)
}

// ensureBarSlices grows the per-core and per-GPU bar slices to match the
// current snapshot. Widths are set here; percentages are passed to ViewAs at
// render time so no FrameMsg animation is triggered.
func (m *Model) ensureBarSlices() {
	if m.snap == nil {
		return
	}
	s := m.snap

	if len(s.CPUCores) != len(m.coreBars) {
		m.coreBars = make([]progress.Model, len(s.CPUCores))
		for i := range m.coreBars {
			m.coreBars[i] = newBar()
			m.coreBars[i].SetWidth(6)
		}
	}

	if len(s.GPUs) != len(m.gpuDevBars) {
		w := m.width
		if w < 60 {
			w = 60
		}
		gpuBarW := max(6, (w-4)-(2+labelColW+2+6))
		m.gpuDevBars = make([]progress.Model, len(s.GPUs))
		m.gpuRenBars = make([]progress.Model, len(s.GPUs))
		m.gpuTilBars = make([]progress.Model, len(s.GPUs))
		for i := range s.GPUs {
			m.gpuDevBars[i] = newBar()
			m.gpuRenBars[i] = newBar()
			m.gpuTilBars[i] = newBar()
			m.gpuDevBars[i].SetWidth(gpuBarW)
			m.gpuRenBars[i].SetWidth(gpuBarW)
			m.gpuTilBars[i].SetWidth(gpuBarW)
		}
	}
}
