package viewstate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jgordijn/pr-dashboard/internal/model"
)

// Store is the persistence seam for complete view-state snapshots.
type Store interface {
	Load() (*State, error)
	Save(*State) error
}

// FileStore persists view state as strict, versioned JSON.
type FileStore struct {
	path string
	fs   fileSystem
}

func NewFileStore(path string) *FileStore { return &FileStore{path: path, fs: osFS{}} }

var userHomeDir = os.UserHomeDir

// DefaultPath returns ~/.config/pr-dashboard/view-state.json.
func DefaultPath() (string, error) {
	home, err := userHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "pr-dashboard", "view-state.json"), nil
}

// Load returns an empty current state for a missing file and rejects malformed,
// unknown, or unsupported data.
func (s *FileStore) Load() (*State, error) {
	if s == nil || strings.TrimSpace(s.path) == "" {
		return nil, errors.New("view state path is empty")
	}
	data, err := s.fs.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return NewState(), nil
		}
		return nil, fmt.Errorf("read view state %s: %w", s.path, err)
	}
	state, err := decodeState(data)
	if err != nil {
		return nil, fmt.Errorf("load view state %s: %w", s.path, err)
	}
	return state, nil
}

// Save validates and atomically replaces a complete snapshot with restrictive
// permissions and durable file and directory synchronization.
func (s *FileStore) Save(state *State) error {
	if s == nil || strings.TrimSpace(s.path) == "" {
		return errors.New("view state path is empty")
	}
	if state == nil {
		return errors.New("cannot save nil view state")
	}
	data, err := encodeState(state)
	if err != nil {
		return fmt.Errorf("validate view state: %w", err)
	}

	dir := filepath.Dir(s.path)
	if err := s.fs.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("mkdir view state directory: %w", err)
	}
	if err := s.fs.Chmod(dir, 0700); err != nil {
		return fmt.Errorf("chmod-dir view state directory: %w", err)
	}
	tmp, err := s.fs.CreateTemp(dir, ".view-state-*.tmp")
	if err != nil {
		return fmt.Errorf("create-temp view state: %w", err)
	}
	tmpPath := tmp.Name()
	renamed := false
	defer func() {
		if !renamed {
			_ = s.fs.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod-temp view state: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write view state: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync-temp view state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close-temp view state: %w", err)
	}
	if err := s.fs.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("rename view state: %w", err)
	}
	renamed = true
	directory, err := s.fs.Open(dir)
	if err != nil {
		return fmt.Errorf("open-dir view state: %w", err)
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return fmt.Errorf("sync-dir view state: %w", err)
	}
	if err := directory.Close(); err != nil {
		return fmt.Errorf("close-dir view state: %w", err)
	}
	return nil
}

type diskState struct {
	Version  int                    `json:"version"`
	Accounts map[string]diskAccount `json:"accounts"`
}

type diskAccount struct {
	Grouping                  string   `json:"grouping"`
	DisplayMode               string   `json:"display_mode"`
	ShowDrafts                bool     `json:"show_drafts"`
	SortField                 string   `json:"sort_field"`
	SortDirection             string   `json:"sort_direction"`
	SelectedKey               string   `json:"selected_key"`
	OrganizationCollapsed     []string `json:"organization_collapsed"`
	TreeOrganizationCollapsed []string `json:"tree_organization_collapsed"`
	RepositoryCollapsed       []string `json:"repository_collapsed"`
}

func decodeState(data []byte) (*State, error) {
	var disk diskState
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&disk); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("trailing JSON data")
		}
		return nil, fmt.Errorf("trailing JSON data: %w", err)
	}
	if disk.Version != CurrentVersion {
		return nil, fmt.Errorf("unsupported version %d (want %d)", disk.Version, CurrentVersion)
	}
	state := NewState()
	seen := make(map[string]bool)
	for login, account := range disk.Accounts {
		key := canonical(login)
		if key == "" {
			return nil, errors.New("account login cannot be empty")
		}
		if seen[key] {
			return nil, fmt.Errorf("duplicate account login %q", login)
		}
		seen[key] = true
		decoded, err := decodeAccount(account)
		if err != nil {
			return nil, fmt.Errorf("account %q: %w", login, err)
		}
		state.accounts[key] = decoded
	}
	return state, nil
}

func decodeAccount(account diskAccount) (AccountState, error) {
	grouping, err := strictGrouping(account.Grouping)
	if err != nil {
		return AccountState{}, err
	}
	display, err := strictDisplayMode(account.DisplayMode)
	if err != nil {
		return AccountState{}, err
	}
	field, err := strictSortField(account.SortField)
	if err != nil {
		return AccountState{}, err
	}
	direction, err := strictSortDirection(account.SortDirection)
	if err != nil {
		return AccountState{}, err
	}
	organizations, err := decodeCollapse("organization", account.OrganizationCollapsed)
	if err != nil {
		return AccountState{}, err
	}
	treeOrganizations, err := decodeCollapse("tree organization", account.TreeOrganizationCollapsed)
	if err != nil {
		return AccountState{}, err
	}
	repositories, err := decodeCollapse("repository", account.RepositoryCollapsed)
	if err != nil {
		return AccountState{}, err
	}
	return AccountState{
		GroupingMode: grouping, DisplayMode: display, ShowDrafts: account.ShowDrafts,
		SortField: field, SortDirection: direction, SelectedKey: account.SelectedKey,
		OrganizationCollapsed: organizations, TreeOrganizationCollapsed: treeOrganizations,
		RepositoryCollapsed: repositories,
	}, nil
}

