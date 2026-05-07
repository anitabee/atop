package ui

import (
	"atop/internal/metrics"
	"image/color"
	"math"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/progress"
	"charm.land/lipgloss/v2"
)

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

// LIPGLOSS THEME
func newBar() progress.Model {
	gradient := lipgloss.Blend1D(101,
		lipgloss.Color("#04B575"),
		lipgloss.Color("#EDFF82"),
		lipgloss.Color("#EB4268"),
	)
	return progress.New(
		progress.WithColorFunc(func(total, current float64) color.Color {
			if total <= 0 {
				return gradient[0]
			}
			idx := int(math.Round(current / total * 100))
			if idx < 0 {
				idx = 0
			}
			if idx > 100 {
				idx = 100
			}
			return gradient[idx]
		}),
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

	case progress.FrameMsg:
		var cmds []tea.Cmd
		var cmd tea.Cmd

		m.cpuBar, cmd = m.cpuBar.Update(msg)
		cmds = append(cmds, cmd)
		m.cpuPBar, cmd = m.cpuPBar.Update(msg)
		cmds = append(cmds, cmd)
		m.cpuEBar, cmd = m.cpuEBar.Update(msg)
		cmds = append(cmds, cmd)
		for i := range m.coreBars {
			m.coreBars[i], cmd = m.coreBars[i].Update(msg)
			cmds = append(cmds, cmd)
		}
		for i := range m.gpuDevBars {
			m.gpuDevBars[i], cmd = m.gpuDevBars[i].Update(msg)
			cmds = append(cmds, cmd)
			m.gpuRenBars[i], cmd = m.gpuRenBars[i].Update(msg)
			cmds = append(cmds, cmd)
			m.gpuTilBars[i], cmd = m.gpuTilBars[i].Update(msg)
			cmds = append(cmds, cmd)
		}
		m.ramBar, cmd = m.ramBar.Update(msg)
		cmds = append(cmds, cmd)
		m.swapBar, cmd = m.swapBar.Update(msg)
		cmds = append(cmds, cmd)
		m.diskReadBar, cmd = m.diskReadBar.Update(msg)
		cmds = append(cmds, cmd)
		m.diskWriteBar, cmd = m.diskWriteBar.Update(msg)
		cmds = append(cmds, cmd)
		m.netUpBar, cmd = m.netUpBar.Update(msg)
		cmds = append(cmds, cmd)
		m.netDownBar, cmd = m.netDownBar.Update(msg)
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)

	case resultMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.snap = msg.snap
			m.err = nil
		}
		cmds := m.updateBarPercents()
		cmds = append(cmds, m.scheduleNext())
		return m, tea.Batch(cmds...)
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
	cpuBarW := max(6, innerW-18)
	memBarW := max(6, innerW-32)

	leftOuter := (w - 1) / 2
	rightOuter := w - 1 - leftOuter
	leftInner := leftOuter - 4
	rightInner := rightOuter - 4
	diskBarW := max(4, leftInner-22)
	netBarW := max(4, rightInner-22)

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

// updateBarPercents calls SetPercent on every bar and returns the resulting cmds.
func (m *Model) updateBarPercents() []tea.Cmd {
	if m.snap == nil {
		return nil
	}
	s := m.snap
	var cmds []tea.Cmd

	cmds = append(cmds, m.cpuBar.SetPercent(s.CPUTotal/100))

	if s.IsAppleSi && s.NumPCores > 0 && s.NumECores > 0 && len(s.CPUCores) > 0 {
		n := len(s.CPUCores)
		pEnd := min(s.NumPCores, n)
		eStart := pEnd
		eEnd := min(eStart+s.NumECores, n)
		cmds = append(cmds, m.cpuPBar.SetPercent(avgSlice(s.CPUCores[:pEnd])/100))
		cmds = append(cmds, m.cpuEBar.SetPercent(avgSlice(s.CPUCores[eStart:eEnd])/100))
	}

	// Resize per-core slice when core count changes.
	if len(s.CPUCores) != len(m.coreBars) {
		m.coreBars = make([]progress.Model, len(s.CPUCores))
		for i := range m.coreBars {
			m.coreBars[i] = newBar()
			m.coreBars[i].SetWidth(6)
		}
	}
	for i, p := range s.CPUCores {
		cmds = append(cmds, m.coreBars[i].SetPercent(p/100))
	}

	// Resize GPU slices when GPU count changes.
	if len(s.GPUs) != len(m.gpuDevBars) {
		w := m.width
		if w < 60 {
			w = 60
		}
		gpuBarW := max(6, (w-4)-18)
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
	for i, g := range s.GPUs {
		cmds = append(cmds, m.gpuDevBars[i].SetPercent(g.DeviceUtil/100))
		cmds = append(cmds, m.gpuRenBars[i].SetPercent(g.RendererUtil/100))
		cmds = append(cmds, m.gpuTilBars[i].SetPercent(g.TilerUtil/100))
	}

	cmds = append(cmds, m.ramBar.SetPercent(s.MemPercent/100))
	cmds = append(cmds, m.swapBar.SetPercent(s.SwapPercent/100))

	maxDisk := math.Max(s.DiskReadPS, s.DiskWritePS)
	cmds = append(cmds, m.diskReadBar.SetPercent(relativePct(s.DiskReadPS, maxDisk)/100))
	cmds = append(cmds, m.diskWriteBar.SetPercent(relativePct(s.DiskWritePS, maxDisk)/100))

	maxNet := math.Max(s.NetUpPS, s.NetDownPS)
	cmds = append(cmds, m.netUpBar.SetPercent(relativePct(s.NetUpPS, maxNet)/100))
	cmds = append(cmds, m.netDownBar.SetPercent(relativePct(s.NetDownPS, maxNet)/100))

	return cmds
}
