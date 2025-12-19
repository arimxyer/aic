package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var version = "dev"

type Section struct {
	Name    string   `json:"name"`
	Changes []string `json:"changes"`
}

type ChangelogEntry struct {
	Version  string    `json:"version"`
	Sections []Section `json:"sections,omitempty"`
	Changes  []string  `json:"changes,omitempty"`
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
	fmt.Fprintf(os.Stderr, "Usage: aic <source> [flags]\n\n")
	fmt.Fprintf(os.Stderr, "Sources:\n")
	fmt.Fprintf(os.Stderr, "  claude      Claude Code (Anthropic)\n")
	fmt.Fprintf(os.Stderr, "  codex       Codex CLI (OpenAI)\n")
	fmt.Fprintf(os.Stderr, "  opencode    OpenCode (SST)\n")
	fmt.Fprintf(os.Stderr, "  gemini      Gemini CLI (Google)\n")
	fmt.Fprintf(os.Stderr, "  copilot     Copilot CLI (GitHub)\n\n")
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
}

func fetchClaudeChangelog() ([]ChangelogEntry, error) {
	url := "https://raw.githubusercontent.com/anthropics/claude-code/main/CHANGELOG.md"
	content, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	return parseMarkdownChangelog(content, `(?m)^## (\d+\.\d+\.\d+)\s*$`), nil
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
	return parseMarkdownChangelog(content, `(?m)^## ([\d.]+) - \d{4}-\d{2}-\d{2}\s*$`), nil
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
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
		Body    string `json:"body"`
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

		entries = append(entries, ChangelogEntry{
			Version:  ver,
			Sections: sections,
			Changes:  ungroupedChanges,
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
	fmt.Printf("## %s\n\n", entry.Version)

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
	fmt.Printf("%s %s\n", displayName, entry.Version)
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