func encodeState(state *State) ([]byte, error) {
	if state.Version != CurrentVersion {
		return nil, fmt.Errorf("unsupported version %d (want %d)", state.Version, CurrentVersion)
	}
	disk := diskState{Version: CurrentVersion, Accounts: make(map[string]diskAccount, len(state.accounts))}
	for login, account := range state.accounts {
		key := canonical(login)
		if key == "" {
			return nil, errors.New("account login cannot be empty")
		}
		if err := validateAccount(account); err != nil {
			return nil, fmt.Errorf("account %q: %w", login, err)
		}
		if _, exists := disk.Accounts[key]; exists {
			return nil, fmt.Errorf("duplicate account login %q", login)
		}
		disk.Accounts[key] = diskAccount{
			Grouping: account.GroupingMode.String(), DisplayMode: account.DisplayMode.String(),
			ShowDrafts: account.ShowDrafts, SortField: account.SortField.String(),
			SortDirection: account.SortDirection.String(), SelectedKey: account.SelectedKey,
			OrganizationCollapsed:     encodeCollapse(account.OrganizationCollapsed),
			TreeOrganizationCollapsed: encodeCollapse(account.TreeOrganizationCollapsed),
			RepositoryCollapsed:       encodeCollapse(account.RepositoryCollapsed),
		}
	}
	// diskState contains only JSON-native values, so marshaling cannot fail.
	data, _ := json.MarshalIndent(disk, "", "  ")
	return append(data, '\n'), nil
}

func strictGrouping(value string) (model.GroupingMode, error) {
	switch value {
	case "organization":
		return model.GroupingModeOrganization, nil
	case "repository":
		return model.GroupingModeRepository, nil
	default:
		return 0, fmt.Errorf("invalid grouping %q", value)
	}
}

func strictDisplayMode(value string) (model.DisplayMode, error) {
	switch value {
	case "full":
		return model.DisplayModeFull, nil
	case "compact":
		return model.DisplayModeCompact, nil
	case "minimal":
		return model.DisplayModeMinimal, nil
	default:
		return 0, fmt.Errorf("invalid display mode %q", value)
	}
}

func strictSortField(value string) (model.SortField, error) {
	switch model.SortField(value) {
	case model.SortFieldName, model.SortFieldAge, model.SortFieldState:
		return model.SortField(value), nil
	default:
		return "", fmt.Errorf("invalid sort field %q", value)
	}
}

func strictSortDirection(value string) (model.SortDirection, error) {
	switch model.SortDirection(value) {
	case model.SortAscending, model.SortDescending:
		return model.SortDirection(value), nil
	default:
		return "", fmt.Errorf("invalid sort direction %q", value)
	}
}

func decodeCollapse(name string, keys []string) (map[string]bool, error) {
	result := make(map[string]bool, len(keys))
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("%s collapse key cannot be empty", name)
		}
		if result[key] {
			return nil, fmt.Errorf("duplicate %s collapse key %q", name, key)
		}
		result[key] = true
	}
	return result, nil
}

func encodeCollapse(collapsed map[string]bool) []string {
	keys := make([]string, 0, len(collapsed))
	for key, value := range collapsed {
		if value {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

type syncFile interface {
	Write([]byte) (int, error)
	Chmod(fs.FileMode) error
	Sync() error
	Close() error
	Name() string
}

type fileSystem interface {
	ReadFile(string) ([]byte, error)
	MkdirAll(string, fs.FileMode) error
	Chmod(string, fs.FileMode) error
	CreateTemp(string, string) (syncFile, error)
	Rename(string, string) error
	Remove(string) error
	Open(string) (syncFile, error)
}

type osFS struct{}

func (osFS) ReadFile(path string) ([]byte, error)             { return os.ReadFile(path) }
func (osFS) MkdirAll(path string, mode fs.FileMode) error     { return os.MkdirAll(path, mode) }
func (osFS) Chmod(path string, mode fs.FileMode) error        { return os.Chmod(path, mode) }
func (osFS) CreateTemp(dir, pattern string) (syncFile, error) { return os.CreateTemp(dir, pattern) }
func (osFS) Rename(oldPath, newPath string) error             { return os.Rename(oldPath, newPath) }
func (osFS) Remove(path string) error                         { return os.Remove(path) }
func (osFS) Open(path string) (syncFile, error)               { return os.Open(path) }
