package ui

import (
	"atop/internal/metrics"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Model is the root bubbletea model.
type Model struct {
	collector *metrics.Collector
	snap      *metrics.Snapshot
	interval  time.Duration
	width     int
	height    int
	err       error
}

type resultMsg struct {
	snap *metrics.Snapshot
	err  error
}

// New creates a Model with the given refresh interval.
func New(interval time.Duration) Model {
	return Model{
		collector: metrics.New(),
		interval:  interval,
	}
}

func (m Model) Init() tea.Cmd {
	// Collect immediately so we have data on first render.
	return m.collectNow()
}

// collectNow fires a collection with no sleep — used for the first sample.
func (m Model) collectNow() tea.Cmd {
	return func() tea.Msg {
		snap, err := m.collector.Collect()
		return resultMsg{snap: snap, err: err}
	}
}

// scheduleNext sleeps for the interval then collects.
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
		return m, m.scheduleNext()
	}

	return m, nil
}
