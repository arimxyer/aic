package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func outputJSON(entry *ChangelogEntry) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(entry); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func outputMarkdown(entry *ChangelogEntry) {
	if !entry.ReleasedAt.IsZero() {
		fmt.Printf("## %s (%s)\n\n", entry.Version, entry.ReleasedAt.Format("2006-01-02"))
	} else {
		fmt.Printf("## %s\n\n", entry.Version)
	}

	// Output sectioned changes
	for _, section := range entry.Sections {
		fmt.Printf("### %s\n\n", section.Name)
		for _, change := range section.Changes {
			fmt.Printf("- %s\n", change)
		}
		fmt.Println()
	}

	// Output ungrouped changes
	for _, change := range entry.Changes {
		fmt.Printf("- %s\n", change)
	}
}

func outputPlainText(displayName string, entry *ChangelogEntry) {
	if !entry.ReleasedAt.IsZero() {
		fmt.Printf("%s %s (%s)\n", displayName, entry.Version, entry.ReleasedAt.Format("2006-01-02"))
	} else {
		fmt.Printf("%s %s\n", displayName, entry.Version)
	}
	fmt.Println(strings.Repeat("-", 40))

	// Output sectioned changes
	for _, section := range entry.Sections {
		fmt.Printf("\n[%s]\n", section.Name)
		for _, change := range section.Changes {
			fmt.Printf("  * %s\n", change)
		}
	}

	// Output ungrouped changes
	if len(entry.Sections) > 0 && len(entry.Changes) > 0 {
		fmt.Println()
	}
	for _, change := range entry.Changes {
		fmt.Printf("  * %s\n", change)
	}
}
