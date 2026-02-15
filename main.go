package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
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
	DisplayName string
	Owner       string
	Repo        string
	BinaryNames []string
	VersionArgs []string
}

type InstalledInfo struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
	Binary    string `json:"binary,omitempty"`
}

var semverRegex = regexp.MustCompile(`(\d+\.\d+\.\d+(?:-[a-zA-Z0-9.]+)?)`)

func (s Source) DetectInstalled(ctx context.Context) InstalledInfo {
	for _, bin := range s.BinaryNames {
		path, err := exec.LookPath(bin)
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, path, s.VersionArgs...)
		out, _ := cmd.CombinedOutput()
		ver := semverRegex.FindString(string(out))
		return InstalledInfo{Installed: true, Version: ver, Binary: bin}
	}
	return InstalledInfo{}
}

func (s Source) URL() string {
	return fmt.Sprintf("https://github.com/%s/%s/releases", s.Owner, s.Repo)
}

func (s Source) Fetch() ([]ChangelogEntry, error) {
	return fetchGitHubReleases(s.Owner, s.Repo)
}

var sources = map[string]Source{
	"claude":   {DisplayName: "Claude Code", Owner: "anthropics", Repo: "claude-code", BinaryNames: []string{"claude"}, VersionArgs: []string{"--version"}},
	"codex":    {DisplayName: "OpenAI Codex", Owner: "openai", Repo: "codex", BinaryNames: []string{"codex"}, VersionArgs: []string{"--version"}},
	"opencode": {DisplayName: "OpenCode", Owner: "anomalyco", Repo: "opencode", BinaryNames: []string{"opencode"}, VersionArgs: []string{"--version"}},
	"gemini":   {DisplayName: "Gemini CLI", Owner: "google-gemini", Repo: "gemini-cli", BinaryNames: []string{"gemini"}, VersionArgs: []string{"--version"}},
	"copilot":  {DisplayName: "GitHub Copilot CLI", Owner: "github", Repo: "copilot-cli", BinaryNames: []string{"github-copilot-cli", "copilot"}, VersionArgs: []string{"--version"}},
	"openclaw": {DisplayName: "OpenClaw", Owner: "openclaw", Repo: "openclaw", BinaryNames: []string{"openclaw"}, VersionArgs: []string{"--version"}},
	"kimi":     {DisplayName: "Kimi CLI", Owner: "MoonshotAI", Repo: "kimi-cli", BinaryNames: []string{"kimi"}, VersionArgs: []string{"--version"}},
	"qwen":     {DisplayName: "Qwen Code", Owner: "QwenLM", Repo: "qwen-code", BinaryNames: []string{"qwen"}, VersionArgs: []string{"--version"}},
	"goose":    {DisplayName: "Goose", Owner: "block", Repo: "goose", BinaryNames: []string{"goose"}, VersionArgs: []string{"--version"}},
}

type Config struct {
	DisabledSources []string `json:"disabled_sources"`
}

func configPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "aic", "config.json")
}

func loadConfig() Config {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return Config{}
	}
	var cfg Config
	json.Unmarshal(data, &cfg)
	return cfg
}

