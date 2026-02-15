package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"
)

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
