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
	Path    string `yaml:"path,omitempty"`
	ScanDir string `yaml:"scan_dir,omitempty"`
}

// RepoEntry is a resolved repository with its name and absolute directory.
type RepoEntry struct {
	Name string
	Dir  string
}

// ResolveRepoDirs resolves a list of RepoSource entries into deduplicated RepoEntry values.
// Each path is expanded (~ → home), validated as a git repo, and deduplicated by absolute path.
func ResolveRepoDirs(sources []RepoSource) ([]RepoEntry, error) {
	seen := make(map[string]bool)
	var entries []RepoEntry

	for _, src := range sources {
		if src.Path != "" {
			dir, err := expandPath(src.Path)
			if err != nil {
				return nil, fmt.Errorf("expanding path %q: %w", src.Path, err)
			}
			if seen[dir] {
				continue
			}
			if !isGitRepo(dir) {
				return nil, fmt.Errorf("%q is not a git repository", dir)
			}
			seen[dir] = true
			entries = append(entries, RepoEntry{
				Name: filepath.Base(dir),
				Dir:  dir,
			})
		} else if src.ScanDir != "" {
			scanDir, err := expandPath(src.ScanDir)
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
				if !isGitRepo(dir) {
					continue
				}
				seen[dir] = true
				entries = append(entries, RepoEntry{
					Name: child.Name(),
					Dir:  dir,
				})
			}
		}
	}

	return entries, nil
}

func expandPath(p string) (string, error) {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, p[2:])
	}
	return filepath.Abs(p)
}

func isGitRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	return cmd.Run() == nil
}
