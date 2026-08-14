package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kraft/approver/internal/claude"
)

func TestSaveAndLoadReview(t *testing.T) {
	tmpDir := t.TempDir()

	path := filepath.Join(tmpDir, "review-42.json")

	original := claude.ReviewResult{
		Summary:   "Found 2 issues",
		RawOutput: "Full review output here",
		Agent1Out: "Agent 1 found X",
		Agent2Out: "Agent 2 validated",
		Agent3Out: "1. Issue one\n2. Issue two",
		Issues: []claude.Issue{
			{Number: 1, File: "main.go", Line: 10, Severity: "high", Problem: "Buffer overflow"},
			{Number: 2, File: "util.go", Line: 25, Severity: "low", Problem: "Unused variable"},
		},
	}

	// Save via direct marshal (same logic as saveReviewResult)
	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal review: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("failed to write review file: %v", err)
	}

	// Read back
	readData, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read review file: %v", err)
	}

	var loaded claude.ReviewResult
	if err := json.Unmarshal(readData, &loaded); err != nil {
		t.Fatalf("failed to unmarshal review: %v", err)
	}

	// Compare
	if loaded.Summary != original.Summary {
		t.Errorf("Summary: got %q, want %q", loaded.Summary, original.Summary)
	}
	if loaded.RawOutput != original.RawOutput {
		t.Errorf("RawOutput: got %q, want %q", loaded.RawOutput, original.RawOutput)
	}
	if loaded.Agent1Out != original.Agent1Out {
		t.Errorf("Agent1Out: got %q, want %q", loaded.Agent1Out, original.Agent1Out)
	}
	if loaded.Agent2Out != original.Agent2Out {
		t.Errorf("Agent2Out: got %q, want %q", loaded.Agent2Out, original.Agent2Out)
	}
	if loaded.Agent3Out != original.Agent3Out {
		t.Errorf("Agent3Out: got %q, want %q", loaded.Agent3Out, original.Agent3Out)
	}
	if len(loaded.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(loaded.Issues))
	}
	if loaded.Issues[0].Severity != "high" {
		t.Errorf("Issue[0] severity: got %q, want %q", loaded.Issues[0].Severity, "high")
	}
	if loaded.Issues[1].File != "util.go" {
		t.Errorf("Issue[1] file: got %q, want %q", loaded.Issues[1].File, "util.go")
	}
}

func TestTrackedRepoMatches(t *testing.T) {
	tests := []struct {
		name       string
		storedRepo string
		repo       string
		want       bool
	}{
		{name: "same repo", storedRepo: "jetpack", repo: "jetpack", want: true},
		{name: "legacy stored repo", storedRepo: "", repo: "jetpack", want: true},
		{name: "legacy requested repo", storedRepo: "jetpack", repo: "", want: true},
		{name: "different repos", storedRepo: "jetpack", repo: "woocommerce", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trackedRepoMatches(tt.storedRepo, tt.repo); got != tt.want {
				t.Fatalf("trackedRepoMatches(%q, %q) = %v, want %v", tt.storedRepo, tt.repo, got, tt.want)
			}
		})
	}
}
