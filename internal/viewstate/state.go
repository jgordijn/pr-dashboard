// Package viewstate stores account-scoped persistent dashboard preferences.
package viewstate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jgordijn/pr-dashboard/internal/model"
)

// CurrentVersion is the persistence format understood by this package.
const CurrentVersion = 1

// AccountState contains the complete persistent dashboard state for one
// canonical GitHub account.
type AccountState struct {
	GroupingMode  model.GroupingMode
	DisplayMode   model.DisplayMode
	ShowDrafts    bool
	SortField     model.SortField
	SortDirection model.SortDirection
	SelectedKey   string

	OrganizationCollapsed     map[string]bool
	TreeOrganizationCollapsed map[string]bool
	RepositoryCollapsed       map[string]bool
}

// State contains independently scoped account snapshots.
type State struct {
	Version  int
	accounts map[string]AccountState
}

// NewState returns an initialized empty state at the current version.
func NewState() *State {
	return &State{Version: CurrentVersion, accounts: make(map[string]AccountState)}
}

// Clone returns a deep copy. Cloning nil returns an initialized empty state.
func (s *State) Clone() *State {
	clone := NewState()
	if s == nil {
		return clone
	}
	clone.Version = s.Version
	for login, account := range s.accounts {
		clone.accounts[login] = cloneAccount(account)
	}
	return clone
}

// Account returns an independent copy of login's state.
func (s *State) Account(login string) (AccountState, bool) {
	if s == nil {
		return AccountState{}, false
	}
	account, ok := s.accounts[canonical(login)]
	if !ok {
		return AccountState{}, false
	}
	return cloneAccount(account), true
}

// SetAccount validates and replaces one account without exposing shared maps.
func (s *State) SetAccount(login string, account AccountState) error {
	if s == nil {
		return errors.New("cannot set account on nil state")
	}
	key := canonical(login)
	if key == "" {
		return errors.New("account login cannot be empty")
	}
	if err := validateAccount(account); err != nil {
		return err
	}
	if s.accounts == nil {
		s.accounts = make(map[string]AccountState)
	}
	if s.Version == 0 {
		s.Version = CurrentVersion
	}
	s.accounts[key] = cloneAccount(account)
	return nil
}

func validateAccount(account AccountState) error {
	if account.GroupingMode != model.GroupingModeOrganization && account.GroupingMode != model.GroupingModeRepository {
		return fmt.Errorf("invalid grouping mode %d", account.GroupingMode)
	}
	if account.DisplayMode != model.DisplayModeFull && account.DisplayMode != model.DisplayModeCompact && account.DisplayMode != model.DisplayModeMinimal {
		return fmt.Errorf("invalid display mode %d", account.DisplayMode)
	}
	if account.SortField != model.SortFieldName && account.SortField != model.SortFieldAge && account.SortField != model.SortFieldState {
		return fmt.Errorf("invalid sort field %q", account.SortField)
	}
	if account.SortDirection != model.SortAscending && account.SortDirection != model.SortDescending {
		return fmt.Errorf("invalid sort direction %q", account.SortDirection)
	}
	if err := validateCollapseMap("organization", account.OrganizationCollapsed); err != nil {
		return err
	}
	if err := validateCollapseMap("tree organization", account.TreeOrganizationCollapsed); err != nil {
		return err
	}
	return validateCollapseMap("repository", account.RepositoryCollapsed)
}

func validateCollapseMap(name string, collapsed map[string]bool) error {
	for key, value := range collapsed {
		if value && strings.TrimSpace(key) == "" {
			return fmt.Errorf("%s collapse key cannot be empty", name)
		}
	}
	return nil
}

func cloneAccount(account AccountState) AccountState {
	account.OrganizationCollapsed = cloneTrueMap(account.OrganizationCollapsed)
	account.TreeOrganizationCollapsed = cloneTrueMap(account.TreeOrganizationCollapsed)
	account.RepositoryCollapsed = cloneTrueMap(account.RepositoryCollapsed)
	return account
}

func cloneTrueMap(source map[string]bool) map[string]bool {
	clone := make(map[string]bool)
	for key, collapsed := range source {
		if collapsed {
			clone[key] = true
		}
	}
	return clone
}

func canonical(value string) string { return strings.ToLower(strings.TrimSpace(value)) }
