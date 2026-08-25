// Package hidden stores persistent repository and pull-request visibility rules.
package hidden

import (
	"sort"
	"strings"
	"time"
)

// CurrentVersion is the JSON persistence format understood by this package.
const CurrentVersion = 1

// Kind identifies the kind of hidden rule.
type Kind string

const (
	// KindRepository hides every pull request in a repository.
	KindRepository Kind = "repository"
	// KindPullRequest hides one pull request independently of repository rules.
	KindPullRequest Kind = "pull_request"
)

// Entry is one persistent hidden-item rule. Organization, Repository, and
// Title retain their display casing while matching remains case-insensitive.
type Entry struct {
	Kind         Kind      `json:"kind"`
	Organization string    `json:"organization"`
	Repository   string    `json:"repository"`
	Number       int       `json:"number,omitempty"`
	Title        string    `json:"title,omitempty"`
	HiddenAt     time.Time `json:"hidden_at"`
}

// State contains hidden rules scoped by canonical GitHub account login.
type State struct {
	Version  int `json:"version"`
	accounts map[string][]Entry
}

// NewState returns an initialized, empty state using the current format.
func NewState() *State {
	return &State{Version: CurrentVersion, accounts: make(map[string][]Entry)}
}

// Entries returns an independent copy of an account's rules, newest first.
// Equal timestamps are ordered deterministically by kind and identity.
func (s *State) Entries(account string) []Entry {
	entries := make([]Entry, 0)
	if s != nil {
		entries = append(entries, s.accounts[canonical(account)]...)
	}
	sort.Slice(entries, func(i, j int) bool {
		if !entries[i].HiddenAt.Equal(entries[j].HiddenAt) {
			return entries[i].HiddenAt.After(entries[j].HiddenAt)
		}
		if kindRank(entries[i].Kind) != kindRank(entries[j].Kind) {
			return kindRank(entries[i].Kind) < kindRank(entries[j].Kind)
		}
		leftOrg, rightOrg := canonical(entries[i].Organization), canonical(entries[j].Organization)
		if leftOrg != rightOrg {
			return leftOrg < rightOrg
		}
		leftRepo, rightRepo := canonical(entries[i].Repository), canonical(entries[j].Repository)
		if leftRepo != rightRepo {
			return leftRepo < rightRepo
		}
		if entries[i].Number != entries[j].Number {
			return entries[i].Number < entries[j].Number
		}
		return entries[i].Title < entries[j].Title
	})
	return entries
}

// IsRepositoryHidden reports whether an exact repository rule exists.
func (s *State) IsRepositoryHidden(account, organization, repository string) bool {
	return s.has(account, KindRepository, organization, repository, 0)
}

// IsPRHidden reports whether a pull request is hidden by either its own rule
// or a repository rule.
func (s *State) IsPRHidden(account, organization, repository string, number int) bool {
	return s.IsRepositoryHidden(account, organization, repository) ||
		s.has(account, KindPullRequest, organization, repository, number)
}

// HideRepository adds an exact repository rule. It returns false when the
// same case-insensitive rule already exists or the receiver is nil.
func (s *State) HideRepository(account, organization, repository, title string, hiddenAt time.Time) bool {
	return s.add(account, Entry{Kind: KindRepository, Organization: organization, Repository: repository, Title: title, HiddenAt: hiddenAt})
}

// HidePR adds an exact pull-request rule. It returns false when the same
// case-insensitive rule already exists or the receiver is nil.
func (s *State) HidePR(account, organization, repository string, number int, title string, hiddenAt time.Time) bool {
	return s.add(account, Entry{Kind: KindPullRequest, Organization: organization, Repository: repository, Number: number, Title: title, HiddenAt: hiddenAt})
}

// Unhide removes only the exact rule represented by entry. Removing a
// repository rule does not remove independent pull-request rules, or vice versa.
func (s *State) Unhide(account string, entry Entry) bool {
	if s == nil || (entry.Kind != KindRepository && entry.Kind != KindPullRequest) {
		return false
	}
	key := canonical(account)
	entries := s.accounts[key]
	for i := range entries {
		if sameRule(entries[i], entry) {
			s.accounts[key] = append(entries[:i], entries[i+1:]...)
			if len(s.accounts[key]) == 0 {
				delete(s.accounts, key)
			}
			return true
		}
	}
	return false
}

// RuleCount returns the number of explicit rules for an account.
func (s *State) RuleCount(account string) int {
	if s == nil {
		return 0
	}
	return len(s.accounts[canonical(account)])
}

// Clone returns a deep copy. Cloning a nil state returns an initialized empty state.
func (s *State) Clone() *State {
	clone := NewState()
	if s == nil {
		return clone
	}
	clone.Version = s.Version
	for account, entries := range s.accounts {
		clone.accounts[account] = append([]Entry(nil), entries...)
	}
	return clone
}

func (s *State) add(account string, entry Entry) bool {
	if s == nil {
		return false
	}
	if s.accounts == nil {
		s.accounts = make(map[string][]Entry)
	}
	if s.Version == 0 {
		s.Version = CurrentVersion
	}
	key := canonical(account)
	for _, existing := range s.accounts[key] {
		if sameRule(existing, entry) {
			return false
		}
	}
	s.accounts[key] = append(s.accounts[key], entry)
	return true
}

func (s *State) has(account string, kind Kind, organization, repository string, number int) bool {
	if s == nil {
		return false
	}
	candidate := Entry{Kind: kind, Organization: organization, Repository: repository, Number: number}
	for _, entry := range s.accounts[canonical(account)] {
		if sameRule(entry, candidate) {
			return true
		}
	}
	return false
}

func sameRule(left, right Entry) bool {
	return left.Kind == right.Kind && canonical(left.Organization) == canonical(right.Organization) &&
		canonical(left.Repository) == canonical(right.Repository) && left.Number == right.Number
}

func canonical(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

func kindRank(kind Kind) int {
	if kind == KindRepository {
		return 0
	}
	return 1
}