func saveConfig(cfg Config) error {
	p := configPath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

func enabledSources() map[string]Source {
	cfg := loadConfig()
	disabled := make(map[string]bool)
	for _, s := range cfg.DisabledSources {
		disabled[s] = true
	}
	result := make(map[string]Source)
	for name, src := range sources {
		if !disabled[name] {
			result[name] = src
		}
	}
	return result
}

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

	if args[0] == "config" {
		runConfigCommand()
		os.Exit(0)
	}

	if args[0] == "list-sources" {
		for name, src := range sources {
			fmt.Printf("  %s\t%s\n", name, src.DisplayName)
		}
		os.Exit(0)
	}

	if args[0] == "latest" {
		var jsonOutput, webOpen bool
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "-json", "--json":
				jsonOutput = true
			case "-web", "--web":
				webOpen = true
			}
		}
		if webOpen {
			for _, src := range enabledSources() {
				openBrowser(src.URL())
			}
			os.Exit(0)
		}
		runLatestCommand(jsonOutput)
		os.Exit(0)
	}

	if args[0] == "status" {
		var jsonOutput, webOpen bool
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "-json", "--json":
				jsonOutput = true
			case "-web", "--web":
				webOpen = true
			}
		}
		if webOpen {
			for _, src := range enabledSources() {
				openBrowser(src.URL())
			}
			os.Exit(0)
		}
		runStatusCommand(jsonOutput)
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

	var jsonOutput, mdOutput, listVersions, webOpen bool
	var targetVersion string

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-json", "--json":
			jsonOutput = true
		case "-md", "--md":
			mdOutput = true
		case "-list", "--list":
			listVersions = true
		case "-web", "--web":
			webOpen = true
		case "-version", "--version":
			if i+1 < len(args) {
				targetVersion = args[i+1]
				i++
			}
		}
	}

	if webOpen {
		openBrowser(source.URL())
		os.Exit(0)
	}

	entries, err := source.Fetch()
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
	fmt.Fprintf(os.Stderr, "       aic latest [flags]\n")
	fmt.Fprintf(os.Stderr, "       aic status [flags]\n\n")
	fmt.Fprintf(os.Stderr, "Sources:\n")
	fmt.Fprintf(os.Stderr, "  claude      Claude Code (Anthropic)\n")
	fmt.Fprintf(os.Stderr, "  codex       Codex CLI (OpenAI)\n")
	fmt.Fprintf(os.Stderr, "  opencode    OpenCode\n")
	fmt.Fprintf(os.Stderr, "  gemini      Gemini CLI (Google)\n")
	fmt.Fprintf(os.Stderr, "  copilot     Copilot CLI (GitHub)\n")
	fmt.Fprintf(os.Stderr, "  openclaw    OpenClaw\n")
	fmt.Fprintf(os.Stderr, "  kimi        Kimi CLI (Moonshot AI)\n")
	fmt.Fprintf(os.Stderr, "  qwen        Qwen Code (Alibaba)\n")
	fmt.Fprintf(os.Stderr, "  goose       Goose (Block)\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  latest             Show releases from all sources in last 24h\n")
	fmt.Fprintf(os.Stderr, "  status             Show status table of all sources\n")
	fmt.Fprintf(os.Stderr, "  config             Configure which sources to show\n\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	fmt.Fprintf(os.Stderr, "  -json              Output as JSON\n")
	fmt.Fprintf(os.Stderr, "  -md                Output as markdown\n")
	fmt.Fprintf(os.Stderr, "  -list              List all versions\n")
	fmt.Fprintf(os.Stderr, "  -version <ver>     Get specific version\n")
	fmt.Fprintf(os.Stderr, "  -web               Open changelog source in browser\n")
	fmt.Fprintf(os.Stderr, "  -v, --version      Show aic version\n")
	fmt.Fprintf(os.Stderr, "  -h, --help         Show this help\n\n")
	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "  aic claude                    # Latest Claude Code entry\n")
	fmt.Fprintf(os.Stderr, "  aic codex -json               # Latest Codex entry as JSON\n")
	fmt.Fprintf(os.Stderr, "  aic opencode -list            # List OpenCode versions\n")
	fmt.Fprintf(os.Stderr, "  aic gemini -version 0.21.0    # Specific Gemini version\n")
	fmt.Fprintf(os.Stderr, "  aic latest                    # All releases in last 24h\n")
	fmt.Fprintf(os.Stderr, "  aic status                    # Status table of all tools\n")
	fmt.Fprintf(os.Stderr, "  aic claude -web               # Open Claude changelog in browser\n")
	fmt.Fprintf(os.Stderr, "  aic status -web               # Open all changelogs in browser\n")
}

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
			outputPlainText(entry.Source, &entry)
		}
	}
}

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
