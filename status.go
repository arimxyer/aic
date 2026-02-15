package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
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

	// Build table rows
	rows := make([][]string, len(statusEntries))
	for i, e := range statusEntries {
		recentMarker := ""
		if e.UpdatedRecently {
			recentMarker = "✓"
		}
		rows[i] = []string{
			e.Name,
			recentMarker,
			e.InstalledVersion,
			e.Version,
			e.UpdatedAgo,
			e.AvgReleaseFreq,
		}
	}

	headerStyle := lipgloss.NewStyle().Bold(true)

	t := table.New().
		Border(lipgloss.NormalBorder()).
		Headers("Tool", "24h", "Installed", "Latest", "Updated", "Release Freq.").
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			s := lipgloss.NewStyle()
			if col == 1 {
				s = s.Foreground(lipgloss.Color("2"))
			}
			return s
		})

	fmt.Println(t)
}
