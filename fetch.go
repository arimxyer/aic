package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Section struct {
	Name    string   `json:"name"`
	Changes []string `json:"changes"`
}

type ChangelogEntry struct {
	Version    string    `json:"version"`
	ReleasedAt time.Time `json:"released_at,omitempty"`
	Source     string    `json:"source,omitempty"`
	Sections   []Section `json:"sections,omitempty"`
	Changes    []string  `json:"changes,omitempty"`
}

func fetchGitHubReleases(owner, repo string) ([]ChangelogEntry, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "aic-changelog")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	var releases []struct {
		TagName     string `json:"tag_name"`
		Name        string `json:"name"`
		Body        string `json:"body"`
		PublishedAt string `json:"published_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse releases: %w", err)
	}

	var entries []ChangelogEntry
	for _, rel := range releases {
		ver := rel.TagName
		ver = strings.TrimPrefix(ver, "v")
		ver = strings.TrimPrefix(ver, "rust-v")

		sections, ungroupedChanges := parseReleaseBody(rel.Body)

		releasedAt, _ := time.Parse(time.RFC3339, rel.PublishedAt)

		entries = append(entries, ChangelogEntry{
			Version:    ver,
			ReleasedAt: releasedAt,
			Sections:   sections,
			Changes:    ungroupedChanges,
		})
	}

	return entries, nil
}

func parseReleaseBody(body string) ([]Section, []string) {
	var sections []Section
	var ungroupedChanges []string

	headerRegex := regexp.MustCompile(`^#{1,3}\s+(.+)$`)
	lines := strings.Split(body, "\n")

	var currentSection *Section

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for section header (# ## or ###)
		if match := headerRegex.FindStringSubmatch(trimmed); match != nil {
			headerName := strings.TrimSpace(match[1])
			// Skip "What's Changed" as it's just a wrapper, not a real category
			if headerName == "What's Changed" {
				continue
			}
			// Save previous section if exists
			if currentSection != nil && len(currentSection.Changes) > 0 {
				sections = append(sections, *currentSection)
			}
			currentSection = &Section{Name: headerName}
			continue
		}

		// Check for list item
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			change := strings.TrimPrefix(trimmed, "- ")
			change = strings.TrimPrefix(change, "* ")
			if change != "" && !strings.HasPrefix(change, "@") {
				if currentSection != nil {
					currentSection.Changes = append(currentSection.Changes, change)
				} else {
					ungroupedChanges = append(ungroupedChanges, change)
				}
			}
		}
	}

	// Don't forget the last section
	if currentSection != nil && len(currentSection.Changes) > 0 {
		sections = append(sections, *currentSection)
	}

	return sections, ungroupedChanges
}
