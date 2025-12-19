package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var version = "dev"

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

type Source struct {
	Name        string
	DisplayName string
	FetchFunc   func() ([]ChangelogEntry, error)
}

var sources = map[string]Source{
	"claude": {
		Name:        "claude",
		DisplayName: "Claude Code",
		FetchFunc:   fetchClaudeChangelog,
	},
	"codex": {
		Name:        "codex",
		DisplayName: "OpenAI Codex",
		FetchFunc:   fetchCodexChangelog,
	},
	"opencode": {
		Name:        "opencode",
		DisplayName: "OpenCode",
		FetchFunc:   fetchOpenCodeChangelog,
	},
	"gemini": {
		Name:        "gemini",
		DisplayName: "Gemini CLI",
		FetchFunc:   fetchGeminiChangelog,
	},
	"copilot": {
		Name:        "copilot",
		DisplayName: "GitHub Copilot CLI",
		FetchFunc:   fetchCopilotChangelog,
	},
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		printUsage()
		os.Exit(0)
	}

	if args[0] == "-v" || args[0] == "--version" {
		fmt.Printf("aic version %s\n", version)
		os.Exit(0)
	}

	if args[0] == "list-sources" {
		for name, src := range sources {
			fmt.Printf("  %s\t%s\n", name, src.DisplayName)
		}
		os.Exit(0)
	}

	if args[0] == "latest" {
		var jsonOutput bool
		for i := 1; i < len(args); i++ {
			if args[i] == "-json" || args[i] == "--json" {
				jsonOutput = true
			}
		}
		runLatestCommand(jsonOutput)
		os.Exit(0)
	}

	sourceName := args[0]
	source, ok := sources[sourceName]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: Unknown source '%s'\n\n", sourceName)
		fmt.Fprintf(os.Stderr, "Available sources:\n")
		for name := range sources {
			fmt.Fprintf(os.Stderr, "  %s\n", name)
		}
		os.Exit(1)
	}

	var jsonOutput, mdOutput, listVersions bool
	var targetVersion string

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-json", "--json":
			jsonOutput = true
		case "-md", "--md":
			mdOutput = true
		case "-list", "--list":
			listVersions = true
		case "-version", "--version":
			if i+1 < len(args) {
				targetVersion = args[i+1]
				i++
			}
		}
	}

	entries, err := source.FetchFunc()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching changelog: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Fprintf(os.Stderr, "Error: No changelog entries found\n")
		os.Exit(1)
	}

	if listVersions {
		for _, entry := range entries {
			fmt.Println(entry.Version)
		}
		os.Exit(0)
	}

	var entry *ChangelogEntry
	if targetVersion != "" {
		for i := range entries {
			if entries[i].Version == targetVersion {
				entry = &entries[i]
				break
			}
		}
		if entry == nil {
			fmt.Fprintf(os.Stderr, "Error: Version %s not found\n", targetVersion)
			os.Exit(1)
		}
	} else {
		entry = &entries[0]
	}

	if jsonOutput {
		outputJSON(entry)
	} else if mdOutput {
		outputMarkdown(entry)
	} else {
		outputPlainText(source.DisplayName, entry)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "aic - AI Coding Agent Changelog Viewer\n\n")
	fmt.Fprintf(os.Stderr, "Usage: aic <source> [flags]\n")
	fmt.Fprintf(os.Stderr, "       aic latest [flags]\n\n")
	fmt.Fprintf(os.Stderr, "Sources:\n")
	fmt.Fprintf(os.Stderr, "  claude      Claude Code (Anthropic)\n")
	fmt.Fprintf(os.Stderr, "  codex       Codex CLI (OpenAI)\n")
	fmt.Fprintf(os.Stderr, "  opencode    OpenCode (SST)\n")
	fmt.Fprintf(os.Stderr, "  gemini      Gemini CLI (Google)\n")
	fmt.Fprintf(os.Stderr, "  copilot     Copilot CLI (GitHub)\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  latest             Show releases from all sources in last 24h\n\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	fmt.Fprintf(os.Stderr, "  -json              Output as JSON\n")
	fmt.Fprintf(os.Stderr, "  -md                Output as markdown\n")
	fmt.Fprintf(os.Stderr, "  -list              List all versions\n")
	fmt.Fprintf(os.Stderr, "  -version <ver>     Get specific version\n")
	fmt.Fprintf(os.Stderr, "  -v, --version      Show aic version\n")
	fmt.Fprintf(os.Stderr, "  -h, --help         Show this help\n\n")
	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "  aic claude                    # Latest Claude Code entry\n")
	fmt.Fprintf(os.Stderr, "  aic codex -json               # Latest Codex entry as JSON\n")
	fmt.Fprintf(os.Stderr, "  aic opencode -list            # List OpenCode versions\n")
	fmt.Fprintf(os.Stderr, "  aic gemini -version 0.21.0    # Specific Gemini version\n")
	fmt.Fprintf(os.Stderr, "  aic latest                    # All releases in last 24h\n")
}

