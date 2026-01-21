package config

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseOrganizations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single org",
			input:    "myorg",
			expected: []string{"myorg"},
		},
		{
			name:     "multiple orgs",
			input:    "org1,org2,org3",
			expected: []string{"org1", "org2", "org3"},
		},
		{
			name:     "orgs with whitespace",
			input:    "  org1 , org2  ,  org3  ",
			expected: []string{"org1", "org2", "org3"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: nil,
		},
		{
			name:     "empty entries filtered",
			input:    "org1,,org2,  ,org3",
			expected: []string{"org1", "org2", "org3"},
		},
		{
			name:     "single org with trailing comma",
			input:    "myorg,",
			expected: []string{"myorg"},
		},
		{
			name:     "single org with leading comma",
			input:    ",myorg",
			expected: []string{"myorg"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseOrganizations(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("ParseOrganizations(%q) = %v (len %d), want %v (len %d)",
					tt.input, result, len(result), tt.expected, len(tt.expected))
				return
			}
			for i, org := range result {
				if org != tt.expected[i] {
					t.Errorf("ParseOrganizations(%q)[%d] = %q, want %q",
						tt.input, i, org, tt.expected[i])
				}
			}
		})
	}
}

func TestParseRefreshInterval(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  int
		expectErr bool
	}{
		{
			name:     "valid value",
			input:    "30",
			expected: 30,
		},
		{
			name:     "minimum value",
			input:    "10",
			expected: 10,
		},
		{
			name:     "maximum value",
			input:    "300",
			expected: 300,
		},
		{
			name:     "value with whitespace",
			input:    "  60  ",
			expected: 60,
		},
		{
			name:      "below minimum",
			input:     "5",
			expectErr: true,
		},
		{
			name:      "above maximum",
			input:     "500",
			expectErr: true,
		},
		{
			name:      "not a number",
			input:     "abc",
			expectErr: true,
		},
		{
			name:      "empty string",
			input:     "",
			expectErr: true,
		},
		{
			name:      "negative number",
			input:     "-10",
			expectErr: true,
		},
		{
			name:      "floating point",
			input:     "30.5",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseRefreshInterval(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Errorf("ParseRefreshInterval(%q) expected error, got %d", tt.input, result)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseRefreshInterval(%q) unexpected error: %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("ParseRefreshInterval(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateConfigTOML(t *testing.T) {
	cfg := &Config{
		General: GeneralConfig{
			Username:        "testuser",
			RefreshInterval: 45,
		},
		Organizations: []OrganizationConfig{
			{Login: "org1"},
			{Login: "org2"},
		},
		Display: DisplayConfig{
			ShowDrafts:  true,
			InitialMode: "full",
		},
		Notifications: NotificationsConfig{
			HighlightChanges: true,
		},
	}

	result := GenerateConfigTOML(cfg)

	// Verify required content
	requiredStrings := []string{
		`username = "testuser"`,
		`refresh_interval = 45`,
		`[[organizations]]`,
		`login = "org1"`,
		`login = "org2"`,
		`show_drafts = true`,
		`initial_mode = "full"`,
		`highlight_changes = true`,
		`[general]`,
		`[display]`,
		`[notifications]`,
	}

	for _, s := range requiredStrings {
		if !strings.Contains(result, s) {
			t.Errorf("GenerateConfigTOML() missing expected content: %q\nGot:\n%s", s, result)
		}
	}

	// Verify the TOML is parseable
	_, err := loadFromString(result)
	if err != nil {
		t.Errorf("GenerateConfigTOML() produced invalid TOML: %v\nContent:\n%s", err, result)
	}
}

// loadFromString is a helper for testing that loads config from a TOML string.
func loadFromString(tomlContent string) (*Config, error) {
	// Create a temp file with the content
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(tmpFile, []byte(tomlContent), 0644); err != nil {
		return nil, err
	}

	return LoadFromPath(tmpFile)
}

func TestBuildConfig(t *testing.T) {
	w := NewWizardWithIO(nil, &bytes.Buffer{})

	cfg := w.buildConfig("myuser", []string{"org1", "org2"}, 60)

	if cfg.General.Username != "myuser" {
		t.Errorf("buildConfig username = %q, want %q", cfg.General.Username, "myuser")
	}

	if cfg.General.RefreshInterval != 60 {
		t.Errorf("buildConfig refresh_interval = %d, want %d", cfg.General.RefreshInterval, 60)
	}

	if len(cfg.Organizations) != 2 {
		t.Fatalf("buildConfig organizations count = %d, want 2", len(cfg.Organizations))
	}

	if cfg.Organizations[0].Login != "org1" {
		t.Errorf("buildConfig organizations[0] = %q, want %q", cfg.Organizations[0].Login, "org1")
	}

	if cfg.Organizations[1].Login != "org2" {
		t.Errorf("buildConfig organizations[1] = %q, want %q", cfg.Organizations[1].Login, "org2")
	}

	// Check defaults are applied
	if cfg.Display.ShowDrafts != DefaultShowDrafts {
		t.Errorf("buildConfig show_drafts = %t, want %t", cfg.Display.ShowDrafts, DefaultShowDrafts)
	}

	if cfg.Display.InitialMode != DefaultInitialMode {
		t.Errorf("buildConfig initial_mode = %q, want %q", cfg.Display.InitialMode, DefaultInitialMode)
	}

	if cfg.Notifications.HighlightChanges != DefaultHighlightChanges {
		t.Errorf("buildConfig highlight_changes = %t, want %t", cfg.Notifications.HighlightChanges, DefaultHighlightChanges)
	}
}

func TestGenerateConfigTOML_Roundtrip(t *testing.T) {
	// Create a config, generate TOML, parse it back, and verify
	original := &Config{
		General: GeneralConfig{
			Username:        "roundtrip-user",
			RefreshInterval: 120,
		},
		Organizations: []OrganizationConfig{
			{Login: "test-org-1"},
			{Login: "test-org-2"},
		},
		Display: DisplayConfig{
			ShowDrafts:  false,
			InitialMode: "compact",
		},
		Notifications: NotificationsConfig{
			HighlightChanges: false,
		},
	}

	toml := GenerateConfigTOML(original)
	parsed, err := loadFromString(toml)
	if err != nil {
		t.Fatalf("Failed to parse generated TOML: %v", err)
	}

	// Verify values match
	if parsed.General.Username != original.General.Username {
		t.Errorf("Username mismatch: %q != %q", parsed.General.Username, original.General.Username)
	}

	if parsed.General.RefreshInterval != original.General.RefreshInterval {
		t.Errorf("RefreshInterval mismatch: %d != %d", parsed.General.RefreshInterval, original.General.RefreshInterval)
	}

	if len(parsed.Organizations) != len(original.Organizations) {
		t.Fatalf("Organizations count mismatch: %d != %d", len(parsed.Organizations), len(original.Organizations))
	}

	for i, org := range parsed.Organizations {
		if org.Login != original.Organizations[i].Login {
			t.Errorf("Organizations[%d] mismatch: %q != %q", i, org.Login, original.Organizations[i].Login)
		}
	}

	if parsed.Display.ShowDrafts != original.Display.ShowDrafts {
		t.Errorf("ShowDrafts mismatch: %t != %t", parsed.Display.ShowDrafts, original.Display.ShowDrafts)
	}

	if parsed.Display.InitialMode != original.Display.InitialMode {
		t.Errorf("InitialMode mismatch: %q != %q", parsed.Display.InitialMode, original.Display.InitialMode)
	}

	if parsed.Notifications.HighlightChanges != original.Notifications.HighlightChanges {
		t.Errorf("HighlightChanges mismatch: %t != %t", parsed.Notifications.HighlightChanges, original.Notifications.HighlightChanges)
	}
}

func TestGenerateConfigTOML_SpecialCharacters(t *testing.T) {
	// Test that special characters in usernames/orgs are properly escaped
	cfg := &Config{
		General: GeneralConfig{
			Username:        `user"with"quotes`,
			RefreshInterval: 30,
		},
		Organizations: []OrganizationConfig{
			{Login: `org-with-"quotes"`},
		},
		Display: DisplayConfig{
			ShowDrafts:  true,
			InitialMode: "full",
		},
		Notifications: NotificationsConfig{
			HighlightChanges: true,
		},
	}

	toml := GenerateConfigTOML(cfg)

	// Verify the TOML is parseable (quotes should be escaped)
	parsed, err := loadFromString(toml)
	if err != nil {
		t.Fatalf("Failed to parse TOML with special characters: %v\nContent:\n%s", err, toml)
	}

	if parsed.General.Username != cfg.General.Username {
		t.Errorf("Username with quotes mismatch: %q != %q", parsed.General.Username, cfg.General.Username)
	}

	if parsed.Organizations[0].Login != cfg.Organizations[0].Login {
		t.Errorf("Org with quotes mismatch: %q != %q", parsed.Organizations[0].Login, cfg.Organizations[0].Login)
	}
}

func TestErrWizardCancelled(t *testing.T) {
	if ErrWizardCancelled == nil {
		t.Error("ErrWizardCancelled should not be nil")
	}

	expectedMsg := "wizard cancelled by user"
	if ErrWizardCancelled.Error() != expectedMsg {
		t.Errorf("ErrWizardCancelled.Error() = %q, want %q", ErrWizardCancelled.Error(), expectedMsg)
	}

	// Verify errors.Is works
	if !errors.Is(ErrWizardCancelled, ErrWizardCancelled) {
		t.Error("errors.Is should return true for ErrWizardCancelled")
	}
}

func TestNewWizard(t *testing.T) {
	w := NewWizard()
	if w == nil {
		t.Error("NewWizard() should not return nil")
	}
	if w.in != os.Stdin {
		t.Error("NewWizard().in should be os.Stdin")
	}
	if w.out != os.Stdout {
		t.Error("NewWizard().out should be os.Stdout")
	}
}

func TestNewWizardWithIO(t *testing.T) {
	in := strings.NewReader("test")
	out := &bytes.Buffer{}

	w := NewWizardWithIO(in, out)
	if w == nil {
		t.Error("NewWizardWithIO() should not return nil")
	}
	if w.in != in {
		t.Error("NewWizardWithIO().in should be the provided reader")
	}
	if w.out != out {
		t.Error("NewWizardWithIO().out should be the provided writer")
	}
}

func TestWizard_PrintWelcome(t *testing.T) {
	out := &bytes.Buffer{}
	w := NewWizardWithIO(nil, out)

	w.printWelcome()

	output := out.String()
	if !strings.Contains(output, "Welcome to pr-dashboard") {
		t.Error("printWelcome() should contain welcome message")
	}
	if !strings.Contains(output, "Ctrl+D") {
		t.Error("printWelcome() should mention Ctrl+D (EOF) to cancel")
	}
}

func TestWizard_PrintSuccess(t *testing.T) {
	out := &bytes.Buffer{}
	w := NewWizardWithIO(nil, out)

	w.printSuccess()

	output := out.String()
	if !strings.Contains(output, "Setup complete") {
		t.Error("printSuccess() should contain success message")
	}
}

func TestParseOrganizations_PreservesCase(t *testing.T) {
	// Ensure organization names preserve their case
	input := "MyOrg,UPPER,lower"
	orgs := ParseOrganizations(input)

	expected := []string{"MyOrg", "UPPER", "lower"}
	if len(orgs) != len(expected) {
		t.Fatalf("ParseOrganizations() count = %d, want %d", len(orgs), len(expected))
	}

	for i, org := range orgs {
		if org != expected[i] {
			t.Errorf("ParseOrganizations()[%d] = %q, want %q (case should be preserved)", i, org, expected[i])
		}
	}
}

func TestParseRefreshInterval_BoundaryValues(t *testing.T) {
	// Test just below minimum
	_, err := ParseRefreshInterval("9")
	if err == nil {
		t.Error("ParseRefreshInterval(9) should fail (below minimum)")
	}

	// Test just above maximum
	_, err = ParseRefreshInterval("301")
	if err == nil {
		t.Error("ParseRefreshInterval(301) should fail (above maximum)")
	}
}

func TestGenerateConfigTOML_SingleOrganization(t *testing.T) {
	cfg := &Config{
		General: GeneralConfig{
			Username:        "user",
			RefreshInterval: 30,
		},
		Organizations: []OrganizationConfig{
			{Login: "single-org"},
		},
		Display: DisplayConfig{
			ShowDrafts:  true,
			InitialMode: "full",
		},
		Notifications: NotificationsConfig{
			HighlightChanges: true,
		},
	}

	toml := GenerateConfigTOML(cfg)

	// Should have exactly one [[organizations]] block
	count := strings.Count(toml, "[[organizations]]")
	if count != 1 {
		t.Errorf("Expected 1 [[organizations]] block, got %d", count)
	}

	// Verify parseable
	parsed, err := loadFromString(toml)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(parsed.Organizations) != 1 {
		t.Errorf("Parsed organizations count = %d, want 1", len(parsed.Organizations))
	}
}

func TestGenerateConfigTOML_ManyOrganizations(t *testing.T) {
	orgs := make([]OrganizationConfig, 10)
	for i := 0; i < 10; i++ {
		orgs[i] = OrganizationConfig{Login: strings.Repeat("org", i+1)}
	}

	cfg := &Config{
		General: GeneralConfig{
			Username:        "user",
			RefreshInterval: 30,
		},
		Organizations: orgs,
		Display: DisplayConfig{
			ShowDrafts:  true,
			InitialMode: "full",
		},
		Notifications: NotificationsConfig{
			HighlightChanges: true,
		},
	}

	toml := GenerateConfigTOML(cfg)

	// Should have 10 [[organizations]] blocks
	count := strings.Count(toml, "[[organizations]]")
	if count != 10 {
		t.Errorf("Expected 10 [[organizations]] blocks, got %d", count)
	}

	// Verify parseable
	parsed, err := loadFromString(toml)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(parsed.Organizations) != 10 {
		t.Errorf("Parsed organizations count = %d, want 10", len(parsed.Organizations))
	}
}

func TestBuildConfig_EmptyOrganizations(t *testing.T) {
	w := NewWizardWithIO(nil, &bytes.Buffer{})

	cfg := w.buildConfig("user", []string{}, 30)

	if len(cfg.Organizations) != 0 {
		t.Errorf("buildConfig with empty orgs should have 0 organizations, got %d", len(cfg.Organizations))
	}
}

func TestBuildConfig_SingleOrganization(t *testing.T) {
	w := NewWizardWithIO(nil, &bytes.Buffer{})

	cfg := w.buildConfig("user", []string{"myorg"}, 30)

	if len(cfg.Organizations) != 1 {
		t.Fatalf("buildConfig organizations count = %d, want 1", len(cfg.Organizations))
	}

	if cfg.Organizations[0].Login != "myorg" {
		t.Errorf("buildConfig organizations[0] = %q, want %q", cfg.Organizations[0].Login, "myorg")
	}
}

func TestGenerateConfigTOML_EndsWithNewline(t *testing.T) {
	cfg := &Config{
		General: GeneralConfig{
			Username:        "user",
			RefreshInterval: 30,
		},
		Organizations: []OrganizationConfig{
			{Login: "org1"},
		},
		Display: DisplayConfig{
			ShowDrafts:  true,
			InitialMode: "full",
		},
		Notifications: NotificationsConfig{
			HighlightChanges: true,
		},
	}

	toml := GenerateConfigTOML(cfg)

	if !strings.HasSuffix(toml, "\n") {
		t.Error("GenerateConfigTOML() should end with a newline")
	}
}

func TestErrConfigExists(t *testing.T) {
	if ErrConfigExists == nil {
		t.Error("ErrConfigExists should not be nil")
	}

	expectedMsg := "configuration file already exists"
	if ErrConfigExists.Error() != expectedMsg {
		t.Errorf("ErrConfigExists.Error() = %q, want %q", ErrConfigExists.Error(), expectedMsg)
	}

	// Verify errors.Is works
	if !errors.Is(ErrConfigExists, ErrConfigExists) {
		t.Error("errors.Is should return true for ErrConfigExists")
	}
}

// Tests for interactive Run() flow using scripted input

func TestWizard_PromptUsername_Success(t *testing.T) {
	in := strings.NewReader("testuser\n")
	out := &bytes.Buffer{}
	w := NewWizardWithIO(in, out)

	scanner := bufio.NewScanner(in)
	username, err := w.promptUsername(scanner)

	if err != nil {
		t.Fatalf("promptUsername() unexpected error: %v", err)
	}
	if username != "testuser" {
		t.Errorf("promptUsername() = %q, want %q", username, "testuser")
	}
}

func TestWizard_PromptUsername_RetryOnEmpty(t *testing.T) {
	// First line empty, second line has username
	in := strings.NewReader("\ntestuser\n")
	out := &bytes.Buffer{}
	w := NewWizardWithIO(in, out)

	scanner := bufio.NewScanner(in)
	username, err := w.promptUsername(scanner)

	if err != nil {
		t.Fatalf("promptUsername() unexpected error: %v", err)
	}
	if username != "testuser" {
		t.Errorf("promptUsername() = %q, want %q", username, "testuser")
	}
	if !strings.Contains(out.String(), "Error") {
		t.Error("promptUsername() should show error message for empty input")
	}
}

func TestWizard_PromptUsername_EOF(t *testing.T) {
	in := strings.NewReader("") // EOF immediately
	out := &bytes.Buffer{}
	w := NewWizardWithIO(in, out)

	scanner := bufio.NewScanner(in)
	_, err := w.promptUsername(scanner)

	if !errors.Is(err, ErrWizardCancelled) {
		t.Errorf("promptUsername() error = %v, want ErrWizardCancelled", err)
	}
}

func TestWizard_PromptOrganizations_Success(t *testing.T) {
	in := strings.NewReader("org1, org2\n")
	out := &bytes.Buffer{}
	w := NewWizardWithIO(in, out)

	scanner := bufio.NewScanner(in)
	orgs, err := w.promptOrganizations(scanner)

	if err != nil {
		t.Fatalf("promptOrganizations() unexpected error: %v", err)
	}
	if len(orgs) != 2 {
		t.Fatalf("promptOrganizations() count = %d, want 2", len(orgs))
	}
	if orgs[0] != "org1" || orgs[1] != "org2" {
		t.Errorf("promptOrganizations() = %v, want [org1, org2]", orgs)
	}
}

func TestWizard_PromptOrganizations_RetryOnEmpty(t *testing.T) {
	// First line empty, second line has orgs
	in := strings.NewReader("\norg1\n")
	out := &bytes.Buffer{}
	w := NewWizardWithIO(in, out)

	scanner := bufio.NewScanner(in)
	orgs, err := w.promptOrganizations(scanner)

	if err != nil {
		t.Fatalf("promptOrganizations() unexpected error: %v", err)
	}
	if len(orgs) != 1 || orgs[0] != "org1" {
		t.Errorf("promptOrganizations() = %v, want [org1]", orgs)
	}
	if !strings.Contains(out.String(), "Error") {
		t.Error("promptOrganizations() should show error message for empty input")
	}
}

func TestWizard_PromptOrganizations_EOF(t *testing.T) {
	in := strings.NewReader("") // EOF immediately
	out := &bytes.Buffer{}
	w := NewWizardWithIO(in, out)

	scanner := bufio.NewScanner(in)
	_, err := w.promptOrganizations(scanner)

	if !errors.Is(err, ErrWizardCancelled) {
		t.Errorf("promptOrganizations() error = %v, want ErrWizardCancelled", err)
	}
}

func TestWizard_PromptRefreshInterval_Default(t *testing.T) {
	in := strings.NewReader("\n") // Empty input for default
	out := &bytes.Buffer{}
	w := NewWizardWithIO(in, out)

	scanner := bufio.NewScanner(in)
	interval, err := w.promptRefreshInterval(scanner)

	if err != nil {
		t.Fatalf("promptRefreshInterval() unexpected error: %v", err)
	}
	if interval != DefaultRefreshInterval {
		t.Errorf("promptRefreshInterval() = %d, want %d", interval, DefaultRefreshInterval)
	}
}

func TestWizard_PromptRefreshInterval_CustomValue(t *testing.T) {
	in := strings.NewReader("60\n")
	out := &bytes.Buffer{}
	w := NewWizardWithIO(in, out)

	scanner := bufio.NewScanner(in)
	interval, err := w.promptRefreshInterval(scanner)

	if err != nil {
		t.Fatalf("promptRefreshInterval() unexpected error: %v", err)
	}
	if interval != 60 {
		t.Errorf("promptRefreshInterval() = %d, want 60", interval)
	}
}

func TestWizard_PromptRefreshInterval_RetryOnInvalid(t *testing.T) {
	// First: invalid value (too low), second: valid value
	in := strings.NewReader("5\n60\n")
	out := &bytes.Buffer{}
	w := NewWizardWithIO(in, out)

	scanner := bufio.NewScanner(in)
	interval, err := w.promptRefreshInterval(scanner)

	if err != nil {
		t.Fatalf("promptRefreshInterval() unexpected error: %v", err)
	}
	if interval != 60 {
		t.Errorf("promptRefreshInterval() = %d, want 60", interval)
	}
	if !strings.Contains(out.String(), "Error") {
		t.Error("promptRefreshInterval() should show error message for invalid input")
	}
}

func TestWizard_PromptRefreshInterval_RetryOnNonNumeric(t *testing.T) {
	// First: non-numeric, second: valid value
	in := strings.NewReader("abc\n45\n")
	out := &bytes.Buffer{}
	w := NewWizardWithIO(in, out)

	scanner := bufio.NewScanner(in)
	interval, err := w.promptRefreshInterval(scanner)

	if err != nil {
		t.Fatalf("promptRefreshInterval() unexpected error: %v", err)
	}
	if interval != 45 {
		t.Errorf("promptRefreshInterval() = %d, want 45", interval)
	}
	if !strings.Contains(out.String(), "Error") {
		t.Error("promptRefreshInterval() should show error message for non-numeric input")
	}
}

func TestWizard_PromptRefreshInterval_EOF(t *testing.T) {
	in := strings.NewReader("") // EOF immediately
	out := &bytes.Buffer{}
	w := NewWizardWithIO(in, out)

	scanner := bufio.NewScanner(in)
	_, err := w.promptRefreshInterval(scanner)

	if !errors.Is(err, ErrWizardCancelled) {
		t.Errorf("promptRefreshInterval() error = %v, want ErrWizardCancelled", err)
	}
}

func TestWriteConfigToPath_RefusesOverwrite(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "wizard-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.toml")

	// Create an existing config file
	if err := os.WriteFile(configPath, []byte("existing config"), 0644); err != nil {
		t.Fatalf("Failed to create existing config: %v", err)
	}

	// Try to write a new config - should fail
	cfg := &Config{
		General: GeneralConfig{
			Username:        "testuser",
			RefreshInterval: 30,
		},
		Organizations: []OrganizationConfig{{Login: "testorg"}},
		Display: DisplayConfig{
			ShowDrafts:  true,
			InitialMode: "full",
		},
		Notifications: NotificationsConfig{
			HighlightChanges: true,
		},
	}

	err = WriteConfigToPath(configPath, cfg)
	if err == nil {
		t.Fatal("WriteConfigToPath should fail when file exists")
	}

	if !errors.Is(err, ErrConfigExists) {
		t.Errorf("WriteConfigToPath error should wrap ErrConfigExists, got: %v", err)
	}

	// Verify original file wasn't modified
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}
	if string(content) != "existing config" {
		t.Errorf("Original config file was modified: got %q", string(content))
	}
}

func TestWriteConfigToPath_CreatesDirectoryAndFile(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "wizard-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use a nested path that doesn't exist yet
	configPath := filepath.Join(tmpDir, "subdir", "nested", "config.toml")

	cfg := &Config{
		General: GeneralConfig{
			Username:        "testuser",
			RefreshInterval: 45,
		},
		Organizations: []OrganizationConfig{
			{Login: "org1"},
			{Login: "org2"},
		},
		Display: DisplayConfig{
			ShowDrafts:  false,
			InitialMode: "compact",
		},
		Notifications: NotificationsConfig{
			HighlightChanges: false,
		},
	}

	err = WriteConfigToPath(configPath, cfg)
	if err != nil {
		t.Fatalf("WriteConfigToPath failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Verify the config can be loaded back
	loaded, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("Failed to load written config: %v", err)
	}

	if loaded.General.Username != cfg.General.Username {
		t.Errorf("Username mismatch: got %q, want %q", loaded.General.Username, cfg.General.Username)
	}
	if loaded.General.RefreshInterval != cfg.General.RefreshInterval {
		t.Errorf("RefreshInterval mismatch: got %d, want %d", loaded.General.RefreshInterval, cfg.General.RefreshInterval)
	}
	if len(loaded.Organizations) != len(cfg.Organizations) {
		t.Errorf("Organizations count mismatch: got %d, want %d", len(loaded.Organizations), len(cfg.Organizations))
	}
}

func TestWriteConfigToPath_RestrictivePermissions(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "wizard-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configDir := filepath.Join(tmpDir, "newdir")
	configPath := filepath.Join(configDir, "config.toml")

	cfg := &Config{
		General: GeneralConfig{
			Username:        "testuser",
			RefreshInterval: 30,
		},
		Organizations: []OrganizationConfig{{Login: "testorg"}},
		Display: DisplayConfig{
			ShowDrafts:  true,
			InitialMode: "full",
		},
		Notifications: NotificationsConfig{
			HighlightChanges: true,
		},
	}

	err = WriteConfigToPath(configPath, cfg)
	if err != nil {
		t.Fatalf("WriteConfigToPath failed: %v", err)
	}

	// Verify directory permissions (0700)
	dirInfo, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("Failed to stat config dir: %v", err)
	}
	dirPerm := dirInfo.Mode().Perm()
	if dirPerm != 0700 {
		t.Errorf("Config directory permissions = %o, want 0700", dirPerm)
	}

	// Verify file permissions (0600)
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}
	filePerm := fileInfo.Mode().Perm()
	if filePerm != 0600 {
		t.Errorf("Config file permissions = %o, want 0600", filePerm)
	}
}
