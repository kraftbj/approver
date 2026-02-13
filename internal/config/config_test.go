package config

import "testing"

func TestRepoSetupCommand_Match(t *testing.T) {
	cfg := &Config{
		Repos: map[string]RepoConfig{
			"kraft/approver": {SetupCommand: "make setup"},
			"other/repo":    {SetupCommand: "npm install"},
		},
	}

	got := cfg.RepoSetupCommand("git@github.com:kraft/approver.git")
	if got != "make setup" {
		t.Errorf("expected 'make setup', got %q", got)
	}
}

func TestRepoSetupCommand_NoMatch(t *testing.T) {
	cfg := &Config{
		Repos: map[string]RepoConfig{
			"other/repo": {SetupCommand: "npm install"},
		},
	}

	got := cfg.RepoSetupCommand("git@github.com:kraft/approver.git")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestRepoSetupCommand_NilConfig(t *testing.T) {
	var cfg *Config
	got := cfg.RepoSetupCommand("anything")
	if got != "" {
		t.Errorf("expected empty string for nil config, got %q", got)
	}
}

func TestRepoSetupCommand_EmptyRepos(t *testing.T) {
	cfg := &Config{}
	got := cfg.RepoSetupCommand("git@github.com:kraft/approver.git")
	if got != "" {
		t.Errorf("expected empty string for empty repos, got %q", got)
	}
}

func TestRepoSetupCommand_EmptyURL(t *testing.T) {
	cfg := &Config{
		Repos: map[string]RepoConfig{
			"kraft/approver": {SetupCommand: "make setup"},
		},
	}
	got := cfg.RepoSetupCommand("")
	if got != "" {
		t.Errorf("expected empty string for empty URL, got %q", got)
	}
}

func TestRepoSetupCommand_HTTPSMatch(t *testing.T) {
	cfg := &Config{
		Repos: map[string]RepoConfig{
			"kraft/approver": {SetupCommand: "yarn install"},
		},
	}

	got := cfg.RepoSetupCommand("https://github.com/kraft/approver.git")
	if got != "yarn install" {
		t.Errorf("expected 'yarn install', got %q", got)
	}
}

func TestRepoSetupCommand_SSHMatch(t *testing.T) {
	cfg := &Config{
		Repos: map[string]RepoConfig{
			"kraft/approver": {SetupCommand: "make dev"},
		},
	}

	got := cfg.RepoSetupCommand("git@github.com:kraft/approver.git")
	if got != "make dev" {
		t.Errorf("expected 'make dev', got %q", got)
	}
}
