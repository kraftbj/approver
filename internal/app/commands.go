package app

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kraft/approver/internal/claude"
	"github.com/kraft/approver/internal/config"
	gh "github.com/kraft/approver/internal/github"
	"github.com/kraft/approver/internal/worktree"
)

func fetchPRsCmd(repoDir string, limit int) tea.Cmd {
	return func() tea.Msg {
		prs, err := gh.FetchPRs(repoDir, limit)
		if err != nil {
			return prsErrorMsg{err: err}
		}
		return prsLoadedMsg{prs: prs}
	}
}

func fetchSinglePRCmd(repoDir, numberOrURL string) tea.Cmd {
	return func() tea.Msg {
		pr, err := gh.FetchPR(repoDir, numberOrURL)
		if err != nil {
			return prAddErrorMsg{err: err}
		}
		return prAddedMsg{pr: *pr}
	}
}

func clearErrorAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return hideErrMsg{}
	})
}

func openInBrowserCmd(url string) tea.Cmd {
	return func() tea.Msg {
		if !strings.HasPrefix(url, "https://") {
			return nil
		}
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}
		if err := cmd.Run(); err != nil {
			return browserErrorMsg{err: err}
		}
		return nil
	}
}

func createWorktreeCmd(mgr *worktree.Manager, prNumber int, branch string) tea.Cmd {
	return func() tea.Msg {
		path, err := mgr.Create(prNumber, branch)
		if err != nil {
			return worktreeErrorMsg{prNumber: prNumber, err: err}
		}
		return worktreeCreatedMsg{prNumber: prNumber, path: path}
	}
}

func deleteWorktreeCmd(mgr *worktree.Manager, prNumber int) tea.Cmd {
	return func() tea.Msg {
		if err := mgr.Delete(prNumber); err != nil {
			return worktreeErrorMsg{prNumber: prNumber, err: err}
		}
		return worktreeDeletedMsg{prNumber: prNumber}
	}
}

func scanWorktreesCmd(mgr *worktree.Manager) tea.Cmd {
	return func() tea.Msg {
		wts, _ := mgr.ScanExisting()
		return worktreesScanMsg{worktrees: wts}
	}
}

func scanIssueWorktreesCmd(mgr *worktree.Manager) tea.Cmd {
	return func() tea.Msg {
		wts, _ := mgr.ScanIssueWorktrees()
		return issueWorktreesScanMsg{worktrees: wts}
	}
}

func pollTick(intervalSeconds int) tea.Cmd {
	return tea.Tick(time.Duration(intervalSeconds)*time.Second, func(time.Time) tea.Msg {
		return pollTickMsg{}
	})
}

func pollPRsCmd(repoDir string, limit int) tea.Cmd {
	return func() tea.Msg {
		prs, err := gh.FetchPRs(repoDir, limit)
		if err != nil {
			return nil
		}
		return pollPRsLoadedMsg{prs: prs}
	}
}

func fetchCommentsCmd(repoDir string, prNumber int) tea.Cmd {
	return func() tea.Msg {
		comments, err := gh.FetchComments(repoDir, prNumber)
		if err != nil {
			return commentsErrorMsg{prNumber: prNumber, err: err}
		}
		return commentsLoadedMsg{prNumber: prNumber, comments: comments}
	}
}

func removeTrackedPRCmd(prNumber int) tea.Cmd {
	return func() tea.Msg {
		if err := removeTrackedPR(prNumber); err != nil {
			return trackedRemoveErrorMsg{err: err}
		}
		return nil
	}
}

func runClaudeReviewCmd(ctx context.Context, cfg *config.Config, prNumber int, worktreePath, repoDir string) tea.Cmd {
	return func() tea.Msg {
		prompt := ""
		allowedTools := ""
		if cfg != nil {
			prompt = cfg.ReviewPrompt
			allowedTools = cfg.AllowedTools
		}
		result, err := claude.RunReviewPipeline(ctx, worktreePath, repoDir, prNumber, prompt, allowedTools, nil)
		if err != nil {
			return claudeReviewErrorMsg{prNumber: prNumber, err: err}
		}
		return claudeReviewDoneMsg{prNumber: prNumber, review: *result}
	}
}

var funReviewMessages = []string{
	"Reading the diff with fresh eyes...",
	"Checking for off-by-one errors...",
	"Looking for forgotten TODOs...",
	"Sniffing out race conditions...",
	"Pondering variable names...",
	"Hunting for edge cases...",
	"Scrutinizing error handling...",
	"Questioning every nil check...",
	"Considering the blast radius...",
	"Almost there, double-checking...",
}

func reviewSpinnerTick() tea.Cmd {
	return tea.Tick(4*time.Second, func(time.Time) tea.Msg {
		return reviewSpinnerTickMsg{}
	})
}

func updateBranchCmd(wtPath, baseBranch string, prNumber int) tea.Cmd {
	return func() tea.Msg {
		fetchCmd := exec.Command("git", "-C", wtPath, "fetch", "origin", "--", baseBranch)
		if out, err := fetchCmd.CombinedOutput(); err != nil {
			return branchUpdateErrorMsg{prNumber: prNumber, err: fmt.Errorf("fetch: %s", string(out))}
		}
		mergeCmd := exec.Command("git", "-C", wtPath, "merge", "--", fmt.Sprintf("origin/%s", baseBranch))
		if out, err := mergeCmd.CombinedOutput(); err != nil {
			return branchUpdateErrorMsg{prNumber: prNumber, err: fmt.Errorf("merge: %s", string(out))}
		}
		return branchUpdatedMsg{prNumber: prNumber}
	}
}

func pushBranchCmd(wtPath string, prNumber int) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("git", "-C", wtPath, "push", "origin", "HEAD")
		if out, err := cmd.CombinedOutput(); err != nil {
			return branchPushErrorMsg{prNumber: prNumber, err: fmt.Errorf("%s", string(out))}
		}
		return branchPushedMsg{prNumber: prNumber}
	}
}

func requestChangesPRCmd(repoDir string, prNumber int, body string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("gh", "pr", "review", fmt.Sprintf("%d", prNumber), "--request-changes", "--body", body)
		if repoDir != "" {
			cmd.Dir = repoDir
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return prChangesRequestErrorMsg{prNumber: prNumber, err: fmt.Errorf("%s", string(out))}
		}
		return prChangesRequestedMsg{prNumber: prNumber}
	}
}

func approvePRCmd(repoDir string, prNumber int) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("gh", "pr", "review", fmt.Sprintf("%d", prNumber), "--approve")
		if repoDir != "" {
			cmd.Dir = repoDir
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return prApproveErrorMsg{prNumber: prNumber, err: fmt.Errorf("%s", string(out))}
		}
		return prApprovedMsg{prNumber: prNumber}
	}
}

func runSetupCmd(wtPath, setupCommand string, prNumber int) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("sh", "-c", setupCommand)
		cmd.Dir = wtPath
		out, err := cmd.CombinedOutput()
		if err != nil {
			return setupErrorMsg{prNumber: prNumber, err: fmt.Errorf("%s: %s", err, string(out))}
		}
		return setupDoneMsg{prNumber: prNumber, wtPath: wtPath}
	}
}
