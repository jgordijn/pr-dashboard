package hidden

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDefaultPath(t *testing.T) {
	old := userHomeDir
	userHomeDir = func() (string, error) { return "/home/tester", nil }
	t.Cleanup(func() { userHomeDir = old })

	got, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/home/tester", ".config", "pr-dashboard", "hidden.json")
	if got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}

	userHomeDir = func() (string, error) { return "", errors.New("no home") }
	if _, err := DefaultPath(); err == nil || !strings.Contains(err.Error(), "home directory") {
		t.Fatalf("DefaultPath() error = %v", err)
	}
}

func TestFileStoreRoundTripAndRestrictivePermissions(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nested", "hidden.json")
	store := NewFileStore(path)
	state := NewState()
	at := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	state.HideRepository("Alice", "Acme", "API", "API repository", at)
	state.HidePR("Alice", "Acme", "Web", 42, "Fix login", at.Add(time.Minute))

	if err := store.Save(state); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, state) {
		t.Fatalf("Load() = %#v, want %#v", loaded, state)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("file mode = %o, want 600", got)
	}
	if got := mustStat(t, filepath.Dir(path)).Mode().Perm(); got != 0700 {
		t.Fatalf("directory mode = %o, want 700", got)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(data), "\n") || !strings.Contains(string(data), `"version": 1`) {
		t.Fatalf("unexpected JSON: %s", data)
	}

	state.HidePR("alice", "Acme", "Web", 43, "Another", at.Add(2*time.Minute))
	if err := store.Save(state); err != nil {
		t.Fatalf("replacement Save() error = %v", err)
	}
	loaded, err = store.Load()
	if err != nil || loaded.RuleCount("alice") != 3 {
		t.Fatalf("replacement Load() = %#v, %v", loaded, err)
	}
}

func TestFileStoreMissingAndEmptyPath(t *testing.T) {
	t.Parallel()
	store := NewFileStore(filepath.Join(t.TempDir(), "missing.json"))
	state, err := store.Load()
	if err != nil || state == nil || state.Version != CurrentVersion || state.RuleCount("any") != 0 {
		t.Fatalf("missing Load() = %#v, %v", state, err)
	}

	empty := NewFileStore("")
	if _, err := empty.Load(); err == nil || !strings.Contains(err.Error(), "path") {
		t.Fatalf("empty Load() error = %v", err)
	}
	if err := empty.Save(NewState()); err == nil || !strings.Contains(err.Error(), "path") {
		t.Fatalf("empty Save() error = %v", err)
	}
	if err := store.Save(nil); err == nil || !strings.Contains(err.Error(), "nil state") {
		t.Fatalf("nil Save() error = %v", err)
	}
}

func TestFileStoreStrictLoadErrors(t *testing.T) {
	t.Parallel()
	validTime := "2026-01-02T03:04:05Z"
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"malformed", `{`, "decode"},
		{"trailing", `{"version":1,"accounts":{}} {}`, "trailing"},
		{"unknown top field", `{"version":1,"accounts":{},"extra":true}`, "unknown field"},
		{"unsupported version", `{"version":2,"accounts":{}}`, "unsupported version"},
		{"missing version", `{"accounts":{}}`, "unsupported version"},
		{"unknown entry field", `{"version":1,"accounts":{"me":[{"kind":"repository","organization":"a","repository":"b","hidden_at":"` + validTime + `","extra":1}]}}`, "unknown field"},
		{"invalid kind", `{"version":1,"accounts":{"me":[{"kind":"other","organization":"a","repository":"b","hidden_at":"` + validTime + `"}]}}`, "kind"},
		{"empty account", `{"version":1,"accounts":{" ":[]}}`, "account"},
		{"empty organization", `{"version":1,"accounts":{"me":[{"kind":"repository","organization":"","repository":"b","hidden_at":"` + validTime + `"}]}}`, "organization"},
		{"empty repository", `{"version":1,"accounts":{"me":[{"kind":"repository","organization":"a","repository":"","hidden_at":"` + validTime + `"}]}}`, "repository"},
		{"repository with number", `{"version":1,"accounts":{"me":[{"kind":"repository","organization":"a","repository":"b","number":1,"hidden_at":"` + validTime + `"}]}}`, "number"},
		{"PR without number", `{"version":1,"accounts":{"me":[{"kind":"pull_request","organization":"a","repository":"b","hidden_at":"` + validTime + `"}]}}`, "number"},
		{"duplicate rule", `{"version":1,"accounts":{"me":[{"kind":"repository","organization":"A","repository":"B","hidden_at":"` + validTime + `"},{"kind":"repository","organization":"a","repository":"b","hidden_at":"` + validTime + `"}]}}`, "duplicate"},
		{"bad timestamp", `{"version":1,"accounts":{"me":[{"kind":"repository","organization":"a","repository":"b","hidden_at":"yesterday"}]}}`, "decode"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "hidden.json")
			if err := os.WriteFile(path, []byte(tc.content), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := NewFileStore(path).Load()
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("Load() error = %v, want containing %q", err, tc.want)
			}
		})
	}
}

