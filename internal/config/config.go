package config

import (
	"os"
	"path/filepath"

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

	// ReviewBudget is the max USD per review (default: 1.00).
	ReviewBudget float64 `yaml:"review_budget,omitempty"`

	// AllowedTools is the set of tools review agents can use (default: "Read,Glob,Grep").
	AllowedTools string `yaml:"allowed_tools,omitempty"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		PRLimit: 50,
	}
}

// configPath returns the path to the config file.
func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "approver", "config.yaml")
}

// LoadConfig reads config from disk, falling back to defaults.
func LoadConfig() *Config {
	cfg := DefaultConfig()

	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return DefaultConfig()
	}

	// Apply defaults for zero values
	if cfg.PRLimit <= 0 {
		cfg.PRLimit = 50
	}
	if cfg.ReviewBudget <= 0 {
		cfg.ReviewBudget = 1.00
	}
	if cfg.AllowedTools == "" {
		cfg.AllowedTools = "Read,Glob,Grep"
	}

	return cfg
}
