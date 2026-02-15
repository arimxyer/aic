package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"
)

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default: // linux, freebsd, etc.
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening browser: %v\n", err)
	}
}

func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	duration := time.Since(t)

	minutes := int(duration.Minutes())
	hours := int(duration.Hours())
	days := hours / 24
	weeks := days / 7
	months := days / 30

	if minutes < 60 {
		return fmt.Sprintf("%dm ago", minutes)
	}
	if hours < 24 {
		return fmt.Sprintf("%dh ago", hours)
	}
	if days < 7 {
		return fmt.Sprintf("%dd ago", days)
	}
	if weeks < 4 {
		return fmt.Sprintf("%dw ago", weeks)
	}
	return fmt.Sprintf("%dmo ago", months)
}

func calculateAvgReleaseFreq(entries []ChangelogEntry) string {
	// Need at least 2 entries with valid dates to calculate average
	var validEntries []ChangelogEntry
	for _, e := range entries {
		if !e.ReleasedAt.IsZero() {
			validEntries = append(validEntries, e)
		}
		if len(validEntries) >= 10 {
			break
		}
	}

	if len(validEntries) < 2 {
		return "-"
	}

	// Calculate intervals between consecutive releases
	var totalDuration time.Duration
	for i := 0; i < len(validEntries)-1; i++ {
		interval := validEntries[i].ReleasedAt.Sub(validEntries[i+1].ReleasedAt)
		totalDuration += interval
	}

	avgDuration := totalDuration / time.Duration(len(validEntries)-1)

	// Format as relative time
	hours := int(avgDuration.Hours())
	days := hours / 24
	weeks := days / 7
	months := days / 30

	if days < 1 {
		return fmt.Sprintf("~%dh", hours)
	}
	if days < 7 {
		return fmt.Sprintf("~%dd", days)
	}
	if weeks < 4 {
		return fmt.Sprintf("~%dw", weeks)
	}
	return fmt.Sprintf("~%dmo", months)
}