func TestFileStoreAdditionalDecodeAndEncodeErrors(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "hidden.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"accounts":{}} x`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFileStore(path).Load(); err == nil || !strings.Contains(err.Error(), "trailing") {
		t.Fatalf("invalid trailing token error = %v", err)
	}

	tests := []struct {
		name  string
		alter func(*State)
		want  string
	}{
		{"unsupported version", func(s *State) { s.Version = 2 }, "unsupported version"},
		{"empty account", func(s *State) { s.accounts[" "] = nil }, "account"},
		{"duplicate", func(s *State) {
			e := Entry{Kind: KindRepository, Organization: "a", Repository: "b"}
			s.accounts["me"] = []Entry{e, e}
		}, "duplicate"},
		{"JSON encoding", func(s *State) {
			s.accounts["me"] = []Entry{{Kind: KindRepository, Organization: "a", Repository: "b", HiddenAt: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)}}
		}, "encode JSON"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := NewState()
			tc.alter(state)
			if _, err := encodeState(state); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("encodeState() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestEncodeStateDeterministicallySortsIdentities(t *testing.T) {
	t.Parallel()
	state := NewState()
	at := time.Now().UTC()
	state.HidePR("me", "z", "z", 2, "", at)
	state.HidePR("me", "a", "z", 2, "", at)
	state.HidePR("me", "a", "a", 2, "", at)
	state.HidePR("me", "a", "a", 1, "", at)
	data, err := encodeState(state)
	if err != nil {
		t.Fatal(err)
	}
	positions := []int{
		strings.Index(string(data), `"number": 1`),
		strings.Index(string(data), `"number": 2`),
		strings.Index(string(data), `"repository": "z"`),
		strings.LastIndex(string(data), `"organization": "z"`),
	}
	for i := 1; i < len(positions); i++ {
		if positions[i-1] < 0 || positions[i] <= positions[i-1] {
			t.Fatalf("unexpected deterministic order in %s", data)
		}
	}
}

func TestFileStoreLoadCanonicalizesAccountKeyButPreservesEntryCase(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "hidden.json")
	content := `{"version":1,"accounts":{"Alice":[{"kind":"pull_request","organization":"Acme","repository":"Web","number":7,"title":"Title","hidden_at":"2026-01-02T03:04:05Z"}]}}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	state, err := NewFileStore(path).Load()
	if err != nil {
		t.Fatal(err)
	}
	entries := state.Entries("ALICE")
	if len(entries) != 1 || entries[0].Organization != "Acme" || entries[0].Repository != "Web" || entries[0].Title != "Title" {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestFileStoreReadError(t *testing.T) {
	t.Parallel()
	path := t.TempDir()
	_, err := NewFileStore(path).Load()
	if err == nil || !strings.Contains(err.Error(), "read") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestFileStoreAtomicFailurePaths(t *testing.T) {
	baseState := NewState()
	baseState.HideRepository("me", "a", "b", "", time.Now().UTC())
	tests := []struct {
		name string
		op   string
	}{
		{"mkdir", "mkdir"}, {"chmod directory", "chmod-dir"}, {"create temp", "create-temp"},
		{"chmod temp", "chmod-temp"}, {"write", "write"}, {"sync temp", "sync-temp"},
		{"close temp", "close-temp"}, {"rename", "rename"}, {"open directory", "open-dir"},
		{"sync directory", "sync-dir"}, {"close directory", "close-dir"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeFS(tc.op)
			store := &FileStore{path: "/state/hidden.json", fs: fake}
			err := store.Save(baseState)
			if err == nil || !strings.Contains(err.Error(), tc.op) {
				t.Fatalf("Save() error = %v, want containing %q", err, tc.op)
			}
			if fake.temp != nil && !fake.renamed && !fake.removed {
				t.Fatal("temporary file was not removed after failure")
			}
		})
	}
}

func TestFileStoreRealRenameFailureCleansUpTemp(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "hidden.json")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "occupied"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	err := NewFileStore(path).Save(NewState())
	if err == nil || !strings.Contains(err.Error(), "rename") {
		t.Fatalf("Save() error = %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".hidden-*.tmp"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary files remain: %v, %v", matches, err)
	}
}

func TestFileStoreRejectsInvalidStateBeforeFilesystemMutation(t *testing.T) {
	t.Parallel()
	state := NewState()
	state.accounts["me"] = []Entry{{Kind: KindPullRequest, Organization: "a", Repository: "b", Number: 0}}
	fake := newFakeFS("")
	err := (&FileStore{path: "/state/hidden.json", fs: fake}).Save(state)
	if err == nil || !strings.Contains(err.Error(), "number") {
		t.Fatalf("Save() error = %v", err)
	}
	if len(fake.calls) != 0 {
		t.Fatalf("filesystem mutated before validation: %v", fake.calls)
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
	f.owner.calls = append(f.owner.calls, "chmod-"+f.kind)
	if f.owner.fail == "chmod-"+f.kind {
		return errors.New("chmod-" + f.kind)
	}
	return nil
}
func (f *fakeFile) Sync() error {
	f.owner.calls = append(f.owner.calls, "sync-"+f.kind)
	if f.owner.fail == "sync-"+f.kind {
		return errors.New("sync-" + f.kind)
	}
	return nil
}
func (f *fakeFile) Close() error {
	f.owner.calls = append(f.owner.calls, "close-"+f.kind)
	if f.owner.fail == "close-"+f.kind {
		return errors.New("close-" + f.kind)
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
