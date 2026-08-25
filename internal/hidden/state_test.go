package hidden

import (
	"reflect"
	"testing"
	"time"
)

func TestStateRulesAndCaseInsensitiveMatching(t *testing.T) {
	t.Parallel()
	state := NewState()
	older := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	newer := older.Add(time.Hour)

	if !state.HideRepository("Alice", "Acme", "API", "Acme API", older) {
		t.Fatal("first repository hide should add a rule")
	}
	if state.HideRepository("alice", "ACME", "api", "replacement", newer) {
		t.Fatal("case-only duplicate should be idempotent")
	}
	if !state.HidePR("ALICE", "Acme", "Web", 42, "Fix Login", newer) {
		t.Fatal("first PR hide should add a rule")
	}
	if state.HidePR("alice", "acme", "WEB", 42, "replacement", newer.Add(time.Hour)) {
		t.Fatal("case-only duplicate PR should be idempotent")
	}

	if !state.IsRepositoryHidden("aLiCe", "aCmE", "Api") {
		t.Fatal("repository matching should be case-insensitive")
	}
	if !state.IsPRHidden("alice", "ACME", "API", 7) {
		t.Fatal("repository rule should effectively hide every PR in it")
	}
	if !state.IsPRHidden("alice", "acme", "web", 42) {
		t.Fatal("explicit PR rule should match case-insensitively")
	}
	if state.IsPRHidden("alice", "acme", "web", 43) {
		t.Fatal("different PR should remain visible")
	}
	if state.IsRepositoryHidden("bob", "acme", "api") {
		t.Fatal("rules must be account-scoped")
	}
	if got := state.RuleCount("ALICE"); got != 2 {
		t.Fatalf("RuleCount() = %d, want 2", got)
	}

	entries := state.Entries("alice")
	want := []Entry{
		{Kind: KindPullRequest, Organization: "Acme", Repository: "Web", Number: 42, Title: "Fix Login", HiddenAt: newer},
		{Kind: KindRepository, Organization: "Acme", Repository: "API", Title: "Acme API", HiddenAt: older},
	}
	if !reflect.DeepEqual(entries, want) {
		t.Fatalf("Entries() = %#v, want %#v", entries, want)
	}
}

func TestEntriesNewestFirstWithDeterministicTiesAndDefensiveCopy(t *testing.T) {
	t.Parallel()
	state := NewState()
	at := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	state.HidePR("me", "zeta", "repo", 2, "second", at)
	state.HideRepository("me", "Beta", "Repo", "beta", at)
	state.HidePR("me", "alpha", "Repo", 10, "ten", at)
	state.HidePR("me", "alpha", "Repo", 2, "two", at)

	entries := state.Entries("ME")
	want := []Entry{
		{Kind: KindRepository, Organization: "Beta", Repository: "Repo", Title: "beta", HiddenAt: at},
		{Kind: KindPullRequest, Organization: "alpha", Repository: "Repo", Number: 2, Title: "two", HiddenAt: at},
		{Kind: KindPullRequest, Organization: "alpha", Repository: "Repo", Number: 10, Title: "ten", HiddenAt: at},
		{Kind: KindPullRequest, Organization: "zeta", Repository: "repo", Number: 2, Title: "second", HiddenAt: at},
	}
	if !reflect.DeepEqual(entries, want) {
		t.Fatalf("Entries() = %#v, want %#v", entries, want)
	}
	entries[0].Title = "mutated"
	if state.Entries("me")[0].Title == "mutated" {
		t.Fatal("Entries exposed internal state")
	}
	if got := NewState().Entries("missing"); got == nil || len(got) != 0 {
		t.Fatalf("empty Entries() = %#v, want non-nil empty slice", got)
	}
}

func TestEntriesCoversAllDeterministicTieBreakers(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	state := NewState()
	state.accounts["me"] = []Entry{
		{Kind: KindPullRequest, Organization: "a", Repository: "z", Number: 1, Title: "z", HiddenAt: at},
		{Kind: KindPullRequest, Organization: "a", Repository: "a", Number: 1, Title: "z", HiddenAt: at},
		{Kind: KindPullRequest, Organization: "a", Repository: "a", Number: 1, Title: "a", HiddenAt: at},
	}
	entries := state.Entries("me")
	if entries[0].Title != "a" || entries[1].Repository != "a" || entries[2].Repository != "z" {
		t.Fatalf("unexpected tie ordering: %#v", entries)
	}
}

