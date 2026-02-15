package main

import (
	"fmt"
	"os"
)

var version = "dev"

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
