# aic

Fetch the latest changelogs for popular AI coding assistants.

https://github.com/user-attachments/assets/061edd6a-f723-49d4-a0c2-b9a8e8ba9a16

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

> **Want to add another tool?** Missing your favorite AI coding assistant? [Open an issue](https://github.com/arimxyer/aic/issues) or [submit a PR](https://github.com/arimxyer/aic/pulls)!

## Installation

### Homebrew (macOS/Linux)

```bash
brew install arimxyer/tap/aic
```

### Scoop (Windows)

```bash
scoop bucket add arimxyer https://github.com/arimxyer/scoop-bucket
scoop install aic
```

### Go

```bash
go install github.com/arimxyer/aic@latest
```

### From source

```bash
git clone https://github.com/arimxyer/aic
cd aic
go build -o aic
```

## Usage

```bash
aic <source> [flags]
aic latest [flags]
aic status [flags]
```

### Examples

```bash
aic claude                    # Latest Claude Code changelog
aic codex -json               # Latest Codex changelog as JSON
aic opencode -list            # List all OpenCode versions
aic gemini -version 0.1.0     # Specific Gemini CLI version
aic copilot -md               # Latest Copilot changelog as markdown
aic latest                    # All releases from last 24 hours
aic status                    # Status table of all tools
aic claude -web               # Open Claude changelog in browser
```

## Commands

### `aic status`

Show a status table of all tools with version info, update recency, and release frequency.

```
$ aic status
┌──────────────────────┬─────┬──────────────┬──────────────┬────────────┬─────────────────────┐
│ Tool                 │ 24h │ Version      │ Previous     │ Updated    │ Vers. Release Freq. │
├──────────────────────┼─────┼──────────────┼──────────────┼────────────┼─────────────────────┤
│ OpenAI Codex         │ [✓] │ 0.92.0       │ 0.92.0-al... │ 6h ago     │ ~3h                 │
│ Claude Code          │ [✓] │ 2.1.20       │ 2.1.19       │ 14h ago    │ -                   │
│ Gemini CLI           │ [✓] │ 0.27.0-ni... │ 0.27.0-ni... │ 17h ago    │ ~15h                │
│ GitHub Copilot CLI   │     │ 0.0.395      │ 0.0.394      │ 1d ago     │ ~18h                │
│ OpenCode             │     │ 1.1.36       │ 1.1.35       │ 1d ago     │ ~13h                │
└──────────────────────┴─────┴──────────────┴──────────────┴────────────┴─────────────────────┘
```

- **24h**: Shows `[✓]` if updated in the last 24 hours
- **Version/Previous**: Current and previous version numbers
- **Updated**: Relative time since last release
- **Vers. Release Freq.**: Average time between releases (calculated from last 10 releases)

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

## Flags

| Flag | Description |
|------|-------------|
| `-json` | Output as JSON |
| `-md` | Output as markdown |
| `-list` | List all available versions |
| `-pick` | Interactive version picker |
| `-version <ver>` | Fetch specific version |
| `-web` | Open changelog source in browser |
| `-v` | Show aic version |
| `-h` | Show help |

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
$ aic opencode -json
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
$ aic opencode -list
1.0.170
1.0.169
1.0.168
...
```

## License

MIT
