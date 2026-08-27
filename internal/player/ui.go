package player

import (
	"fmt"
	"strings"
	"time"

	"github.com/arshnah/replay/internal/vt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var ansiColors = []lipgloss.Color{
	"0", "1", "2", "3", "4", "5", "6", "7",
	"8", "9", "10", "11", "12", "13", "14", "15",
}

type Model struct {
	session *Session
	pos     time.Duration
	step    time.Duration
	playing bool
}

func New(s *Session) Model {
	step := s.Duration / 100
	if step <= 0 {
		step = time.Millisecond * 100
	}
	return Model{session: s, step: step}
}

func (m Model) Init() tea.Cmd {
	return nil
}

type tickMsg struct{}

func tickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "left", "h":
			m.pos = clampDur(m.pos-m.step, 0, m.session.Duration)
		case "right", "l":
			m.pos = clampDur(m.pos+m.step, 0, m.session.Duration)
		case "shift+left", "H":
			m.pos = clampDur(m.pos-m.step*10, 0, m.session.Duration)
		case "shift+right", "L":
			m.pos = clampDur(m.pos+m.step*10, 0, m.session.Duration)
		case "home", "g":
			m.pos = 0
		case "end", "G":
			m.pos = m.session.Duration
		case " ":
			m.playing = !m.playing
			if m.playing {
				return m, tickCmd()
			}
		}
	case tickMsg:
		if !m.playing {
			return m, nil
		}
		m.pos = clampDur(m.pos+m.step, 0, m.session.Duration)
		if m.pos >= m.session.Duration {
			m.playing = false
			return m, nil
		}
		return m, tickCmd()
	}
	return m, nil
}

func clampDur(v, lo, hi time.Duration) time.Duration {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (m Model) View() string {
	screen := m.session.ScreenAt(m.pos)
	var b strings.Builder

	for y := range screen.Rows {
		for x := range screen.Cols {
			cell := screen.Grid[y][x]
			b.WriteString(renderCell(cell))
		}
		b.WriteString("\n")
	}

	b.WriteString(strings.Repeat("─", screen.Cols))
	b.WriteString("\n")
	b.WriteString(m.footer())

	return b.String()
}

func renderCell(c vt.Cell) string {
	if c.FG == vt.DefaultColor && c.BG == vt.DefaultColor && !c.Bold {
		return string(c.Ch)
	}
	style := lipgloss.NewStyle()
	if c.FG != vt.DefaultColor && c.FG < len(ansiColors) {
		style = style.Foreground(ansiColors[c.FG])
	}
	if c.BG != vt.DefaultColor && c.BG < len(ansiColors) {
		style = style.Background(ansiColors[c.BG])
	}
	if c.Bold {
		style = style.Bold(true)
	}
	return style.Render(string(c.Ch))
}

func (m Model) footer() string {
	pct := 0.0
	if m.session.Duration > 0 {
		pct = float64(m.pos) / float64(m.session.Duration)
	}
	barWidth := 40
	filled := int(pct * float64(barWidth))
	bar := strings.Repeat("=", filled) + strings.Repeat(" ", barWidth-filled)

	state := "paused"
	if m.playing {
		state = "playing"
	}

	return fmt.Sprintf("[%s] %s / %s  (%s)  h/l seek  H/L jump  space play/pause  g/G start/end  q quit",
		bar, fmtDur(m.pos), fmtDur(m.session.Duration), state)
}

func fmtDur(d time.Duration) string {
	d = d.Round(time.Millisecond * 10)
	return fmt.Sprintf("%02d:%02d.%02d", int(d.Minutes()), int(d.Seconds())%60, (d.Milliseconds()/10)%100)
}
