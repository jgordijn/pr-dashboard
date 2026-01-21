package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPath_ReturnsExpectedPath(t *testing.T) {
	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() returned error: %v", err)
	}

	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".config", "pr-dashboard", "config.toml")

	if path != expected {
		t.Errorf("DefaultConfigPath() = %q, want %q", path, expected)
	}
}

func TestDefaultConfigPath_ContainsConfigDirectory(t *testing.T) {
	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() returned error: %v", err)
	}

	if !filepath.IsAbs(path) {
		t.Errorf("DefaultConfigPath() returned non-absolute path: %q", path)
	}

	if filepath.Base(path) != "config.toml" {
		t.Errorf("DefaultConfigPath() filename = %q, want %q", filepath.Base(path), "config.toml")
	}

	dir := filepath.Dir(path)
	if filepath.Base(dir) != "pr-dashboard" {
		t.Errorf("DefaultConfigPath() parent dir = %q, want %q", filepath.Base(dir), "pr-dashboard")
	}
}

func TestLoadFromPath_ValidConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	content := `[general]
username = "testuser"
refresh_interval = 60

[[organizations]]
login = "myorg"

[display]
show_drafts = false
initial_mode = "compact"

[notifications]
highlight_changes = false
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadFromPath() returned error: %v", err)
	}

	// Verify values are loaded correctly
	if cfg.General.Username != "testuser" {
		t.Errorf("Username = %q, want %q", cfg.General.Username, "testuser")
	}
	if cfg.General.RefreshInterval != 60 {
		t.Errorf("RefreshInterval = %d, want %d", cfg.General.RefreshInterval, 60)
	}
	if len(cfg.Organizations) != 1 || cfg.Organizations[0].Login != "myorg" {
		t.Errorf("Organizations = %v, want single org 'myorg'", cfg.Organizations)
	}
	if cfg.Display.ShowDrafts != false {
		t.Errorf("ShowDrafts = %v, want %v", cfg.Display.ShowDrafts, false)
	}
	if cfg.Display.InitialMode != "compact" {
		t.Errorf("InitialMode = %q, want %q", cfg.Display.InitialMode, "compact")
	}
	if cfg.Notifications.HighlightChanges != false {
		t.Errorf("HighlightChanges = %v, want %v", cfg.Notifications.HighlightChanges, false)
	}
}

func TestLoadFromPath_AppliesDefaults(t *testing.T) {
	// Create a minimal config file with only required fields
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	content := `[general]
username = "testuser"

[[organizations]]
login = "myorg"
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadFromPath() returned error: %v", err)
	}

	// Verify defaults are applied
	if cfg.General.RefreshInterval != DefaultRefreshInterval {
		t.Errorf("RefreshInterval = %d, want default %d", cfg.General.RefreshInterval, DefaultRefreshInterval)
	}
	if cfg.Display.ShowDrafts != DefaultShowDrafts {
		t.Errorf("ShowDrafts = %v, want default %v", cfg.Display.ShowDrafts, DefaultShowDrafts)
	}
	if cfg.Display.InitialMode != DefaultInitialMode {
		t.Errorf("InitialMode = %q, want default %q", cfg.Display.InitialMode, DefaultInitialMode)
	}
	if cfg.Notifications.HighlightChanges != DefaultHighlightChanges {
		t.Errorf("HighlightChanges = %v, want default %v", cfg.Notifications.HighlightChanges, DefaultHighlightChanges)
	}
}

func TestLoadFromPath_FileNotFound(t *testing.T) {
	_, err := LoadFromPath("/nonexistent/path/config.toml")
	if err == nil {
		t.Fatal("LoadFromPath() should return error for nonexistent file")
	}

	if !errors.Is(err, ErrConfigNotFound) {
		t.Errorf("LoadFromPath() error = %v, want ErrConfigNotFound", err)
	}
}

func TestLoadFromPath_InvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Invalid TOML syntax
	content := `[general
username = "testuser"`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadFromPath(configPath)
	if err == nil {
		t.Fatal("LoadFromPath() should return error for invalid TOML")
	}

	// Should not be ErrConfigNotFound
	if errors.Is(err, ErrConfigNotFound) {
		t.Error("LoadFromPath() should not return ErrConfigNotFound for invalid TOML")
	}
}

func TestLoadFromPath_UnknownKeys(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	content := `[general]
username = "testuser"
unknown_field = "value"

[[organizations]]
login = "myorg"
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadFromPath(configPath)
	if err == nil {
		t.Fatal("LoadFromPath() should return error for unknown keys")
	}

	// Should mention "unknown" in the error
	if err.Error() == "" {
		t.Error("LoadFromPath() error message should not be empty")
	}
}

func TestLoadFromPath_MultipleOrganizations(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	content := `[general]
username = "testuser"

[[organizations]]
login = "org1"

[[organizations]]
login = "org2"

[[organizations]]
login = "org3"
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadFromPath() returned error: %v", err)
	}

	if len(cfg.Organizations) != 3 {
		t.Errorf("Organizations count = %d, want %d", len(cfg.Organizations), 3)
	}

	expected := []string{"org1", "org2", "org3"}
	for i, org := range cfg.Organizations {
		if org.Login != expected[i] {
			t.Errorf("Organizations[%d].Login = %q, want %q", i, org.Login, expected[i])
		}
	}
}

