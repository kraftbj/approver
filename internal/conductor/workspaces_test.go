package conductor

import "testing"

func TestParseWorkspaceRows(t *testing.T) {
	row := "jetpack\x1f/Users/kraft/code/jetpack\x1ffeature/test\x1f/Users/kraft/conductor/workspaces/jetpack/kigali\x1fkigali\x1fTest workspace\x1fready\x1fworking\x1f2026-08-14T21:09:45.776Z"

	workspaces, err := ParseWorkspaceRows(row)
	if err != nil {
		t.Fatal(err)
	}
	if len(workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(workspaces))
	}

	ws := workspaces[0]
	if ws.RepoName != "jetpack" {
		t.Errorf("RepoName: got %q", ws.RepoName)
	}
	if ws.RepoRoot != "/Users/kraft/code/jetpack" {
		t.Errorf("RepoRoot: got %q", ws.RepoRoot)
	}
	if ws.Branch != "feature/test" {
		t.Errorf("Branch: got %q", ws.Branch)
	}
	if ws.Path != "/Users/kraft/conductor/workspaces/jetpack/kigali" {
		t.Errorf("Path: got %q", ws.Path)
	}
	if ws.SessionStatus != "working" {
		t.Errorf("SessionStatus: got %q", ws.SessionStatus)
	}
}

func TestParseWorkspaceRowsEmpty(t *testing.T) {
	workspaces, err := ParseWorkspaceRows("")
	if err != nil {
		t.Fatal(err)
	}
	if len(workspaces) != 0 {
		t.Fatalf("expected no workspaces, got %d", len(workspaces))
	}
}

func TestParseWorkspaceRowsRejectsUnexpectedFields(t *testing.T) {
	_, err := ParseWorkspaceRows("too\x1ffew")
	if err == nil {
		t.Fatal("expected parse error")
	}
}
