package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

func runLatestCommand(jsonOutput bool) {
	cutoff := time.Now().Add(-24 * time.Hour)

	type result struct {
		source  string
		display string
		entry   *ChangelogEntry
		err     error
	}

	active := enabledSources()
	results := make(chan result, len(active))
	var wg sync.WaitGroup

	for name, src := range active {
		wg.Add(1)
		go func(name string, src Source) {
			defer wg.Done()
			entries, err := src.Fetch()
			if err != nil {
				results <- result{source: name, display: src.DisplayName, err: err}
				return
			}
			if len(entries) > 0 {
				entry := entries[0]
				entry.Source = src.DisplayName
				results <- result{source: name, display: src.DisplayName, entry: &entry}
			}
		}(name, src)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var recentEntries []ChangelogEntry
	for r := range results {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to fetch %s: %v\n", r.display, r.err)
			continue
		}
		if r.entry != nil && !r.entry.ReleasedAt.IsZero() && r.entry.ReleasedAt.After(cutoff) {
			recentEntries = append(recentEntries, *r.entry)
		}
	}

	// Sort by release date descending
	sort.Slice(recentEntries, func(i, j int) bool {
		return recentEntries[i].ReleasedAt.After(recentEntries[j].ReleasedAt)
	})

	if len(recentEntries) == 0 {
		fmt.Println("No releases in the last 24 hours.")
		return
	}

	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		encoder.Encode(recentEntries)
	} else {
		for i, entry := range recentEntries {
			if i > 0 {
				fmt.Println()
			}
			outputRendered(entry.Source, &entry)
		}
	}
}
