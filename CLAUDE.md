# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development

```bash
mise run build          # Build binary → ./aic
mise run test           # Run all tests (go test ./...)
mise run run <args>     # Run without building (go run . <args>)
```

Releases are handled by GoReleaser via GitHub Actions on version tags (`v*`). The `main.version` variable is set at build time via `-X main.version={{.Version}}` ldflags.

To release: `git tag v<X.Y.Z> && git push origin v<X.Y.Z>`

## Architecture

Go CLI built with **Cobra** for command/flag handling, using the Charm ecosystem (lipgloss, bubbletea, glamour) for terminal UI. Fetches changelogs for AI coding assistants from the GitHub Releases API.

### File Structure

| File | Purpose |
|------|---------|
| `main.go` | Cobra command definitions, flag registration, `createSourceCommand()` |
| `sources.go` | `Source` type, `sources` map, config (load/save/XDG), `enabledSources()` |
| `fetch.go` | `ChangelogEntry`/`Section` types, `fetchGitHubReleases()`, `parseReleaseBody()` |
| `output.go` | Output formatters: `outputJSON()`, `outputMarkdown()`, `outputPlainText()`, `outputRendered()` (glamour), `outputVersionList()` (lipgloss table) |
| `status.go` | `runStatusCommand()` — lipgloss table with concurrent fetch + install detection |
| `latest.go` | `runLatestCommand()` — concurrent fetch, 24h filter |
| `config.go` | `runConfigCommand()` — bubbletea interactive source picker |
| `pick.go` | `runPickCommand()` — bubbletea interactive version picker |
| `helpers.go` | `formatRelativeTime()`, `calculateAvgReleaseFreq()`, `openBrowser()`, `truncateString()` |

### Key Libraries

- **Cobra** — command routing, flag parsing with short/long aliases, mutual exclusivity, grouped help
- **Lipgloss** (`lipgloss/table`) — styled tables in `status.go` and `output.go`
- **Glamour** — markdown rendering for changelogs in `output.go`. Auto-detects TTY; falls back to plain text when piped.
- **Bubbletea** — interactive TUI for config picker (`config.go`) and version picker (`pick.go`)

### Key Data Flow

- All sources use `fetchGitHubReleases(owner, repo)` → GitHub Releases API
- `ChangelogEntry.RawBody` stores the raw markdown for glamour rendering (excluded from JSON via `json:"-"`)
- `parseReleaseBody()` creates structured `Sections`/`Changes` for JSON output
- User config at `~/.config/aic/config.json` (XDG-aware) stores disabled sources

### Commands (Cobra)

Commands are defined in `main.go` with Cobra. Source commands are dynamically registered from the `sources` map in `init()`.

| Command | Flags |
|---------|-------|
| `aic <source>` | `-j/--json`, `-m/--md`, `-l/--list`, `-p/--pick`, `--version <ver>`, `-w/--web` |
| `aic latest` | `-j/--json`, `-w/--web` |
| `aic status` | `-j/--json`, `-w/--web` |
| `aic config` | (none) |

Mutually exclusive flags on source commands: `--json`/`--md`, `--list`/`--pick`, `--list`/`--version`, `--pick`/`--version`.

### Adding a New Source

Add an entry to the `sources` map in `sources.go`. All sources use `fetchGitHubReleases`, so only GitHub owner/repo and binary detection info are needed. Cobra automatically registers the new source as a subcommand via the `init()` loop. Update the README supported tools table.
