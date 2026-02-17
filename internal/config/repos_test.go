package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init in %s: %v", dir, err)
	}
}

func TestResolveRepoDirs_Path(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "myrepo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, repo)

	entries, err := ResolveRepoDirs([]RepoSource{{Path: repo}})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "myrepo" {
		t.Errorf("expected name myrepo, got %s", entries[0].Name)
	}
	if entries[0].Dir != repo {
		t.Errorf("expected dir %s, got %s", repo, entries[0].Dir)
	}
}

func TestResolveRepoDirs_ScanDir(t *testing.T) {
	tmp := t.TempDir()

	// Create two git repos and one non-git dir
	for _, name := range []string{"repo-a", "repo-b"} {
		dir := filepath.Join(tmp, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		initGitRepo(t, dir)
	}
	// Non-git directory should be skipped
	if err := os.MkdirAll(filepath.Join(tmp, "not-a-repo"), 0o755); err != nil {
		t.Fatal(err)
	}

	entries, err := ResolveRepoDirs([]RepoSource{{ScanDir: tmp}})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name] = true
	}
	if !names["repo-a"] || !names["repo-b"] {
		t.Errorf("expected repo-a and repo-b, got %v", names)
	}
}

func TestResolveRepoDirs_Dedup(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "myrepo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, repo)

	entries, err := ResolveRepoDirs([]RepoSource{
		{Path: repo},
		{Path: repo},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after dedup, got %d", len(entries))
	}
}

func TestResolveRepoDirs_NotGitRepo(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "norepo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := ResolveRepoDirs([]RepoSource{{Path: dir}})
	if err == nil {
		t.Fatal("expected error for non-git repo path")
	}
}
