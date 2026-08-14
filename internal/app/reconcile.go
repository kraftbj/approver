package app

import "path/filepath"

func (h *home) reconcileWorktrees() {
	h.reconcileConductorWorktrees()
	for i := range h.pr.prList.PRs {
		k := PRKey(&h.pr.prList.PRs[i])
		_, source, exists := h.worktreePathForKey(k)
		h.pr.prList.PRs[i].HasWorktree = exists
		h.pr.prList.PRs[i].WorktreeSource = source
		h.pr.prList.PRs[i].IsCreatingWorktree = h.creatingWorktrees[k]
	}
}

func (h *home) reconcileConductorWorktrees() {
	if h.conductorWorktrees == nil {
		h.conductorWorktrees = make(map[ItemKey]string)
	}
	for key := range h.conductorWorktrees {
		delete(h.conductorWorktrees, key)
	}
	if len(h.conductorWorkspaces) == 0 {
		return
	}

	for i := range h.pr.prList.PRs {
		pr := &h.pr.prList.PRs[i]
		for _, ws := range h.conductorWorkspaces {
			if !sameCleanPath(ws.RepoRoot, pr.RepoDir) || ws.Branch != pr.HeadRefName {
				continue
			}
			h.conductorWorktrees[PRKey(pr)] = ws.Path
			break
		}
	}
}

func sameCleanPath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func (h *home) reconcileClaudeState() {
	for i := range h.pr.prList.PRs {
		k := PRKey(&h.pr.prList.PRs[i])
		_, hasReview := h.reviews[k]
		_, isReviewing := h.reviewing[k]
		session, hasTmux := h.tmuxSessions[k]
		reviewSession, hasReviewTmux := h.reviewTmuxSessions[k]
		h.pr.prList.PRs[i].HasReview = hasReview
		h.pr.prList.PRs[i].IsReviewing = isReviewing
		h.pr.prList.PRs[i].HasTmux = (hasTmux && session.Exists()) || (hasReviewTmux && reviewSession.Exists())
	}
}

func (h *home) takeSnapshots() {
	for _, pr := range h.pr.prList.PRs {
		k := PRKey(&pr)
		h.prSnapshots[k] = prSnapshot{
			commentCount:   pr.CommentCount,
			ciStatus:       pr.CIStatus(),
			reviewDecision: pr.ReviewDecision,
		}
	}
}

func (h *home) reconcileNotifications() {
	for i := range h.pr.prList.PRs {
		pr := &h.pr.prList.PRs[i]
		k := PRKey(pr)
		snap, exists := h.prSnapshots[k]
		if !exists {
			continue
		}
		if pr.CommentCount != snap.commentCount ||
			pr.CIStatus() != snap.ciStatus ||
			pr.ReviewDecision != snap.reviewDecision {
			pr.HasNotification = true
		}
	}
}

func (h *home) reconcileIssueWorktrees() {
	for i := range h.issues.issueList.Issues {
		k := IssueKey(&h.issues.issueList.Issues[i])
		_, exists := h.issueWorktrees[k]
		h.issues.issueList.Issues[i].HasWorktree = exists
		h.issues.issueList.Issues[i].IsCreatingWorktree = h.creatingIssueWorktrees[k]
	}
}

func (h *home) reconcileIssueTmuxSessions() {
	for i := range h.issues.issueList.Issues {
		k := IssueKey(&h.issues.issueList.Issues[i])
		session, hasTmux := h.issueTmuxSessions[k]
		h.issues.issueList.Issues[i].HasTmux = hasTmux && session.Exists()
	}
}
