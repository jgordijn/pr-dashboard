package viewstate

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jgordijn/pr-dashboard/internal/model"
)

func validAccountState() AccountState {
	return AccountState{
		GroupingMode:              model.GroupingModeRepository,
		DisplayMode:               model.DisplayModeCompact,
		SortField:                 model.SortFieldState,
		SortDirection:             model.SortDescending,
		ShowDrafts:                true,
		SelectedKey:               "acme/api#7",
		OrganizationCollapsed:     map[string]bool{"org:acme": true, "org:open": false},
		TreeOrganizationCollapsed: map[string]bool{"org:tree": true},
		RepositoryCollapsed:       map[string]bool{"repo:acme/api": true},
	}
}

func TestStateAccountScopeCopiesAndCanonicalizes(t *testing.T) {
	state := NewState()
	if state == nil || state.Version != CurrentVersion {
		t.Fatalf("NewState() = %#v", state)
	}
	if _, ok := state.Account("missing"); ok {
		t.Fatal("missing account found")
	}

	account := validAccountState()
	if err := state.SetAccount(" Alice ", account); err != nil {
		t.Fatal(err)
	}
	account.OrganizationCollapsed["org:later"] = true
	got, ok := state.Account("ALICE")
	if !ok {
		t.Fatal("canonical account missing")
	}
	if got.OrganizationCollapsed["org:open"] || got.OrganizationCollapsed["org:later"] || !got.OrganizationCollapsed["org:acme"] {
		t.Fatalf("true-only/deep copy failed: %#v", got.OrganizationCollapsed)
	}
	got.RepositoryCollapsed["repo:mutated/x"] = true
	again, _ := state.Account("alice")
	if again.RepositoryCollapsed["repo:mutated/x"] {
		t.Fatal("Account returned shared map")
	}

	clone := state.Clone()
	cloneAccount, _ := clone.Account("alice")
	cloneAccount.TreeOrganizationCollapsed["org:clone"] = true
	if err := clone.SetAccount("alice", cloneAccount); err != nil {
		t.Fatal(err)
	}
	original, _ := state.Account("alice")
	if original.TreeOrganizationCollapsed["org:clone"] {
		t.Fatal("Clone shared account maps")
	}
	if clone.Version != state.Version {
		t.Fatal("Clone lost version")
	}
	var nilState *State
	if got := nilState.Clone(); got == nil || got.Version != CurrentVersion {
		t.Fatalf("nil Clone() = %#v", got)
	}
	if _, ok := nilState.Account("alice"); ok {
		t.Fatal("nil state returned account")
	}
}

func TestSetAccountValidation(t *testing.T) {
	tests := []struct {
		name   string
		login  string
		alter  func(*AccountState)
		needle string
	}{
		{"empty login", " ", func(*AccountState) {}, "login"},
		{"grouping", "me", func(s *AccountState) { s.GroupingMode = model.GroupingMode(99) }, "grouping"},
		{"display", "me", func(s *AccountState) { s.DisplayMode = model.DisplayMode(99) }, "display"},
		{"sort field", "me", func(s *AccountState) { s.SortField = model.SortField("bad") }, "sort field"},
		{"sort direction", "me", func(s *AccountState) { s.SortDirection = model.SortDirection("bad") }, "sort direction"},
		{"empty organization key", "me", func(s *AccountState) { s.OrganizationCollapsed[" "] = true }, "organization"},
		{"empty tree organization key", "me", func(s *AccountState) { s.TreeOrganizationCollapsed[""] = true }, "tree organization"},
		{"empty repository key", "me", func(s *AccountState) { s.RepositoryCollapsed[" "] = true }, "repository"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := NewState()
			account := validAccountState()
			tc.alter(&account)
			err := state.SetAccount(tc.login, account)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.needle) {
				t.Fatalf("SetAccount() error = %v, want %q", err, tc.needle)
			}
			if _, ok := state.Account("me"); ok {
				t.Fatal("invalid account was committed")
			}
		})
	}

	var nilState *State
	if err := nilState.SetAccount("me", validAccountState()); err == nil || !strings.Contains(err.Error(), "nil") {
		t.Fatalf("nil SetAccount() error = %v", err)
	}
}

func TestSetAccountInitializesZeroStateAndNilMaps(t *testing.T) {
	state := &State{}
	account := validAccountState()
	account.OrganizationCollapsed = nil
	account.TreeOrganizationCollapsed = nil
	account.RepositoryCollapsed = nil
	if err := state.SetAccount("me", account); err != nil {
		t.Fatal(err)
	}
	got, ok := state.Account("me")
	if !ok || got.OrganizationCollapsed == nil || got.TreeOrganizationCollapsed == nil || got.RepositoryCollapsed == nil {
		t.Fatalf("account maps not initialized: %#v", got)
	}
	if state.Version != CurrentVersion {
		t.Fatalf("version = %d", state.Version)
	}
}

func TestAccountStateEqualityFixture(t *testing.T) {
	state := NewState()
	want := validAccountState()
	delete(want.OrganizationCollapsed, "org:open")
	if err := state.SetAccount("me", validAccountState()); err != nil {
		t.Fatal(err)
	}
	got, _ := state.Account("me")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Account() = %#v, want %#v", got, want)
	}
}
