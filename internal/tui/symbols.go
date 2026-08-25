package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/jgordijn/pr-dashboard/internal/model"
)

// SymbolSet is the fixed-width visual vocabulary used by PR rows.
type SymbolSet struct {
	CIPassing, CIFailing, CIPending, CINone                   string
	ReviewApproved, ReviewChanges, ReviewRequired, ReviewNone string
	MergeReady, MergeBehind, MergeBlocked, MergeConflicts     string
	MergeUnstable, MergeUnknown, MergeDraft                   string
	Selected, Changed, Thread, Expanded, Collapsed            string
}

var UnicodeSymbols = SymbolSet{
	CIPassing: "✓", CIFailing: "✗", CIPending: "◐", CINone: "·",
	ReviewApproved: "✓", ReviewChanges: "!", ReviewRequired: "?", ReviewNone: "·",
	MergeReady: "✓", MergeBehind: "↓", MergeBlocked: "⊘", MergeConflicts: "≠",
	MergeUnstable: "~", MergeUnknown: "?", MergeDraft: "○",
	Selected: "▶", Changed: "●", Thread: "◈", Expanded: "▾", Collapsed: "▸",
}

var ASCIISymbols = SymbolSet{
	CIPassing: "+", CIFailing: "x", CIPending: "~", CINone: "-",
	ReviewApproved: "+", ReviewChanges: "!", ReviewRequired: "?", ReviewNone: "-",
	MergeReady: "=", MergeBehind: "v", MergeBlocked: "#", MergeConflicts: "X",
	MergeUnstable: "~", MergeUnknown: "?", MergeDraft: "o",
	Selected: ">", Changed: "*", Thread: "t", Expanded: "v", Collapsed: ">",
}

func (s SymbolSet) all() map[string]string {
	return map[string]string{
		"ci passing": s.CIPassing, "ci failing": s.CIFailing, "ci pending": s.CIPending, "ci none": s.CINone,
		"review approved": s.ReviewApproved, "review changes": s.ReviewChanges, "review required": s.ReviewRequired, "review none": s.ReviewNone,
		"merge ready": s.MergeReady, "merge behind": s.MergeBehind, "merge blocked": s.MergeBlocked, "merge conflicts": s.MergeConflicts,
		"merge unstable": s.MergeUnstable, "merge unknown": s.MergeUnknown, "merge draft": s.MergeDraft,
		"selected": s.Selected, "changed": s.Changed, "thread": s.Thread, "expanded": s.Expanded, "collapsed": s.Collapsed,
	}
}

func (m Model) checkSymbol(status model.CheckStatus) (string, lipgloss.Style, string) {
	switch status {
	case model.CheckStatusPassing:
		return m.Symbols.CIPassing, m.Styles.StatusPassingStyle, "passing"
	case model.CheckStatusFailing:
		return m.Symbols.CIFailing, m.Styles.StatusFailingStyle, "failing"
	case model.CheckStatusPending:
		return m.Symbols.CIPending, m.Styles.StatusPendingStyle, "pending"
	default:
		return m.Symbols.CINone, m.Styles.StatusNoneStyle, "none"
	}
}

func (m Model) reviewSymbol(status model.ReviewStatus) (string, lipgloss.Style, string) {
	switch status {
	case model.ReviewStatusApproved:
		return m.Symbols.ReviewApproved, m.Styles.StatusPassingStyle, "approved"
	case model.ReviewStatusChangesRequested:
		return m.Symbols.ReviewChanges, m.Styles.StatusChangesRequestedStyle, "changes"
	case model.ReviewStatusReviewRequired:
		return m.Symbols.ReviewRequired, m.Styles.StatusPendingStyle, "required"
	default:
		return m.Symbols.ReviewNone, m.Styles.StatusNoneStyle, "none"
	}
}

func (m Model) mergeSymbol(pr model.PullRequest) (string, lipgloss.Style, string) {
	status := pr.MergeStatus
	// Draft is a styling channel, but must never conceal conflicts.
	if pr.IsDraft && status != model.MergeStatusConflicts && status != model.MergeStatusDirty {
		status = model.MergeStatusDraft
	}
	switch status {
	case model.MergeStatusReady:
		return m.Symbols.MergeReady, m.Styles.MergeReadyStyle, "ready"
	case model.MergeStatusBehind:
		return m.Symbols.MergeBehind, m.Styles.MergeBehindStyle, "behind"
	case model.MergeStatusBlocked:
		return m.Symbols.MergeBlocked, m.Styles.MergeBlockedStyle, "blocked"
	case model.MergeStatusConflicts, model.MergeStatusDirty:
		return m.Symbols.MergeConflicts, m.Styles.MergeConflictStyle, "conflicts"
	case model.MergeStatusUnstable, model.MergeStatusHasHooks:
		return m.Symbols.MergeUnstable, m.Styles.MergeBehindStyle, "unstable"
	case model.MergeStatusDraft:
		return m.Symbols.MergeDraft, m.Styles.MergeUnknownStyle, "draft"
	default:
		return m.Symbols.MergeUnknown, m.Styles.MergeUnknownStyle, "unknown"
	}
}
