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
	"github.com/kraft/approver/internal/debug"
	gh "github.com/kraft/approver/internal/github"
	"github.com/kraft/approver/internal/worktree"
)

func fetchPRsCmd(repoDir, repoName string, limit int) tea.Cmd {
	return func() tea.Msg {
		debug.Log("[fetchPRsCmd] start repo=%q dir=%q limit=%d", repoName, repoDir, limit)
		prs, err := gh.FetchPRs(repoDir, limit)
		if err != nil {
			debug.Log("[fetchPRsCmd] ERROR repo=%q: %v", repoName, err)
			return prsErrorMsg{err: fmt.Errorf("%s: %w", repoName, err)}
		}
		for i := range prs {
			prs[i].Repo = repoName
			prs[i].RepoDir = repoDir
		}
		debug.Log("[fetchPRsCmd] done repo=%q returned %d PRs", repoName, len(prs))
		return prsLoadedMsg{prs: prs}
	}
}

func fetchSinglePRCmd(repoDir string, key ItemKey, numberOrURL string) tea.Cmd {
	return func() tea.Msg {
		pr, err := gh.FetchPR(repoDir, numberOrURL)
		if err != nil {
			return prAddErrorMsg{err: err}
		}
		pr.Repo = key.Repo
		pr.RepoDir = repoDir
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

func createWorktreeCmd(mgr *worktree.Manager, key ItemKey, branch string) tea.Cmd {
	return func() tea.Msg {
		path, err := mgr.Create(key.Number, branch)
		if err != nil {
			return worktreeErrorMsg{key: key, err: err}
		}
		return worktreeCreatedMsg{key: key, path: path}
	}
}

func deleteWorktreeCmd(mgr *worktree.Manager, key ItemKey) tea.Cmd {
	return func() tea.Msg {
		if err := mgr.Delete(key.Number); err != nil {
			return worktreeErrorMsg{key: key, err: err}
		}
		return worktreeDeletedMsg{key: key}
	}
}

func scanWorktreesCmd(mgr *worktree.Manager, repoName string) tea.Cmd {
	return func() tea.Msg {
		wts, _ := mgr.ScanExisting()
		result := make(map[ItemKey]string, len(wts))
		for num, path := range wts {
			result[ItemKey{Repo: repoName, Number: num}] = path
		}
		return worktreesScanMsg{repo: repoName, worktrees: result}
	}
}

func scanIssueWorktreesCmd(mgr *worktree.Manager, repoName string) tea.Cmd {
	return func() tea.Msg {
		wts, _ := mgr.ScanIssueWorktrees()
		result := make(map[ItemKey]string, len(wts))
		for num, path := range wts {
			result[ItemKey{Repo: repoName, Number: num}] = path
		}
		return issueWorktreesScanMsg{repo: repoName, worktrees: result}
	}
}

func pollTick(intervalSeconds int) tea.Cmd {
	return tea.Tick(time.Duration(intervalSeconds)*time.Second, func(time.Time) tea.Msg {
		return pollTickMsg{}
	})
}

func pollPRsCmd(repoDir, repoName string, limit int) tea.Cmd {
	return func() tea.Msg {
		prs, err := gh.FetchPRs(repoDir, limit)
		if err != nil {
			return nil
		}
		for i := range prs {
			prs[i].Repo = repoName
			prs[i].RepoDir = repoDir
		}
		return pollPRsLoadedMsg{prs: prs}
	}
}

func fetchCommentsCmd(repoDir string, key ItemKey) tea.Cmd {
	return func() tea.Msg {
		comments, err := gh.FetchComments(repoDir, key.Number)
		if err != nil {
			return commentsErrorMsg{key: key, err: err}
		}
		return commentsLoadedMsg{key: key, comments: comments}
	}
}

func detectAndFetchItemCmd(repoDir string, key ItemKey, input string) tea.Cmd {
	return func() tea.Msg {
		// Try PR first
		pr, err := gh.FetchPR(repoDir, input)
		if err == nil {
			pr.Source = "manual"
			pr.Repo = key.Repo
			pr.RepoDir = repoDir
			return itemDetectedMsg{pr: pr}
		}
		// Try issue
		issue, err2 := gh.FetchIssue(repoDir, input)
		if err2 == nil {
			issue.Source = "manual"
			issue.Repo = key.Repo
			issue.RepoDir = repoDir
			return itemDetectedMsg{issue: issue}
		}
		return itemDetectErrorMsg{err: fmt.Errorf("not found as PR or issue: %v", err)}
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

func runClaudeReviewCmd(ctx context.Context, cfg *config.Config, key ItemKey, worktreePath, repoDir string) tea.Cmd {
	return func() tea.Msg {
		prompt := ""
		allowedTools := ""
		if cfg != nil {
			prompt = cfg.ReviewPrompt
			allowedTools = cfg.AllowedTools
		}
		result, err := claude.RunReviewPipeline(ctx, worktreePath, repoDir, key.Number, prompt, allowedTools, nil)
		if err != nil {
			return claudeReviewErrorMsg{key: key, err: err}
		}
		return claudeReviewDoneMsg{key: key, review: *result}
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

func updateBranchCmd(wtPath, baseBranch string, key ItemKey) tea.Cmd {
	return func() tea.Msg {
		fetchCmd := exec.Command("git", "-C", wtPath, "fetch", "origin", "--", baseBranch)
		if out, err := fetchCmd.CombinedOutput(); err != nil {
			return branchUpdateErrorMsg{key: key, err: fmt.Errorf("fetch: %s", string(out))}
		}
		mergeCmd := exec.Command("git", "-C", wtPath, "merge", "--", fmt.Sprintf("origin/%s", baseBranch))
		if out, err := mergeCmd.CombinedOutput(); err != nil {
			return branchUpdateErrorMsg{key: key, err: fmt.Errorf("merge: %s", string(out))}
		}
		return branchUpdatedMsg{key: key}
	}
}

func pushBranchCmd(wtPath string, key ItemKey) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("git", "-C", wtPath, "push", "origin", "HEAD")
		if out, err := cmd.CombinedOutput(); err != nil {
			return branchPushErrorMsg{key: key, err: fmt.Errorf("%s", string(out))}
		}
		return branchPushedMsg{key: key}
	}
}

func requestChangesPRCmd(repoDir string, key ItemKey, body string) tea.Cmd {
	return func() tea.Msg {
		cmd := gh.GHCommand(repoDir, "pr", "review", fmt.Sprintf("%d", key.Number), "--request-changes", "--body", body)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return prChangesRequestErrorMsg{key: key, err: fmt.Errorf("%s", string(out))}
		}
		return prChangesRequestedMsg{key: key}
	}
}

func approvePRCmd(repoDir string, key ItemKey) tea.Cmd {
	return func() tea.Msg {
		cmd := gh.GHCommand(repoDir, "pr", "review", fmt.Sprintf("%d", key.Number), "--approve")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return prApproveErrorMsg{key: key, err: fmt.Errorf("%s", string(out))}
		}
		return prApprovedMsg{key: key}
	}
}

func runFixCmd(ctx context.Context, cfg *config.Config, key ItemKey, worktreePath string, items []claude.FixItem) tea.Cmd {
	return func() tea.Msg {
		allowedTools := ""
		if cfg != nil {
			allowedTools = cfg.FixAllowedTools
		}
		output, err := claude.RunFixAgent(ctx, worktreePath, items, allowedTools, nil)
		if err != nil {
			return fixErrorMsg{key: key, err: err}
		}
		return fixDoneMsg{key: key, output: output}
	}
}

func fixCommitPushCmd(wtPath string, key ItemKey) tea.Cmd {
	return func() tea.Msg {
		addCmd := exec.Command("git", "-C", wtPath, "add", "-A")
		if out, err := addCmd.CombinedOutput(); err != nil {
			return fixCommitPushErrorMsg{key: key, err: fmt.Errorf("git add: %s", string(out))}
		}

		commitMsg := fmt.Sprintf("Fix review findings for PR #%d", key.Number)
		commitCmd := exec.Command("git", "-C", wtPath, "commit", "-m", commitMsg)
		if out, err := commitCmd.CombinedOutput(); err != nil {
			return fixCommitPushErrorMsg{key: key, err: fmt.Errorf("git commit: %s", string(out))}
		}

		pushCmd := exec.Command("git", "-C", wtPath, "push", "origin", "HEAD")
		if out, err := pushCmd.CombinedOutput(); err != nil {
			return fixCommitPushErrorMsg{key: key, err: fmt.Errorf("git push: %s", string(out))}
		}

		return fixCommitPushDoneMsg{key: key}
	}
}

var funFixMessages = []string{
	"Applying fixes...",
	"Patching things up...",
	"Rewriting the bugs away...",
	"Making it right...",
	"Surgeon at work...",
	"Almost done fixing...",
}

func saveConfigCmd(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		if err := config.SaveConfig(cfg); err != nil {
			return configSaveErrorMsg{err: err}
		}
		return configSavedMsg{}
	}
}

func fixSpinnerTick() tea.Cmd {
	return tea.Tick(4*time.Second, func(time.Time) tea.Msg {
		return fixSpinnerTickMsg{}
	})
}

func runSetupCmd(wtPath, setupCommand string, key ItemKey) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("sh", "-c", setupCommand)
		cmd.Dir = wtPath
		out, err := cmd.CombinedOutput()
		if err != nil {
			return setupErrorMsg{key: key, err: fmt.Errorf("%s: %s", err, string(out))}
		}
		return setupDoneMsg{key: key, wtPath: wtPath}
	}
}
