package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds user preferences loaded from ~/.config/approver/config.yaml.
type Config struct {
	// WorktreeDir overrides the default worktree base directory.
	WorktreeDir string `yaml:"worktree_dir,omitempty"`

	// PRLimit is the max number of PRs to fetch (default: 50).
	PRLimit int `yaml:"pr_limit,omitempty"`

	// ReviewPrompt overrides the default Agent 1 review prompt.
	ReviewPrompt string `yaml:"review_prompt,omitempty"`

	// AllowedTools is the set of tools review agents can use (default: "Read,Glob,Grep").
	AllowedTools string `yaml:"allowed_tools,omitempty"`

	// PollInterval is the number of seconds between background PR refreshes.
	// Default: 300 (5 minutes). Set to 0 to disable polling.
	PollInterval *int `yaml:"poll_interval,omitempty"`

	// Repos holds per-repository configuration keyed by a substring of the remote URL.
	Repos map[string]RepoConfig `yaml:"repos,omitempty"`
}

// RepoConfig holds configuration for a specific repository.
type RepoConfig struct {
	// SetupCommand is run in the worktree after creation (e.g. "npm install").
	SetupCommand string `yaml:"setup_command,omitempty"`
}

// RepoSetupCommand returns the setup command for a repo whose remote URL
// contains the given repoURL substring. Returns empty string if none match.
func (c *Config) RepoSetupCommand(repoURL string) string {
	if c == nil || len(c.Repos) == 0 || repoURL == "" {
		return ""
	}
	for pattern, rc := range c.Repos {
		if strings.Contains(repoURL, pattern) {
			return rc.SetupCommand
		}
	}
	return ""
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	defaultPoll := 300
	defaultLimit := 50
	return &Config{
		PRLimit:      defaultLimit,
		PollInterval: &defaultPoll,
		AllowedTools: "Read,Glob,Grep",
	}
}

// configPath returns the path to the config file.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "approver", "config.yaml"), nil
}

// LoadConfig reads config from disk, falling back to defaults.
func LoadConfig() *Config {
	cfg := DefaultConfig()

	path, err := configPath()
	if err != nil {
		return cfg
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return DefaultConfig()
	}

	// Apply defaults for unset values
	if cfg.PollInterval == nil {
		defaultPoll := 300
		cfg.PollInterval = &defaultPoll
	}
	if cfg.PRLimit <= 0 {
		cfg.PRLimit = 50
	}
	if cfg.AllowedTools == "" {
		cfg.AllowedTools = "Read,Glob,Grep"
	}

	return cfg
}
