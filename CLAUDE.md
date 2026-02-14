# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development

```bash
mise run build          # Build binary → ./aic
mise run test           # Run all tests (go test ./...)
mise run run -- <args>  # Run without building (go run . <args>)
```

Releases are handled by GoReleaser via GitHub Actions on version tags (`v*`). The `main.version` variable is set at build time via `-X main.version={{.Version}}` ldflags.

## Architecture

This is a single-file Go CLI (`main.go`, no external dependencies) that fetches changelogs for AI coding assistants from the GitHub Releases API.

### Core Data Model

- **`Source`** — defines a tracked tool (display name, GitHub owner/repo, binary names for local detection, version args). All sources are registered in the `sources` map.
- **`ChangelogEntry`** — a parsed release with version, date, sectioned changes, and ungrouped changes.
- **`Section`** — a named group of changes within a release (parsed from markdown headers in release bodies).

### Key Functions

- **`fetchGitHubReleases(owner, repo)`** — single GitHub API integration point; all sources use this.
- **`parseReleaseBody(body)`** — converts markdown release notes into structured `Section`/changes. Skips "What's Changed" headers and `@`-prefixed lines (contributor mentions).
- **`Source.DetectInstalled(ctx)`** — checks if a tool is locally installed via `exec.LookPath` + version command with 3s timeout.

### Commands

The CLI uses manual arg parsing (no framework). Three command paths:
1. **`aic <source> [flags]`** — fetch changelog for a specific source
2. **`aic latest`** — all releases from last 24h across all sources (concurrent fetching)
3. **`aic status`** — table view with versions, recency, installed status, release frequency (concurrent fetching + detection)

### Adding a New Source

Add an entry to the `sources` map in `main.go`. All sources use the same `fetchGitHubReleases` path, so only the GitHub owner/repo and binary detection info are needed. Update `printUsage()` to include the new source.
