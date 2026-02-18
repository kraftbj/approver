package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RepoSource is a user-configured repository location.
// Either Path (single repo) or ScanDir (directory of repos) should be set.
type RepoSource struct {
	Path     string            `yaml:"path,omitempty"`
	ScanDir  string            `yaml:"scan_dir,omitempty"`
	Host     string            `yaml:"host,omitempty"`
	Env      map[string]string `yaml:"env,omitempty"`
	Features []string          `yaml:"features,omitempty"`
}

// DefaultFeatures is the set of features enabled when none are configured.
var DefaultFeatures = []string{"prs", "issues"}

// RepoEntry is a resolved repository with its name and absolute directory.
type RepoEntry struct {
	Name     string
	Dir      string
	Host     string
	Env      map[string]string
	Features []string
}

// HasFeature returns true if the given feature (e.g. "prs", "issues") is enabled.
func (r RepoEntry) HasFeature(feature string) bool {
	for _, f := range r.Features {
		if f == feature {
			return true
		}
	}
	return false
}

// ResolveRepoDirs resolves a list of RepoSource entries into deduplicated RepoEntry values.
// Each path is expanded (~ → home), validated as a git repo, and deduplicated by absolute path.
func ResolveRepoDirs(sources []RepoSource) ([]RepoEntry, error) {
	seen := make(map[string]bool)
	var entries []RepoEntry

	for _, src := range sources {
		features := src.Features
		if len(features) == 0 {
			features = DefaultFeatures
		}

		if src.Path != "" {
			dir, err := ExpandPath(src.Path)
			if err != nil {
				return nil, fmt.Errorf("expanding path %q: %w", src.Path, err)
			}
			if seen[dir] {
				continue
			}
			if !IsGitRepo(dir) {
				return nil, fmt.Errorf("%q is not a git repository", dir)
			}
			seen[dir] = true
			entries = append(entries, RepoEntry{
				Name:     filepath.Base(dir),
				Dir:      dir,
				Host:     src.Host,
				Env:      src.Env,
				Features: features,
			})
		} else if src.ScanDir != "" {
			scanDir, err := ExpandPath(src.ScanDir)
			if err != nil {
				return nil, fmt.Errorf("expanding scan_dir %q: %w", src.ScanDir, err)
			}
			children, err := os.ReadDir(scanDir)
			if err != nil {
				return nil, fmt.Errorf("reading scan_dir %q: %w", scanDir, err)
			}
			for _, child := range children {
				if !child.IsDir() {
					continue
				}
				dir := filepath.Join(scanDir, child.Name())
				if seen[dir] {
					continue
				}
				if !IsGitRepo(dir) {
					continue
				}
				seen[dir] = true
				entries = append(entries, RepoEntry{
					Name:     child.Name(),
					Dir:      dir,
					Host:     src.Host,
					Env:      src.Env,
					Features: features,
				})
			}
		}
	}

	return entries, nil
}

// ExpandPath resolves ~ and returns an absolute path.
func ExpandPath(p string) (string, error) {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, p[2:])
	}
	return filepath.Abs(p)
}

// IsGitRepo returns true if the given directory is a git repository.
func IsGitRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	return cmd.Run() == nil
}
