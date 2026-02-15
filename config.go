package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sourceItem struct {
	key     string
	display string
	enabled bool
}

type configModel struct {
	items    []sourceItem
	cursor   int
	saved    bool
	quitting bool
}

func (m configModel) Init() tea.Cmd {
	return nil
}

func (m configModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case " ":
			m.items[m.cursor].enabled = !m.items[m.cursor].enabled
		case "enter":
			m.saved = true
			m.quitting = true
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m configModel) View() string {
	if m.quitting {
		return ""
	}

	bold := lipgloss.NewStyle().Bold(true)

	var b strings.Builder
	b.WriteString("Configure sources (\u2191/\u2193 navigate, Space toggle, Enter save, q cancel):\n")
	b.WriteString("\n")

	for i, item := range m.items {
		check := "x"
		if !item.enabled {
			check = " "
		}
		pointer := "  "
		if i == m.cursor {
			pointer = "> "
		}
		line := fmt.Sprintf("%s[%s] %s", pointer, check, item.display)
		if i == m.cursor {
			line = bold.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

func runConfigCommand() {
	cfg := loadConfig()
	disabled := make(map[string]bool)
	for _, s := range cfg.DisabledSources {
		disabled[s] = true
	}

	var items []sourceItem
	for name, src := range sources {
		items = append(items, sourceItem{key: name, display: src.DisplayName, enabled: !disabled[name]})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].display < items[j].display
	})

	model := configModel{items: items}
	p := tea.NewProgram(model)
	result, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	final := result.(configModel)
	if final.saved {
		var disabledList []string
		for _, item := range final.items {
			if !item.enabled {
				disabledList = append(disabledList, item.key)
			}
		}
		sort.Strings(disabledList)
		newCfg := Config{DisabledSources: disabledList}
		if err := saveConfig(newCfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
		enabledCount := 0
		for _, item := range final.items {
			if item.enabled {
				enabledCount++
			}
		}
		fmt.Fprintf(os.Stderr, "Saved. %d/%d sources enabled.\n", enabledCount, len(final.items))
	} else if final.quitting {
		fmt.Fprintf(os.Stderr, "Cancelled.\n")
	}
}
