package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type pickModel struct {
	entries     []ChangelogEntry
	displayName string
	cursor      int
	selected    int
	quitting    bool
}

func (m pickModel) Init() tea.Cmd {
	return nil
}

func (m pickModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = m.cursor
			m.quitting = true
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			m.selected = -1
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m pickModel) View() string {
	if m.quitting {
		return ""
	}

	bold := lipgloss.NewStyle().Bold(true)
	dim := lipgloss.NewStyle().Faint(true)

	var b strings.Builder

	for i, entry := range m.entries {
		date := "-"
		ago := "-"
		if !entry.ReleasedAt.IsZero() {
			date = entry.ReleasedAt.Format("2006-01-02")
			ago = formatRelativeTime(entry.ReleasedAt)
		}

		pointer := "  "
		if i == m.cursor {
			pointer = "> "
		}

		line := fmt.Sprintf("%s%-12s  %-12s  %s", pointer, entry.Version, date, ago)
		if i == m.cursor {
			line = bold.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dim.Render("  ↑/↓ navigate • enter select • q cancel"))
	b.WriteString("\n")

	return b.String()
}

func runPickCommand(displayName string, entries []ChangelogEntry) {
	model := pickModel{
		entries:     entries,
		displayName: displayName,
		selected:    -1,
	}

	p := tea.NewProgram(model)
	result, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	final := result.(pickModel)
	if final.selected >= 0 && final.selected < len(entries) {
		outputRendered(displayName, &entries[final.selected])
	}
}
