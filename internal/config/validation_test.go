package config

import (
	"errors"
	"strings"
	"testing"
)

// validConfig returns a minimal valid configuration for testing.
func validConfig() *Config {
	return &Config{
		General: GeneralConfig{
			Username:        "testuser",
			RefreshInterval: DefaultRefreshInterval,
		},
		Organizations: []OrganizationConfig{
			{Login: "test-org"},
		},
		Display: DisplayConfig{
			ShowDrafts:  true,
			InitialMode: DefaultInitialMode,
		},
		Notifications: NotificationsConfig{
			HighlightChanges: true,
		},
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := validConfig()
	if err := Validate(cfg); err != nil {
		t.Errorf("expected valid config to pass validation, got error: %v", err)
	}
}

func TestValidate_NilConfig(t *testing.T) {
	err := Validate(nil)
	if err == nil {
		t.Error("expected error for nil config, got nil")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("expected error to mention 'nil', got: %v", err)
	}
}

func TestValidate_MissingUsername(t *testing.T) {
	cfg := validConfig()
	cfg.General.Username = ""

	err := Validate(cfg)
	if err == nil {
		t.Error("expected error for missing username, got nil")
	}
	if !strings.Contains(err.Error(), "username") {
		t.Errorf("expected error to mention 'username', got: %v", err)
	}
}

func TestValidate_WhitespaceOnlyUsername(t *testing.T) {
	cfg := validConfig()
	cfg.General.Username = "   \t\n  "

	err := Validate(cfg)
	if err == nil {
		t.Error("expected error for whitespace-only username, got nil")
	}
	if !strings.Contains(err.Error(), "username") {
		t.Errorf("expected error to mention 'username', got: %v", err)
	}
}

func TestValidate_NoOrganizations(t *testing.T) {
	cfg := validConfig()
	cfg.Organizations = nil

	err := Validate(cfg)
	if err == nil {
		t.Error("expected error for no organizations, got nil")
	}
	if !strings.Contains(err.Error(), "organizations") {
		t.Errorf("expected error to mention 'organizations', got: %v", err)
	}
}

func TestValidate_EmptyOrganizationsSlice(t *testing.T) {
	cfg := validConfig()
	cfg.Organizations = []OrganizationConfig{}

	err := Validate(cfg)
	if err == nil {
		t.Error("expected error for empty organizations slice, got nil")
	}
	if !strings.Contains(err.Error(), "organizations") {
		t.Errorf("expected error to mention 'organizations', got: %v", err)
	}
}

func TestValidate_OrganizationEmptyLogin(t *testing.T) {
	cfg := validConfig()
	cfg.Organizations = []OrganizationConfig{
		{Login: "valid-org"},
		{Login: ""},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("expected error for organization with empty login, got nil")
	}
	if !strings.Contains(err.Error(), "organizations[1].login") {
		t.Errorf("expected error to mention 'organizations[1].login', got: %v", err)
	}
}

func TestValidate_OrganizationWhitespaceLogin(t *testing.T) {
	cfg := validConfig()
	cfg.Organizations = []OrganizationConfig{
		{Login: "   "},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("expected error for organization with whitespace login, got nil")
	}
	if !strings.Contains(err.Error(), "organizations[0].login") {
		t.Errorf("expected error to mention 'organizations[0].login', got: %v", err)
	}
}

func TestValidate_RefreshIntervalBelowMinimum(t *testing.T) {
	cfg := validConfig()
	cfg.General.RefreshInterval = 5

	err := Validate(cfg)
	if err == nil {
		t.Error("expected error for refresh_interval below minimum, got nil")
	}
	if !strings.Contains(err.Error(), "refresh_interval") {
		t.Errorf("expected error to mention 'refresh_interval', got: %v", err)
	}
	if !strings.Contains(err.Error(), "at least 10") {
		t.Errorf("expected error to mention minimum value, got: %v", err)
	}
}

func TestValidate_RefreshIntervalAboveMaximum(t *testing.T) {
	cfg := validConfig()
	cfg.General.RefreshInterval = 500

	err := Validate(cfg)
	if err == nil {
		t.Error("expected error for refresh_interval above maximum, got nil")
	}
	if !strings.Contains(err.Error(), "refresh_interval") {
		t.Errorf("expected error to mention 'refresh_interval', got: %v", err)
	}
	if !strings.Contains(err.Error(), "at most 300") {
		t.Errorf("expected error to mention maximum value, got: %v", err)
	}
}

func TestValidate_RefreshIntervalAtMinimum(t *testing.T) {
	cfg := validConfig()
	cfg.General.RefreshInterval = MinRefreshInterval // 10

	if err := Validate(cfg); err != nil {
		t.Errorf("expected refresh_interval=%d (minimum) to pass, got error: %v", MinRefreshInterval, err)
	}
}

func TestValidate_RefreshIntervalAtMaximum(t *testing.T) {
	cfg := validConfig()
	cfg.General.RefreshInterval = MaxRefreshInterval // 300

	if err := Validate(cfg); err != nil {
		t.Errorf("expected refresh_interval=%d (maximum) to pass, got error: %v", MaxRefreshInterval, err)
	}
}

func TestValidate_InvalidInitialMode(t *testing.T) {
	// These remain invalid even after normalization (not valid mode names)
	testCases := []string{"", "invalid", "large", "small", "medium"}

	for _, mode := range testCases {
		t.Run("mode="+mode, func(t *testing.T) {
			cfg := validConfig()
			cfg.Display.InitialMode = mode

			err := Validate(cfg)
			if err == nil {
				t.Errorf("expected error for initial_mode=%q, got nil", mode)
			}
			if !strings.Contains(err.Error(), "initial_mode") {
				t.Errorf("expected error to mention 'initial_mode', got: %v", err)
			}
		})
	}
}

func TestValidate_ValidInitialModes(t *testing.T) {
	validModes := []string{"full", "compact", "minimal"}

	for _, mode := range validModes {
		t.Run("mode="+mode, func(t *testing.T) {
			cfg := validConfig()
			cfg.Display.InitialMode = mode

			if err := Validate(cfg); err != nil {
				t.Errorf("expected initial_mode=%q to be valid, got error: %v", mode, err)
			}
		})
	}
}

func TestValidate_InitialModeWithWhitespaceAndCase(t *testing.T) {
	// These should all be normalized and accepted
	testCases := []string{"FULL", "Full", " full", "full ", " full ", "COMPACT", " minimal"}

	for _, mode := range testCases {
		t.Run("mode="+mode, func(t *testing.T) {
			cfg := validConfig()
			cfg.Display.InitialMode = mode

			if err := Validate(cfg); err != nil {
				t.Errorf("expected initial_mode=%q to be valid after normalization, got error: %v", mode, err)
			}
		})
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	cfg := &Config{
		General: GeneralConfig{
			Username:        "",
			RefreshInterval: 5,
		},
		Organizations: []OrganizationConfig{},
		Display: DisplayConfig{
			InitialMode: "invalid",
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("expected multiple errors, got nil")
	}

	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	// Should have 4 errors: username, organizations, refresh_interval, initial_mode
	if len(validationErr.Errors) != 4 {
		t.Errorf("expected 4 validation errors, got %d: %v", len(validationErr.Errors), validationErr.Errors)
	}
}

func TestValidationError_SingleError(t *testing.T) {
	err := &ValidationError{Errors: []string{"single error"}}
	msg := err.Error()

	if !strings.Contains(msg, "single error") {
		t.Errorf("expected message to contain 'single error', got: %s", msg)
	}
	// Single error should not have bullet points
	if strings.Contains(msg, "\n") {
		t.Errorf("single error should not have newlines, got: %s", msg)
	}
}

func TestValidationError_MultipleErrors(t *testing.T) {
	err := &ValidationError{Errors: []string{"error one", "error two"}}
	msg := err.Error()

	if !strings.Contains(msg, "error one") || !strings.Contains(msg, "error two") {
		t.Errorf("expected message to contain both errors, got: %s", msg)
	}
	// Multiple errors should be formatted with newlines
	if !strings.Contains(msg, "\n") {
		t.Errorf("multiple errors should have newlines for formatting, got: %s", msg)
	}
}

func TestValidationError_EmptyErrors(t *testing.T) {
	err := &ValidationError{Errors: []string{}}
	msg := err.Error()

	expected := "configuration validation failed"
	if msg != expected {
		t.Errorf("expected %q for empty errors, got: %s", expected, msg)
	}
}