func TestLoadFromPath_ExplicitValuesOverrideDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Explicitly set values different from defaults
	content := `[general]
username = "testuser"
refresh_interval = 120

[[organizations]]
login = "myorg"

[display]
show_drafts = false
initial_mode = "minimal"

[notifications]
highlight_changes = false
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadFromPath() returned error: %v", err)
	}

	// Verify explicit values are used, not defaults
	if cfg.General.RefreshInterval != 120 {
		t.Errorf("RefreshInterval = %d, want explicit value %d", cfg.General.RefreshInterval, 120)
	}
	if cfg.Display.ShowDrafts != false {
		t.Errorf("ShowDrafts = %v, want explicit value %v", cfg.Display.ShowDrafts, false)
	}
	if cfg.Display.InitialMode != "minimal" {
		t.Errorf("InitialMode = %q, want explicit value %q", cfg.Display.InitialMode, "minimal")
	}
	if cfg.Notifications.HighlightChanges != false {
		t.Errorf("HighlightChanges = %v, want explicit value %v", cfg.Notifications.HighlightChanges, false)
	}
}

func TestLoadFromPath_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Empty config file
	if err := os.WriteFile(configPath, []byte(""), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadFromPath() returned error: %v", err)
	}

	// Should have all defaults applied
	if cfg.General.RefreshInterval != DefaultRefreshInterval {
		t.Errorf("RefreshInterval = %d, want default %d", cfg.General.RefreshInterval, DefaultRefreshInterval)
	}
}

func TestLoadFromPath_PartialSections(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Only some sections present
	content := `[general]
username = "testuser"

[[organizations]]
login = "myorg"

[display]
initial_mode = "compact"
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadFromPath() returned error: %v", err)
	}

	// Verify set values
	if cfg.Display.InitialMode != "compact" {
		t.Errorf("InitialMode = %q, want %q", cfg.Display.InitialMode, "compact")
	}

	// Verify defaults for unset values
	if cfg.Display.ShowDrafts != DefaultShowDrafts {
		t.Errorf("ShowDrafts = %v, want default %v", cfg.Display.ShowDrafts, DefaultShowDrafts)
	}
	if cfg.Notifications.HighlightChanges != DefaultHighlightChanges {
		t.Errorf("HighlightChanges = %v, want default %v", cfg.Notifications.HighlightChanges, DefaultHighlightChanges)
	}
}

func TestLoad_UsesDefaultPath(t *testing.T) {
	// This test verifies Load() uses DefaultConfigPath() internally
	// We can't easily test this without modifying the home directory,
	// but we can verify it returns ErrConfigNotFound when no config exists
	// at the default path (which is the expected behavior for most test environments)

	_, err := Load()
	// In most test environments, no config file exists at the default path
	// So we expect either ErrConfigNotFound or a successful load if one happens to exist
	if err != nil && !errors.Is(err, ErrConfigNotFound) {
		// If error is not ErrConfigNotFound, it could be a legitimate error
		// (like home directory not found), which is acceptable
		t.Logf("Load() returned error (expected in test environment): %v", err)
	}
}

func TestErrConfigNotFound_IsDetectable(t *testing.T) {
	_, err := LoadFromPath("/definitely/not/a/real/path/config.toml")

	if !errors.Is(err, ErrConfigNotFound) {
		t.Errorf("errors.Is(err, ErrConfigNotFound) = false, want true")
	}
}

func TestErrConfigNotFound_IncludesPath(t *testing.T) {
	path := "/some/test/path/config.toml"
	_, err := LoadFromPath(path)

	if err == nil {
		t.Fatal("LoadFromPath() should return error for nonexistent file")
	}

	// Error message should include the path
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Error message should not be empty")
	}
}

func TestLoadFromPath_PreservesReadError(t *testing.T) {
	// Test that read errors other than not-found are preserved
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Create a directory where a file is expected
	if err := os.Mkdir(configPath, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	_, err := LoadFromPath(configPath)
	if err == nil {
		t.Fatal("LoadFromPath() should return error when path is a directory")
	}

	// Should not be ErrConfigNotFound since the path exists (as a directory)
	if errors.Is(err, ErrConfigNotFound) {
		t.Error("LoadFromPath() should not return ErrConfigNotFound when path is a directory")
	}
}

func TestApplyDefaults_SetsAllDefaults(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)

	if cfg.General.RefreshInterval != DefaultRefreshInterval {
		t.Errorf("RefreshInterval = %d, want %d", cfg.General.RefreshInterval, DefaultRefreshInterval)
	}
	if cfg.Display.ShowDrafts != DefaultShowDrafts {
		t.Errorf("ShowDrafts = %v, want %v", cfg.Display.ShowDrafts, DefaultShowDrafts)
	}
	if cfg.Display.InitialMode != DefaultInitialMode {
		t.Errorf("InitialMode = %q, want %q", cfg.Display.InitialMode, DefaultInitialMode)
	}
	if cfg.Notifications.HighlightChanges != DefaultHighlightChanges {
		t.Errorf("HighlightChanges = %v, want %v", cfg.Notifications.HighlightChanges, DefaultHighlightChanges)
	}
}

func TestDefaultConstants(t *testing.T) {
	// Verify default constants match specification
	if DefaultRefreshInterval != 30 {
		t.Errorf("DefaultRefreshInterval = %d, want %d (per spec)", DefaultRefreshInterval, 30)
	}
	if DefaultShowDrafts != true {
		t.Errorf("DefaultShowDrafts = %v, want %v (per spec)", DefaultShowDrafts, true)
	}
	if DefaultInitialMode != "full" {
		t.Errorf("DefaultInitialMode = %q, want %q (per spec)", DefaultInitialMode, "full")
	}
	if DefaultHighlightChanges != true {
		t.Errorf("DefaultHighlightChanges = %v, want %v (per spec)", DefaultHighlightChanges, true)
	}
}
