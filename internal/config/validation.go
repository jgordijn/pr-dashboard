package config

import (
	"errors"
	"fmt"
	"strings"
)

// Validation constraints for configuration values.
const (
	MinRefreshInterval = 10
	MaxRefreshInterval = 300
)

// Valid values for InitialMode.
var validInitialModes = map[string]bool{
	"full":    true,
	"compact": true,
	"minimal": true,
}

// ValidationError represents a configuration validation error with multiple issues.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "configuration validation failed"
	}
	if len(e.Errors) == 1 {
		return fmt.Sprintf("configuration validation failed: %s", e.Errors[0])
	}
	return fmt.Sprintf("configuration validation failed:\n  - %s", strings.Join(e.Errors, "\n  - "))
}

// Validate checks the configuration for errors and returns a ValidationError
// containing all validation issues found, or nil if the configuration is valid.
func Validate(cfg *Config) error {
	if cfg == nil {
		return errors.New("configuration is nil")
	}

	var errs []string

	// Validate username
	if strings.TrimSpace(cfg.General.Username) == "" {
		errs = append(errs, "general.username is required and cannot be empty")
	}

	// Validate organizations
	if len(cfg.Organizations) == 0 {
		errs = append(errs, "at least one [[organizations]] entry is required")
	} else {
		for i, org := range cfg.Organizations {
			if strings.TrimSpace(org.Login) == "" {
				errs = append(errs, fmt.Sprintf("organizations[%d].login is required and cannot be empty", i))
			}
		}
	}

	// Validate refresh_interval
	if cfg.General.RefreshInterval < MinRefreshInterval {
		errs = append(errs, fmt.Sprintf("general.refresh_interval must be at least %d seconds, got %d", MinRefreshInterval, cfg.General.RefreshInterval))
	} else if cfg.General.RefreshInterval > MaxRefreshInterval {
		errs = append(errs, fmt.Sprintf("general.refresh_interval must be at most %d seconds, got %d", MaxRefreshInterval, cfg.General.RefreshInterval))
	}

	// Validate initial_mode (normalize by trimming whitespace and lowercasing)
	initialMode := strings.ToLower(strings.TrimSpace(cfg.Display.InitialMode))
	if !validInitialModes[initialMode] {
		errs = append(errs, fmt.Sprintf("display.initial_mode must be one of 'full', 'compact', or 'minimal', got '%s'", cfg.Display.InitialMode))
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}

	return nil
}
