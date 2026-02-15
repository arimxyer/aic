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

Go CLI split across multiple files, using the Charm ecosystem (lipgloss, bubbletea, glamour) for terminal UI. Fetches changelogs for AI coding assistants from the GitHub Releases API.

### File Structure

| File | Purpose |
|------|---------|
| `main.go` | Entry point, arg parsing, `printUsage()` |
| `sources.go` | `Source` type, `sources` map, config (load/save/XDG), `enabledSources()` |
| `fetch.go` | `ChangelogEntry`/`Section` types, `fetchGitHubReleases()`, `parseReleaseBody()` |
| `output.go` | Output formatters: `outputJSON()`, `outputMarkdown()`, `outputPlainText()`, `outputRendered()` (glamour) |
| `status.go` | `runStatusCommand()` — lipgloss table with concurrent fetch + install detection |
| `latest.go` | `runLatestCommand()` — concurrent fetch, 24h filter |
| `config.go` | `runConfigCommand()` — bubbletea interactive source picker |
| `helpers.go` | `formatRelativeTime()`, `calculateAvgReleaseFreq()`, `openBrowser()`, `truncateString()` |

### Charm Libraries

- **Lipgloss** (`lipgloss/table`) — styled status table in `status.go`
- **Glamour** — markdown rendering for changelogs in `output.go` (`outputRendered`). Auto-detects TTY; falls back to plain text when piped.
- **Bubbletea** — interactive config picker in `config.go` (Model/Update/View pattern)

### Key Data Flow

- All sources use `fetchGitHubReleases(owner, repo)` → GitHub Releases API
- `ChangelogEntry.RawBody` stores the raw markdown for glamour rendering (excluded from JSON via `json:"-"`)
- `parseReleaseBody()` creates structured `Sections`/`Changes` for JSON output
- User config at `~/.config/aic/config.json` (XDG-aware) stores disabled sources

### Commands

Manual arg parsing (no framework). Four command paths:
1. **`aic <source> [flags]`** — fetch changelog for a specific source
2. **`aic latest`** — all releases from last 24h across enabled sources (concurrent)
3. **`aic status`** — table view with versions, recency, installed status, release frequency
4. **`aic config`** — interactive picker to enable/disable sources

### Adding a New Source

Add an entry to the `sources` map in `sources.go`. All sources use `fetchGitHubReleases`, so only GitHub owner/repo and binary detection info are needed. Update `printUsage()` in `main.go` and the README table.
