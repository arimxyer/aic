package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

func runStatusCommand(jsonOutput bool) {
	type statusResult struct {
		source      string
		displayName string
		entries     []ChangelogEntry
		err         error
	}

	active := enabledSources()
	results := make(chan statusResult, len(active))
	installedResults := make(map[string]InstalledInfo)
	var mu sync.Mutex
	var wg sync.WaitGroup

	ctx := context.Background()

	// Fetch releases and detect installed tools concurrently
	for name, src := range active {
		wg.Add(2)
		go func(name string, src Source) {
			defer wg.Done()
			entries, err := src.Fetch()
			results <- statusResult{
				source:      name,
				displayName: src.DisplayName,
				entries:     entries,
				err:         err,
			}
		}(name, src)
		go func(name string, src Source) {
			defer wg.Done()
			info := src.DetectInstalled(ctx)
			mu.Lock()
			installedResults[name] = info
			mu.Unlock()
		}(name, src)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	type statusEntry struct {
		Name             string `json:"name"`
		InstalledVersion string `json:"installed_version"`
		Version          string `json:"version"`
		UpdatedAgo       string `json:"updated_ago"`
		UpdatedRecently  bool   `json:"updated_recently"`
		AvgReleaseFreq   string `json:"avg_release_freq"`
		sourceName       string
		releasedAt       time.Time
	}

	var statusEntries []statusEntry
	cutoff := time.Now().Add(-24 * time.Hour)

	for r := range results {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to fetch %s: %v\n", r.displayName, r.err)
			continue
		}

		if len(r.entries) == 0 {
			continue
		}

		entry := statusEntry{
			Name:             r.displayName,
			InstalledVersion: "-",
			Version:          r.entries[0].Version,
			UpdatedAgo:       "-",
			UpdatedRecently:  false,
			AvgReleaseFreq:   "-",
			sourceName:       r.source,
			releasedAt:       r.entries[0].ReleasedAt,
		}

		if !r.entries[0].ReleasedAt.IsZero() {
			entry.UpdatedAgo = formatRelativeTime(r.entries[0].ReleasedAt)
			entry.UpdatedRecently = r.entries[0].ReleasedAt.After(cutoff)
		}

		// Calculate average release frequency from up to 10 entries
		entry.AvgReleaseFreq = calculateAvgReleaseFreq(r.entries)

		statusEntries = append(statusEntries, entry)
	}

	// Populate installed versions
	for i := range statusEntries {
		mu.Lock()
		info, ok := installedResults[statusEntries[i].sourceName]
		mu.Unlock()
		if ok && info.Installed {
			if info.Version != "" {
				statusEntries[i].InstalledVersion = info.Version
			} else {
				statusEntries[i].InstalledVersion = "yes"
			}
		}
	}

	// Sort by most recently updated
	sort.Slice(statusEntries, func(i, j int) bool {
		if statusEntries[i].releasedAt.IsZero() && statusEntries[j].releasedAt.IsZero() {
			return statusEntries[i].Name < statusEntries[j].Name
		}
		if statusEntries[i].releasedAt.IsZero() {
			return false
		}
		if statusEntries[j].releasedAt.IsZero() {
			return true
		}
		return statusEntries[i].releasedAt.After(statusEntries[j].releasedAt)
	})

	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		encoder.Encode(statusEntries)
		return
	}

	// Print table with borders
	// Column widths
	const (
		colTool      = 20
		col24h       = 3
		colInstalled = 12
		colVersion   = 12
		colUpdated   = 10
		colFreq      = 19
	)

	border := func(left, mid, right string) string {
		return fmt.Sprintf("%s%s%s%s%s%s%s%s%s%s%s%s%s\n",
			left, strings.Repeat("─", colTool+2),
			mid, strings.Repeat("─", col24h+2),
			mid, strings.Repeat("─", colInstalled+2),
			mid, strings.Repeat("─", colVersion+2),
			mid, strings.Repeat("─", colUpdated+2),
			mid, strings.Repeat("─", colFreq+2),
			right)
	}

	// Top border
	fmt.Print(border("┌", "┬", "┐"))

	// Header row
	fmt.Printf("│ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │\n",
		colTool, "Tool",
		col24h, "24h",
		colInstalled, "Installed",
		colVersion, "Latest",
		colUpdated, "Updated",
		colFreq, "Vers. Release Freq.")

	// Header separator
	fmt.Print(border("├", "┼", "┤"))

	// Data rows
	for _, e := range statusEntries {
		recentMarker := "   "
		if e.UpdatedRecently {
			recentMarker = "[✓]"
		}
		fmt.Printf("│ %-*s │ %s │ %-*s │ %-*s │ %-*s │ %-*s │\n",
			colTool, truncateString(e.Name, colTool),
			recentMarker,
			colInstalled, truncateString(e.InstalledVersion, colInstalled),
			colVersion, truncateString(e.Version, colVersion),
			colUpdated, e.UpdatedAgo,
			colFreq, e.AvgReleaseFreq)
	}

	// Bottom border
	fmt.Print(border("└", "┴", "┘"))
}
