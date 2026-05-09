# aic

Fetch the latest changelogs for popular AI coding assistants.

https://github.com/user-attachments/assets/68f8b2fc-b6f5-4e6f-a743-f1f31a99dee7

## Supported Tools

| Source | Command | Tool |
|--------|---------|------|
| `claude` | `aic claude` | [Claude Code](https://github.com/anthropics/claude-code) (Anthropic) |
| `codex` | `aic codex` | [Codex CLI](https://github.com/openai/codex) (OpenAI) |
| `opencode` | `aic opencode` | [OpenCode](https://github.com/anomalyco/opencode) |
| `gemini` | `aic gemini` | [Gemini CLI](https://github.com/google-gemini/gemini-cli) (Google) |
| `copilot` | `aic copilot` | [Copilot CLI](https://github.com/github/copilot-cli) (GitHub) |
| `openclaw` | `aic openclaw` | [OpenClaw](https://github.com/openclaw/openclaw) |
| `kimi` | `aic kimi` | [Kimi CLI](https://github.com/MoonshotAI/kimi-cli) (Moonshot AI) |
| `qwen` | `aic qwen` | [Qwen Code](https://github.com/QwenLM/qwen-code) (Alibaba) |
| `goose` | `aic goose` | [Goose](https://github.com/block/goose) (Block) |

> **Want to add another tool?** Missing your favorite AI coding assistant? [Open an issue](https://github.com/reyamira/aic/issues) or [submit a PR](https://github.com/reyamira/aic/pulls)!

## Installation

### Homebrew (macOS/Linux)

```bash
brew install reyamira/tap/aic
```

### Scoop (Windows)

```bash
scoop bucket add reyamira https://github.com/reyamira/scoop-bucket
scoop install aic
```

### Go

```bash
go install github.com/arimxyer/aic@latest
```

### From source

```bash
git clone https://github.com/reyamira/aic
cd aic
go build -o aic
```

## Usage

```bash
aic <source> [flags]
aic latest [flags]
aic status [flags]
aic config
```

### Examples

```bash
aic claude                      # Latest Claude Code changelog
aic codex --json                # Latest Codex changelog as JSON
aic opencode --list             # List all OpenCode versions
aic gemini --version 0.1.0      # Specific Gemini CLI version
aic copilot --md                # Latest Copilot changelog as markdown
aic claude --pick               # Interactive version picker
aic latest                      # All releases from last 24 hours
aic status                      # Status table of all tools
aic claude --web                # Open Claude changelog in browser
```

## Commands

### `aic status`

Show a status table of all tools with version info, update recency, and release frequency.

```
$ aic status
┌────────────────────┬─────┬───────────┬──────────┬─────────┬───────────────┐
│ Tool               │ 24h │ Installed │ Latest   │ Updated │ Release Freq. │
├────────────────────┼─────┼───────────┼──────────┼─────────┼───────────────┤
│ Claude Code        │ ✓   │ 2.1.42    │ 2.1.42   │ 1d ago  │ ~1d           │
│ OpenAI Codex       │ ✓   │ 0.92.0    │ 0.92.0   │ 6h ago  │ ~3h           │
│ Gemini CLI         │     │ 0.27.0    │ 0.28.0   │ 2d ago  │ ~15h          │
│ OpenCode           │     │ -         │ 1.1.36   │ 3d ago  │ ~13h          │
└────────────────────┴─────┴───────────┴──────────┴─────────┴───────────────┘
```

- **24h**: Shows `✓` if updated in the last 24 hours
- **Installed**: Locally installed version (detected from PATH), or `-` if not found
- **Latest**: Most recent release version
- **Updated**: Relative time since last release
- **Release Freq.**: Average time between releases (calculated from last 10 releases)

### `aic latest`

Show releases from all sources in the last 24 hours, sorted by release date (newest first).

```
$ aic latest
OpenAI Codex 0.76.0 (2025-12-19)
----------------------------------------

[New Features]
  * Add a macOS DMG build target
  * Add /ps command
  ...

OpenCode 1.0.170 (2025-12-19)
----------------------------------------

[TUI]
  * User messages as markdown with toggle
  ...

Claude Code 2.0.73 (2025-12-19)
----------------------------------------
  * Added clickable `[Image #N]` links
  ...
```

### `aic config`

Interactive picker to enable/disable sources. Uses arrow keys to navigate, space to toggle, enter to save.

## Flags

### Source flags (`aic <source>`)

| Short | Long | Description |
|-------|------|-------------|
| `-j` | `--json` | Output as JSON |
| `-m` | `--md` | Output as markdown |
| `-l` | `--list` | List all available versions |
| `-p` | `--pick` | Interactive version picker |
| | `--version <ver>` | Fetch specific version |
| `-w` | `--web` | Open in browser |

### Command flags (`latest`, `status`)

| Short | Long | Description |
|-------|------|-------------|
| `-j` | `--json` | Output as JSON |
| `-w` | `--web` | Open in browser |

### Global

| Short | Long | Description |
|-------|------|-------------|
| `-v` | `--version` | Show aic version |
| `-h` | `--help` | Show help |

## Output Examples

### Plain text (default)

Output includes release date and section headers (when available):

```
$ aic opencode
OpenCode 1.0.170 (2025-12-19)
----------------------------------------

[TUI]
  * User messages as markdown with toggle
  * Implement smooth scrolling for autocomplete dropdown

[Desktop]
  * Fixed error handling
  * Separate prompt history for shell
```

### JSON output

```
$ aic opencode --json
{
  "version": "1.0.170",
  "released_at": "2025-12-19T15:30:00Z",
  "sections": [
    {
      "name": "TUI",
      "changes": [
        "User messages as markdown with toggle",
        "Implement smooth scrolling..."
      ]
    },
    {
      "name": "Desktop",
      "changes": [
        "Fixed error handling",
        "Separate prompt history for shell"
      ]
    }
  ]
}
```

### List versions

```
$ aic opencode --list
1.0.170
1.0.169
1.0.168
...
```

## License

MIT
