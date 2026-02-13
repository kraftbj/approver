package app

func (h *home) reconcileWorktrees() {
	for i := range h.pr.prList.PRs {
		num := h.pr.prList.PRs[i].Number
		_, exists := h.worktrees[num]
		h.pr.prList.PRs[i].HasWorktree = exists
		h.pr.prList.PRs[i].IsCreatingWorktree = h.creatingWorktrees[num]
	}
}

func (h *home) reconcileClaudeState() {
	for i := range h.pr.prList.PRs {
		num := h.pr.prList.PRs[i].Number
		_, hasReview := h.reviews[num]
		_, isReviewing := h.reviewing[num]
		session, hasTmux := h.tmuxSessions[num]
		h.pr.prList.PRs[i].HasReview = hasReview
		h.pr.prList.PRs[i].IsReviewing = isReviewing
		h.pr.prList.PRs[i].HasTmux = hasTmux && session.Exists()
	}
}

func (h *home) takeSnapshots() {
	for _, pr := range h.pr.prList.PRs {
		h.prSnapshots[pr.Number] = prSnapshot{
			commentCount:   pr.CommentCount,
			ciStatus:       pr.CIStatus(),
			reviewDecision: pr.ReviewDecision,
		}
	}
}

func (h *home) reconcileNotifications() {
	for i := range h.pr.prList.PRs {
		pr := &h.pr.prList.PRs[i]
		snap, exists := h.prSnapshots[pr.Number]
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
		num := h.issues.issueList.Issues[i].Number
		_, exists := h.issueWorktrees[num]
		h.issues.issueList.Issues[i].HasWorktree = exists
		h.issues.issueList.Issues[i].IsCreatingWorktree = h.creatingIssueWorktrees[num]
	}
}

func (h *home) reconcileIssueTmuxSessions() {
	for i := range h.issues.issueList.Issues {
		num := h.issues.issueList.Issues[i].Number
		session, hasTmux := h.issueTmuxSessions[num]
		h.issues.issueList.Issues[i].HasTmux = hasTmux && session.Exists()
	}
}
