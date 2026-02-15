package main

import (
	"fmt"
	"os"
	"sort"

	"golang.org/x/term"
)

func runConfigCommand() {
	cfg := loadConfig()
	disabled := make(map[string]bool)
	for _, s := range cfg.DisabledSources {
		disabled[s] = true
	}

	// Build sorted list of source keys
	type sourceItem struct {
		key     string
		display string
		enabled bool
	}
	var items []sourceItem
	for name, src := range sources {
		items = append(items, sourceItem{key: name, display: src.DisplayName, enabled: !disabled[name]})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].display < items[j].display
	})

	cursor := 0

	// Enter raw mode before any rendering
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: unable to enter raw mode: %v\n", err)
		os.Exit(1)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Hide cursor
	fmt.Fprintf(os.Stderr, "\033[?25l")

	render := func() {
		// Move cursor to top and clear
		fmt.Fprintf(os.Stderr, "\033[%dA\033[J", len(items)+2)
		fmt.Fprintf(os.Stderr, "Configure sources (\u2191/\u2193 navigate, Space toggle, Enter save, q cancel):\r\n\r\n")
		for i, item := range items {
			check := "x"
			if !item.enabled {
				check = " "
			}
			pointer := "  "
			if i == cursor {
				pointer = "> "
			}
			fmt.Fprintf(os.Stderr, "%s[%s] %s\r\n", pointer, check, item.display)
		}
	}

	// Initial render
	fmt.Fprintf(os.Stderr, "Configure sources (\u2191/\u2193 navigate, Space toggle, Enter save, q cancel):\r\n\r\n")
	for i, item := range items {
		check := "x"
		if !item.enabled {
			check = " "
		}
		pointer := "  "
		if i == cursor {
			pointer = "> "
		}
		fmt.Fprintf(os.Stderr, "%s[%s] %s\r\n", pointer, check, item.display)
	}

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			break
		}

		if n == 1 {
			switch buf[0] {
			case 'q', 3: // q or Ctrl+C
				fmt.Fprintf(os.Stderr, "\033[?25h")
				term.Restore(int(os.Stdin.Fd()), oldState)
				fmt.Fprintf(os.Stderr, "\nCancelled.\n")
				return
			case ' ':
				items[cursor].enabled = !items[cursor].enabled
				render()
			case 13: // Enter
				fmt.Fprintf(os.Stderr, "\033[?25h")
				term.Restore(int(os.Stdin.Fd()), oldState)
				var disabledList []string
				for _, item := range items {
					if !item.enabled {
						disabledList = append(disabledList, item.key)
					}
				}
				sort.Strings(disabledList)
				newCfg := Config{DisabledSources: disabledList}
				if err := saveConfig(newCfg); err != nil {
					fmt.Fprintf(os.Stderr, "\nError saving config: %v\n", err)
					os.Exit(1)
				}
				enabledCount := 0
				for _, item := range items {
					if item.enabled {
						enabledCount++
					}
				}
				fmt.Fprintf(os.Stderr, "\nSaved. %d/%d sources enabled.\n", enabledCount, len(items))
				return
			case 'k': // vim up
				if cursor > 0 {
					cursor--
				}
				render()
			case 'j': // vim down
				if cursor < len(items)-1 {
					cursor++
				}
				render()
			}
		}

		// Arrow key sequences: ESC [ A/B
		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 65: // Up
				if cursor > 0 {
					cursor--
				}
				render()
			case 66: // Down
				if cursor < len(items)-1 {
					cursor++
				}
				render()
			}
		}
	}
}
