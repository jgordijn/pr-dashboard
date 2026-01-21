package config

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

// Sentinel errors for gh CLI checks.
var (
	// ErrGHCLINotFound is returned when the gh CLI is not installed.
	// Use errors.Is(err, ErrGHCLINotFound) to check for this condition.
	ErrGHCLINotFound = errors.New("gh CLI not found. Install from https://cli.github.com")

	// ErrGHNotAuthenticated is returned when the gh CLI is not authenticated.
	// Use errors.Is(err, ErrGHNotAuthenticated) to check for this condition.
	ErrGHNotAuthenticated = errors.New("gh CLI not authenticated. Run `gh auth login` first")
)

// CheckGHCLI verifies that the gh CLI is installed and available in PATH.
// Returns an error wrapping ErrGHCLINotFound if gh is not found.
// The underlying exec.LookPath error is accessible via errors.Unwrap/errors.As.
func CheckGHCLI() error {
	_, err := exec.LookPath("gh")
	if err != nil {
		return errors.Join(ErrGHCLINotFound, err)
	}
	return nil
}

// ghAuthTimeout is the timeout for the gh auth status command.
const ghAuthTimeout = 5 * time.Second

// ghAuthError is a composite error that wraps both the sentinel error and the
// underlying execution error. Callers can use errors.Is(err, ErrGHNotAuthenticated)
// and also inspect the underlying error with errors.Unwrap or errors.As.
type ghAuthError struct {
	output string // combined stdout/stderr from gh auth status
	err    error  // underlying execution error
}

func (e *ghAuthError) Error() string {
	msg := ErrGHNotAuthenticated.Error()
	if e.output != "" {
		msg += ": " + e.output
	} else if e.err != nil {
		msg += ": " + e.err.Error()
	}
	return msg
}

// Unwrap returns both errors so callers can use errors.Is/errors.As on either.
func (e *ghAuthError) Unwrap() []error {
	return []error{ErrGHNotAuthenticated, e.err}
}

// CheckGHAuth verifies that the gh CLI is authenticated with GitHub.
// It first checks if gh CLI is installed, then verifies authentication status.
// Returns ErrGHCLINotFound if gh is not installed.
// Returns an error wrapping ErrGHNotAuthenticated if gh is not authenticated,
// with the underlying execution error available via errors.Unwrap/errors.As.
func CheckGHAuth() error {
	// First verify gh CLI is installed
	if err := CheckGHCLI(); err != nil {
		return err
	}

	// Check authentication status with a timeout to prevent hanging
	// gh auth status returns exit code 0 if authenticated, non-zero otherwise
	ctx, cancel := context.WithTimeout(context.Background(), ghAuthTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gh", "auth", "status", "--hostname", "github.com")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Strip ANSI codes from gh output for clean error display
		cleanOutput := stripANSI(strings.TrimSpace(string(output)))
		return &ghAuthError{
			output: cleanOutput,
			err:    err,
		}
	}

	return nil
}
