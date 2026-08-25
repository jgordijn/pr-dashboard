package viewstate

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDefaultPath(t *testing.T) {
	old := userHomeDir
	userHomeDir = func() (string, error) { return "/home/tester", nil }
	t.Cleanup(func() { userHomeDir = old })
	got, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/home/tester", ".config", "pr-dashboard", "view-state.json")
	if got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}
	userHomeDir = func() (string, error) { return "", errors.New("no home") }
	if _, err := DefaultPath(); err == nil || !strings.Contains(err.Error(), "home directory") {
		t.Fatalf("DefaultPath() error = %v", err)
	}
}

func TestFileStoreRoundTripDeterminismAndPermissions(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nested", "view-state.json")
	store := NewFileStore(path)
	state := NewState()
	first := validAccountState()
	first.OrganizationCollapsed = map[string]bool{"org:z": true, "org:a": true}
	if err := state.SetAccount("Zed", first); err != nil {
		t.Fatal(err)
	}
	second := validAccountState()
	second.SelectedKey = "other/repo#1"
	if err := state.SetAccount("alice", second); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, state) {
		t.Fatalf("round trip = %#v, want %#v", loaded, state)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 || mustStat(t, filepath.Dir(path)).Mode().Perm() != 0700 {
		t.Fatalf("permissions file=%o dir=%o", info.Mode().Perm(), mustStat(t, filepath.Dir(path)).Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.HasSuffix(text, "\n") || strings.Index(text, `"alice"`) > strings.Index(text, `"zed"`) || strings.Index(text, `"org:a"`) > strings.Index(text, `"org:z"`) {
		t.Fatalf("non-deterministic JSON:\n%s", text)
	}
	if strings.Contains(text, "org:open") || !strings.Contains(text, `"version": 1`) {
		t.Fatalf("unexpected JSON:\n%s", text)
	}

	first.SelectedKey = "replacement"
	if err := state.SetAccount("zed", first); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.Load()
	got, _ := loaded.Account("zed")
	if err != nil || got.SelectedKey != "replacement" {
		t.Fatalf("replacement = %#v, %v", got, err)
	}
}

func TestFileStoreMissingAndArgumentErrors(t *testing.T) {
	t.Parallel()
	store := NewFileStore(filepath.Join(t.TempDir(), "missing.json"))
	state, err := store.Load()
	if err != nil || state == nil || state.Version != CurrentVersion {
		t.Fatalf("missing Load() = %#v, %v", state, err)
	}
	empty := NewFileStore("")
	if _, err := empty.Load(); err == nil || !strings.Contains(err.Error(), "path") {
		t.Fatalf("empty Load() = %v", err)
	}
	if err := empty.Save(NewState()); err == nil || !strings.Contains(err.Error(), "path") {
		t.Fatalf("empty Save() = %v", err)
	}
	if err := store.Save(nil); err == nil || !strings.Contains(err.Error(), "nil") {
		t.Fatalf("nil Save() = %v", err)
	}
	var nilStore *FileStore
	if _, err := nilStore.Load(); err == nil || !strings.Contains(err.Error(), "path") {
		t.Fatalf("nil Load() = %v", err)
	}
}

func TestFileStoreStrictLoadErrors(t *testing.T) {
	t.Parallel()
	base := `{"version":1,"accounts":{"me":{"grouping":"organization","display_mode":"full","show_drafts":true,"sort_field":"name","sort_direction":"ascending","selected_key":"","organization_collapsed":[],"tree_organization_collapsed":[],"repository_collapsed":[]}}}`
	tests := []struct{ name, content, want string }{
		{"malformed", `{`, "decode"},
		{"trailing value", base + `{}`, "trailing"},
		{"trailing token", base + `x`, "trailing"},
		{"unknown root", `{"version":1,"accounts":{},"extra":1}`, "unknown field"},
		{"unsupported version", `{"version":2,"accounts":{}}`, "unsupported version"},
		{"missing version", `{"accounts":{}}`, "unsupported version"},
		{"unknown account field", strings.Replace(base, `"selected_key":""`, `"selected_key":"","extra":1`, 1), "unknown field"},
		{"empty account", `{"version":1,"accounts":{" ":{"grouping":"organization","display_mode":"full","show_drafts":true,"sort_field":"name","sort_direction":"ascending"}}}`, "login"},
		{"invalid grouping", strings.Replace(base, `"grouping":"organization"`, `"grouping":"bad"`, 1), "grouping"},
		{"invalid display", strings.Replace(base, `"display_mode":"full"`, `"display_mode":"bad"`, 1), "display"},
		{"invalid sort field", strings.Replace(base, `"sort_field":"name"`, `"sort_field":"bad"`, 1), "sort field"},
		{"invalid direction", strings.Replace(base, `"sort_direction":"ascending"`, `"sort_direction":"bad"`, 1), "sort direction"},
		{"duplicate canonical account", strings.Replace(base, `"me":`, `"ME":{"grouping":"organization","display_mode":"full","show_drafts":true,"sort_field":"name","sort_direction":"ascending"}," me ":`, 1), "duplicate account"},
		{"duplicate collapse", strings.Replace(base, `"organization_collapsed":[]`, `"organization_collapsed":["org:a","org:a"]`, 1), "duplicate"},
		{"duplicate tree collapse", strings.Replace(base, `"tree_organization_collapsed":[]`, `"tree_organization_collapsed":["org:a","org:a"]`, 1), "duplicate"},
		{"duplicate repository collapse", strings.Replace(base, `"repository_collapsed":[]`, `"repository_collapsed":["repo:a/x","repo:a/x"]`, 1), "duplicate"},
		{"empty collapse", strings.Replace(base, `"repository_collapsed":[]`, `"repository_collapsed":[" "]`, 1), "repository"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "view-state.json")
			if err := os.WriteFile(path, []byte(tc.content), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := NewFileStore(path).Load()
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("Load() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestFileStoreReadAndEncodeValidationErrors(t *testing.T) {
	t.Parallel()
	if _, err := NewFileStore(t.TempDir()).Load(); err == nil || !strings.Contains(err.Error(), "read") {
		t.Fatalf("read error = %v", err)
	}
	state := NewState()
	state.Version = 2
	fake := newFakeFS("")
	err := (&FileStore{path: "/state/view-state.json", fs: fake}).Save(state)
	if err == nil || !strings.Contains(err.Error(), "version") || len(fake.calls) != 0 {
		t.Fatalf("validation error = %v, calls=%v", err, fake.calls)
	}
	state = NewState()
	state.accounts["me"] = AccountState{GroupingMode: 99}
	if _, err := encodeState(state); err == nil || !strings.Contains(err.Error(), "grouping") {
		t.Fatalf("encode invalid account = %v", err)
	}
	state = NewState()
	state.accounts[" "] = validAccountState()
	if _, err := encodeState(state); err == nil || !strings.Contains(err.Error(), "login") {
		t.Fatalf("encode invalid login = %v", err)
	}
	state = NewState()
	state.accounts["ME"] = validAccountState()
	state.accounts[" me "] = validAccountState()
	if _, err := encodeState(state); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("encode duplicate login = %v", err)
	}
	if got, err := strictDisplayMode("minimal"); err != nil || got.String() != "minimal" {
		t.Fatalf("strictDisplayMode(minimal) = %v, %v", got, err)
	}
}

func TestFileStoreAtomicFailurePaths(t *testing.T) {
	state := NewState()
	if err := state.SetAccount("me", validAccountState()); err != nil {
		t.Fatal(err)
	}
	operations := []string{"mkdir", "chmod-dir", "create-temp", "chmod-temp", "write", "sync-temp", "close-temp", "rename", "open-dir", "sync-dir", "close-dir"}
	for _, operation := range operations {
		t.Run(operation, func(t *testing.T) {
			fake := newFakeFS(operation)
			err := (&FileStore{path: "/state/view-state.json", fs: fake}).Save(state)
			if err == nil || !strings.Contains(err.Error(), operation) {
				t.Fatalf("Save() error = %v", err)
			}
			if fake.temp != nil && !fake.renamed && !fake.removed {
				t.Fatal("temporary file not removed")
			}
		})
	}
}

func TestFileStoreRealRenameFailureCleansTemp(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "view-state.json")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "occupied"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	err := NewFileStore(path).Save(NewState())
	if err == nil || !strings.Contains(err.Error(), "rename") {
		t.Fatalf("Save() = %v", err)
	}
	matches, globErr := filepath.Glob(filepath.Join(dir, ".view-state-*.tmp"))
	if globErr != nil || len(matches) != 0 {
		t.Fatalf("temporary files = %v, %v", matches, globErr)
	}
}

func mustStat(t *testing.T, path string) fs.FileInfo {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

type fakeFile struct {
	owner *fakeFS
	kind  string
	data  []byte
}

func (f *fakeFile) Write(p []byte) (int, error) {
	f.owner.calls = append(f.owner.calls, "write")
	if f.owner.fail == "write" {
		return 0, errors.New("write")
	}
	f.data = append(f.data, p...)
	return len(p), nil
}
func (f *fakeFile) Chmod(fs.FileMode) error {
	op := "chmod-" + f.kind
	f.owner.calls = append(f.owner.calls, op)
	if f.owner.fail == op {
		return errors.New(op)
	}
	return nil
}
func (f *fakeFile) Sync() error {
	op := "sync-" + f.kind
	f.owner.calls = append(f.owner.calls, op)
	if f.owner.fail == op {
		return errors.New(op)
	}
	return nil
}
func (f *fakeFile) Close() error {
	op := "close-" + f.kind
	f.owner.calls = append(f.owner.calls, op)
	if f.owner.fail == op {
		return errors.New(op)
	}
	return nil
}
func (f *fakeFile) Name() string { return "/state/temp" }

type fakeFS struct {
	fail    string
	calls   []string
	temp    *fakeFile
	removed bool
	renamed bool
}

func newFakeFS(fail string) *fakeFS               { return &fakeFS{fail: fail} }
func (f *fakeFS) ReadFile(string) ([]byte, error) { return nil, os.ErrNotExist }
func (f *fakeFS) MkdirAll(string, fs.FileMode) error {
	f.calls = append(f.calls, "mkdir")
	if f.fail == "mkdir" {
		return errors.New("mkdir")
	}
	return nil
}
func (f *fakeFS) Chmod(string, fs.FileMode) error {
	f.calls = append(f.calls, "chmod-dir")
	if f.fail == "chmod-dir" {
		return errors.New("chmod-dir")
	}
	return nil
}
func (f *fakeFS) CreateTemp(string, string) (syncFile, error) {
	f.calls = append(f.calls, "create-temp")
	if f.fail == "create-temp" {
		return nil, errors.New("create-temp")
	}
	f.temp = &fakeFile{owner: f, kind: "temp"}
	return f.temp, nil
}
func (f *fakeFS) Rename(string, string) error {
	f.calls = append(f.calls, "rename")
	if f.fail == "rename" {
		return errors.New("rename")
	}
	f.renamed = true
	return nil
}
func (f *fakeFS) Remove(string) error {
	f.calls = append(f.calls, "remove")
	f.removed = true
	return nil
}
func (f *fakeFS) Open(string) (syncFile, error) {
	f.calls = append(f.calls, "open-dir")
	if f.fail == "open-dir" {
		return nil, errors.New("open-dir")
	}
	return &fakeFile{owner: f, kind: "dir"}, nil
}
