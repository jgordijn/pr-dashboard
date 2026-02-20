package config

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
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

// GHAccount represents an authenticated GitHub CLI account.
type GHAccount struct {
	Login  string // GitHub username (login)
	Active bool   // Whether this is the currently active account
}


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


// ErrNoAccounts is returned when no authenticated accounts are found.
var ErrNoAccounts = errors.New("no authenticated GitHub accounts found")

// accountPattern matches "Logged in to github.com account <login>" lines from gh auth status.
var accountPattern = regexp.MustCompile(`Logged in to github\.com account (\S+)`)

// activePattern matches "Active account: true" lines from gh auth status.
var activePattern = regexp.MustCompile(`Active account:\s*true`)

// ListGHAccounts returns all authenticated gh CLI accounts for github.com.
// It parses the output of `gh auth status --hostname github.com`.
func ListGHAccounts() ([]GHAccount, error) {
	if err := CheckGHCLI(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), ghAuthTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gh", "auth", "status", "--hostname", "github.com")
	output, err := cmd.CombinedOutput()
	cleanOutput := stripANSI(string(output))

	// gh auth status exits non-zero when not authenticated at all
	if err != nil && !strings.Contains(cleanOutput, "Logged in") {
		return nil, fmt.Errorf("%w: %s", ErrNoAccounts, strings.TrimSpace(cleanOutput))
	}

	return parseGHAuthStatus(cleanOutput)
}

// parseGHAuthStatus parses the output of `gh auth status` and extracts accounts.
// Exported for testing.
func parseGHAuthStatus(output string) ([]GHAccount, error) {
	lines := strings.Split(output, "\n")

	var accounts []GHAccount
	var currentLogin string

	for _, line := range lines {
		if m := accountPattern.FindStringSubmatch(line); m != nil {
			// If we had a previous account without an active line, add it as inactive
			if currentLogin != "" {
				accounts = append(accounts, GHAccount{Login: currentLogin, Active: false})
			}
			currentLogin = m[1]
			continue
		}

		if currentLogin != "" && activePattern.MatchString(line) {
			accounts = append(accounts, GHAccount{Login: currentLogin, Active: true})
			currentLogin = ""
			continue
		}

		if currentLogin != "" && strings.Contains(line, "Active account: false") {
			accounts = append(accounts, GHAccount{Login: currentLogin, Active: false})
			currentLogin = ""
			continue
		}
	}

	// Handle last account if no Active line followed
	if currentLogin != "" {
		accounts = append(accounts, GHAccount{Login: currentLogin, Active: false})
	}

	if len(accounts) == 0 {
		return nil, ErrNoAccounts
	}

	return accounts, nil
}

// SwitchGHAccount switches the active gh CLI account by running
// `gh auth switch --user <login>`.
func SwitchGHAccount(user string) error {
	if err := CheckGHCLI(); err != nil {
		return err
	}

	user = strings.TrimSpace(user)
	if user == "" {
		return fmt.Errorf("account login cannot be empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), ghAuthTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gh", "auth", "switch", "--user", user)
	output, err := cmd.CombinedOutput()
	if err != nil {
		cleanOutput := stripANSI(strings.TrimSpace(string(output)))
		if cleanOutput != "" {
			return fmt.Errorf("failed to switch account: %s", cleanOutput)
		}
		return fmt.Errorf("failed to switch account: %w", err)
	}

	return nil
}

// GHAuthToken returns the authentication token for a specific gh CLI account.
// It runs `gh auth token --user <login>` and returns the token string.
func GHAuthToken(user string) (string, error) {
	if err := CheckGHCLI(); err != nil {
		return "", err
	}

	user = strings.TrimSpace(user)
	if user == "" {
		return "", fmt.Errorf("account login cannot be empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), ghAuthTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gh", "auth", "token", "--user", user)
	output, err := cmd.CombinedOutput()
	if err != nil {
		cleanOutput := stripANSI(strings.TrimSpace(string(output)))
		if cleanOutput != "" {
			return "", fmt.Errorf("failed to get auth token: %s", cleanOutput)
		}
		return "", fmt.Errorf("failed to get auth token: %w", err)
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", fmt.Errorf("empty token returned for user %s", user)
	}

	return token, nil
}