package hidden

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
)

// Store is the persistence seam used by the TUI. Implementations load and
// atomically save complete snapshots.
type Store interface {
	Load() (*State, error)
	Save(*State) error
}

// FileStore persists hidden state as versioned JSON.
type FileStore struct {
	path string
	fs   fileSystem
}

// NewFileStore returns a store for path. Use DefaultPath for the standard location.
func NewFileStore(path string) *FileStore { return &FileStore{path: path, fs: osFS{}} }

var userHomeDir = os.UserHomeDir

// DefaultPath returns ~/.config/pr-dashboard/hidden.json.
func DefaultPath() (string, error) {
	home, err := userHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "pr-dashboard", "hidden.json"), nil
}

// Load reads and strictly validates a snapshot. A missing file returns an
// initialized empty state. Malformed data and unsupported versions are errors.
func (s *FileStore) Load() (*State, error) {
	if s == nil || strings.TrimSpace(s.path) == "" {
		return nil, errors.New("hidden state path is empty")
	}
	data, err := s.fs.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return NewState(), nil
		}
		return nil, fmt.Errorf("read hidden state %s: %w", s.path, err)
	}
	state, err := decodeState(data)
	if err != nil {
		return nil, fmt.Errorf("load hidden state %s: %w", s.path, err)
	}
	return state, nil
}

// Save strictly validates and atomically replaces the snapshot using a
// same-directory temporary file with restrictive permissions.
func (s *FileStore) Save(state *State) error {
	if s == nil || strings.TrimSpace(s.path) == "" {
		return errors.New("hidden state path is empty")
	}
	if state == nil {
		return errors.New("cannot save nil state")
	}
	data, err := encodeState(state)
	if err != nil {
		return fmt.Errorf("validate hidden state: %w", err)
	}

	dir := filepath.Dir(s.path)
	if err := s.fs.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("mkdir hidden state directory: %w", err)
	}
	if err := s.fs.Chmod(dir, 0700); err != nil {
		return fmt.Errorf("chmod-dir hidden state directory: %w", err)
	}
	tmp, err := s.fs.CreateTemp(dir, ".hidden-*.tmp")
	if err != nil {
		return fmt.Errorf("create-temp hidden state: %w", err)
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
		return fmt.Errorf("chmod-temp hidden state: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write hidden state: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync-temp hidden state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close-temp hidden state: %w", err)
	}
	if err := s.fs.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("rename hidden state: %w", err)
	}
	renamed = true

	directory, err := s.fs.Open(dir)
	if err != nil {
		return fmt.Errorf("open-dir hidden state: %w", err)
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return fmt.Errorf("sync-dir hidden state: %w", err)
	}
	if err := directory.Close(); err != nil {
		return fmt.Errorf("close-dir hidden state: %w", err)
	}
	return nil
}

type diskState struct {
	Version  int                `json:"version"`
	Accounts map[string][]Entry `json:"accounts"`
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
	for account, entries := range disk.Accounts {
		key := canonical(account)
		if key == "" {
			return nil, errors.New("account login cannot be empty")
		}
		for _, entry := range entries {
			if err := validateEntry(entry); err != nil {
				return nil, fmt.Errorf("account %q: %w", account, err)
			}
			for _, existing := range state.accounts[key] {
				if sameRule(existing, entry) {
					return nil, fmt.Errorf("account %q contains duplicate rule", account)
				}
			}
			state.accounts[key] = append(state.accounts[key], entry)
		}
	}
	return state, nil
}

func encodeState(state *State) ([]byte, error) {
	if state.Version != CurrentVersion {
		return nil, fmt.Errorf("unsupported version %d (want %d)", state.Version, CurrentVersion)
	}
	disk := diskState{Version: CurrentVersion, Accounts: make(map[string][]Entry, len(state.accounts))}
	for account, entries := range state.accounts {
		key := canonical(account)
		if key == "" {
			return nil, errors.New("account login cannot be empty")
		}
		copyEntries := append([]Entry(nil), entries...)
		for i, entry := range copyEntries {
			if err := validateEntry(entry); err != nil {
				return nil, fmt.Errorf("account %q: %w", account, err)
			}
			for j := 0; j < i; j++ {
				if sameRule(copyEntries[j], entry) {
					return nil, fmt.Errorf("account %q contains duplicate rule", account)
				}
			}
		}
		sort.Slice(copyEntries, func(i, j int) bool {
			left, right := copyEntries[i], copyEntries[j]
			if kindRank(left.Kind) != kindRank(right.Kind) {
				return kindRank(left.Kind) < kindRank(right.Kind)
			}
			if canonical(left.Organization) != canonical(right.Organization) {
				return canonical(left.Organization) < canonical(right.Organization)
			}
			if canonical(left.Repository) != canonical(right.Repository) {
				return canonical(left.Repository) < canonical(right.Repository)
			}
			return left.Number < right.Number
		})
		disk.Accounts[key] = copyEntries
	}
	data, err := json.MarshalIndent(disk, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode JSON: %w", err)
	}
	return append(data, '\n'), nil
}

func validateEntry(entry Entry) error {
	if entry.Kind != KindRepository && entry.Kind != KindPullRequest {
		return fmt.Errorf("invalid kind %q", entry.Kind)
	}
	if canonical(entry.Organization) == "" {
		return errors.New("organization cannot be empty")
	}
	if canonical(entry.Repository) == "" {
		return errors.New("repository cannot be empty")
	}
	if entry.Kind == KindRepository && entry.Number != 0 {
		return errors.New("repository rule number must be zero")
	}
	if entry.Kind == KindPullRequest && entry.Number <= 0 {
		return errors.New("pull request number must be positive")
	}
	return nil
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