func TestUnhideRemovesOnlyExactRule(t *testing.T) {
	t.Parallel()
	state := NewState()
	at := time.Now().UTC()
	state.HideRepository("me", "Acme", "API", "repo", at)
	state.HidePR("me", "Acme", "API", 1, "one", at)
	state.HidePR("me", "Acme", "API", 2, "two", at)

	if !state.Unhide("ME", Entry{Kind: KindRepository, Organization: "acme", Repository: "api"}) {
		t.Fatal("repository rule was not removed")
	}
	if state.IsRepositoryHidden("me", "acme", "api") {
		t.Fatal("repository remains hidden")
	}
	if !state.IsPRHidden("me", "acme", "api", 1) || !state.IsPRHidden("me", "acme", "api", 2) {
		t.Fatal("unhiding repository removed independent PR rules")
	}
	if !state.Unhide("me", Entry{Kind: KindPullRequest, Organization: "ACME", Repository: "API", Number: 1}) {
		t.Fatal("explicit PR rule was not removed")
	}
	if state.IsPRHidden("me", "acme", "api", 1) {
		t.Fatal("PR remains hidden")
	}
	if !state.IsPRHidden("me", "acme", "api", 2) {
		t.Fatal("different PR rule was removed")
	}
	if state.Unhide("missing", Entry{Kind: KindPullRequest, Organization: "acme", Repository: "api", Number: 2}) {
		t.Fatal("missing account unexpectedly removed a rule")
	}
	if state.Unhide("me", Entry{Kind: Kind("invalid"), Organization: "acme", Repository: "api"}) {
		t.Fatal("invalid kind unexpectedly removed a rule")
	}

	only := NewState()
	only.HideRepository("me", "a", "b", "", at)
	if !only.Unhide("me", Entry{Kind: KindRepository, Organization: "a", Repository: "b"}) || only.RuleCount("me") != 0 {
		t.Fatal("last rule was not removed with its account bucket")
	}
}

func TestCloneIsIndependentAndNilSafe(t *testing.T) {
	t.Parallel()
	var nilState *State
	clone := nilState.Clone()
	if clone == nil || clone.Version != CurrentVersion || clone.RuleCount("me") != 0 {
		t.Fatalf("nil Clone() = %#v", clone)
	}

	original := NewState()
	original.HidePR("me", "Acme", "API", 1, "one", time.Now().UTC())
	clone = original.Clone()
	clone.HideRepository("me", "Acme", "Web", "web", time.Now().UTC())
	clone.Unhide("me", Entry{Kind: KindPullRequest, Organization: "acme", Repository: "api", Number: 1})
	if original.RuleCount("me") != 1 || !original.IsPRHidden("me", "acme", "api", 1) {
		t.Fatal("clone mutation affected original")
	}
}

func TestZeroAndNilStateAreUsable(t *testing.T) {
	t.Parallel()
	var state State
	if state.IsRepositoryHidden("me", "a", "b") || state.IsPRHidden("me", "a", "b", 1) || state.RuleCount("me") != 0 {
		t.Fatal("zero state reported hidden rules")
	}
	if !state.HideRepository("me", "A", "B", "", time.Time{}) {
		t.Fatal("zero state could not add repository")
	}

	var nilState *State
	if nilState.IsRepositoryHidden("me", "a", "b") || nilState.IsPRHidden("me", "a", "b", 1) || nilState.RuleCount("me") != 0 {
		t.Fatal("nil state reported hidden rules")
	}
	if nilState.HideRepository("me", "a", "b", "", time.Time{}) || nilState.HidePR("me", "a", "b", 1, "", time.Time{}) || nilState.Unhide("me", Entry{}) {
		t.Fatal("nil mutation unexpectedly succeeded")
	}
}
