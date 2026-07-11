package ui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// visibleRows is how many options render at once. The list scrolls within this
// window, so huge result sets (e.g. hundreds of Jackett hits) stay usable
// instead of overflowing the terminal and breaking the interactive selector.
const visibleRows = 12

type model struct {
	title  string
	labels []string
	cursor int
	offset int // index of the first visible row
	chosen int
	quit   bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c", "q", "esc":
			m.quit = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.labels)-1 {
				m.cursor++
			}
		case "pgup", "left", "h":
			m.cursor -= visibleRows
			if m.cursor < 0 {
				m.cursor = 0
			}
		case "pgdown", "right", "l":
			m.cursor += visibleRows
			if m.cursor > len(m.labels)-1 {
				m.cursor = len(m.labels) - 1
			}
		case "home", "g":
			m.cursor = 0
		case "end", "G":
			m.cursor = len(m.labels) - 1
		case "enter":
			m.chosen = m.cursor
			return m, tea.Quit
		}
		// Keep the cursor inside the visible window.
		if m.cursor < m.offset {
			m.offset = m.cursor
		}
		if m.cursor >= m.offset+visibleRows {
			m.offset = m.cursor - visibleRows + 1
		}
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString(m.title + "\n\n")

	end := m.offset + visibleRows
	if end > len(m.labels) {
		end = len(m.labels)
	}
	if m.offset > 0 {
		b.WriteString("   ↑ more above\n")
	}
	for i := m.offset; i < end; i++ {
		cursor := "  "
		if i == m.cursor {
			cursor = "▶ "
		}
		b.WriteString(cursor + m.labels[i] + "\n")
	}
	if end < len(m.labels) {
		b.WriteString("   ↓ more below\n")
	}
	fmt.Fprintf(&b, "\n%d/%d · ↑/↓ move · pgup/pgdn jump · enter select · q quit\n",
		m.cursor+1, len(m.labels))
	return b.String()
}

// Select shows an interactive list and returns the chosen index. It auto-selects
// index 0 when there is a single option or no interactive terminal is attached.
func Select(title string, labels []string) (int, error) {
	if len(labels) == 0 {
		return 0, fmt.Errorf("nothing to select")
	}
	if len(labels) == 1 || !isInteractive() {
		return 0, nil
	}
	res, err := tea.NewProgram(model{title: title, labels: labels}).Run()
	if err != nil {
		return 0, err
	}
	final := res.(model)
	if final.quit {
		return -1, fmt.Errorf("selection cancelled")
	}
	return final.chosen, nil
}

func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
