package config

import (
	"fmt"
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

	// FixAllowedTools is the set of tools the fix agent can use (default: "Read,Write,Edit,Glob,Grep").
	FixAllowedTools string `yaml:"fix_allowed_tools,omitempty"`

	// PollInterval is the number of seconds between background PR refreshes.
	// Default: 300 (5 minutes). Set to 0 to disable polling.
	PollInterval *int `yaml:"poll_interval,omitempty"`

	// RepoSources lists repositories to monitor. If empty, falls back to CWD.
	RepoSources []RepoSource `yaml:"repo_sources,omitempty"`

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
		PRLimit:         defaultLimit,
		PollInterval:    &defaultPoll,
		AllowedTools:    "Read,Glob,Grep",
		FixAllowedTools: "Read,Write,Edit,Glob,Grep",
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
	if cfg.FixAllowedTools == "" {
		cfg.FixAllowedTools = "Read,Write,Edit,Glob,Grep"
	}

	return cfg
}

// AddRepoSource adds a path-based repo source to the config file.
// It skips duplicates and validates the path is a git repository.
func AddRepoSource(repoPath string) error {
	absPath, err := expandPath(repoPath)
	if err != nil {
		return fmt.Errorf("resolving path %q: %w", repoPath, err)
	}

	if !isGitRepo(absPath) {
		return fmt.Errorf("%q is not a git repository", absPath)
	}

	cfgPath, err := configPath()
	if err != nil {
		return err
	}

	// Load existing config (or start fresh).
	cfg := &Config{}
	data, err := os.ReadFile(cfgPath)
	if err == nil {
		_ = yaml.Unmarshal(data, cfg)
	}

	// Check for duplicates.
	for _, src := range cfg.RepoSources {
		resolved, err := expandPath(src.Path)
		if err == nil && resolved == absPath {
			fmt.Printf("Already tracked: %s\n", absPath)
			return nil
		}
	}

	cfg.RepoSources = append(cfg.RepoSources, RepoSource{Path: absPath})

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	if err := os.WriteFile(cfgPath, out, 0o644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("Added %s to %s\n", absPath, cfgPath)
	return nil
}