func runLatestCommand(jsonOutput bool) {
	cutoff := time.Now().Add(-24 * time.Hour)

	type result struct {
		source  string
		display string
		entry   *ChangelogEntry
		err     error
	}

	results := make(chan result, len(sources))
	var wg sync.WaitGroup

	for name, src := range sources {
		wg.Add(1)
		go func(name string, src Source) {
			defer wg.Done()
			entries, err := src.FetchFunc()
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
			outputPlainText(entry.Source, &entry)
		}
	}
}

func fetchClaudeChangelog() ([]ChangelogEntry, error) {
	url := "https://raw.githubusercontent.com/anthropics/claude-code/main/CHANGELOG.md"
	content, err := httpGet(url)
	if err != nil {
		return nil, err
	}

	entries := parseMarkdownChangelog(content, `(?m)^## (\d+\.\d+\.\d+)\s*$`)

	// Fetch last commit date for the changelog file
	if len(entries) > 0 {
		commitDate := fetchGitHubFileLastCommitDate("anthropics", "claude-code", "CHANGELOG.md")
		if !commitDate.IsZero() {
			entries[0].ReleasedAt = commitDate
		}
	}

	return entries, nil
}

func fetchGitHubFileLastCommitDate(owner, repo, path string) time.Time {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits?path=%s&per_page=1", owner, repo, path)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return time.Time{}
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "aic-changelog")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return time.Time{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return time.Time{}
	}

	var commits []struct {
		Commit struct {
			Committer struct {
				Date string `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil || len(commits) == 0 {
		return time.Time{}
	}

	t, _ := time.Parse(time.RFC3339, commits[0].Commit.Committer.Date)
	return t
}

func fetchCodexChangelog() ([]ChangelogEntry, error) {
	return fetchGitHubReleases("openai", "codex")
}

func fetchOpenCodeChangelog() ([]ChangelogEntry, error) {
	return fetchGitHubReleases("sst", "opencode")
}

func fetchGeminiChangelog() ([]ChangelogEntry, error) {
	return fetchGitHubReleases("google-gemini", "gemini-cli")
}

func fetchCopilotChangelog() ([]ChangelogEntry, error) {
	url := "https://raw.githubusercontent.com/github/copilot-cli/main/changelog.md"
	content, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	return parseMarkdownChangelogWithDate(content, `(?m)^## ([\d.]+) - (\d{4}-\d{2}-\d{2})\s*$`), nil
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

func parseMarkdownChangelog(content, versionPattern string) []ChangelogEntry {
	var entries []ChangelogEntry

	versionRegex := regexp.MustCompile(versionPattern)
	matches := versionRegex.FindAllStringSubmatchIndex(content, -1)

	for i, match := range matches {
		versionEnd := match[1]
		ver := content[match[2]:match[3]]

		var contentEnd int
		if i+1 < len(matches) {
			contentEnd = matches[i+1][0]
		} else {
			contentEnd = len(content)
		}

		sectionContent := content[versionEnd:contentEnd]
		changes := parseChanges(sectionContent)

		entries = append(entries, ChangelogEntry{
			Version: ver,
			Changes: changes,
		})
	}

	return entries
}

func parseMarkdownChangelogWithDate(content, versionPattern string) []ChangelogEntry {
	var entries []ChangelogEntry

	versionRegex := regexp.MustCompile(versionPattern)
	matches := versionRegex.FindAllStringSubmatch(content, -1)
	matchIndexes := versionRegex.FindAllStringSubmatchIndex(content, -1)

	for i, match := range matches {
		ver := match[1]
		dateStr := match[2]

		releasedAt, _ := time.Parse("2006-01-02", dateStr)

		var contentEnd int
		if i+1 < len(matchIndexes) {
			contentEnd = matchIndexes[i+1][0]
		} else {
			contentEnd = len(content)
		}

		sectionContent := content[matchIndexes[i][1]:contentEnd]
		changes := parseChanges(sectionContent)

		entries = append(entries, ChangelogEntry{
			Version:    ver,
			ReleasedAt: releasedAt,
			Changes:    changes,
		})
	}

	return entries
}

func parseChanges(content string) []string {
	var changes []string
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") {
			change := strings.TrimPrefix(trimmed, "- ")
			changes = append(changes, change)
		}
	}
	return changes
}

func httpGet(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), nil
}

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
