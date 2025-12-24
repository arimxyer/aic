# aic

Fetch the latest changelogs for popular AI coding assistants.

https://github.com/user-attachments/assets/061edd6a-f723-49d4-a0c2-b9a8e8ba9a16

## Supported Tools

| Source | Command | Tool |
|--------|---------|------|
| `claude` | `aic claude` | [Claude Code](https://github.com/anthropics/claude-code) (Anthropic) |
| `codex` | `aic codex` | [Codex CLI](https://github.com/openai/codex) (OpenAI) |
| `opencode` | `aic opencode` | [OpenCode](https://github.com/sst/opencode) (SST) |
| `gemini` | `aic gemini` | [Gemini CLI](https://github.com/google-gemini/gemini-cli) (Google) |
| `copilot` | `aic copilot` | [Copilot CLI](https://github.com/github/copilot-cli) (GitHub) |

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
```

### Examples

```bash
aic claude                    # Latest Claude Code changelog
aic codex -json               # Latest Codex changelog as JSON
aic opencode -list            # List all OpenCode versions
aic gemini -version 0.1.0     # Specific Gemini CLI version
aic copilot -md               # Latest Copilot changelog as markdown
aic latest                    # All releases from last 24 hours
aic latest -json              # Recent releases as JSON
```

## Commands

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
| `-version <ver>` | Fetch specific version |
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
