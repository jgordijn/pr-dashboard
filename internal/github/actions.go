package github

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// updateBranchTimeout is the default timeout for the gh pr update-branch command.
// Update-branch can take time as it needs to merge/rebase from the base branch.
// 60 seconds provides reasonable buffer for repos with many commits behind.
const updateBranchTimeout = 60 * time.Second

// maxOutputLen is the maximum length of command output to include in error messages.
// This prevents "wall of text" errors while preserving diagnostic info.
const maxOutputLen = 500

// Sentinel errors for update-branch operations.
var (
	// ErrGHCLINotFound is returned when the gh CLI is not installed.
	// Use errors.Is(err, ErrGHCLINotFound) to check for this condition.
	ErrGHCLINotFound = errors.New("gh CLI not found")

	// ErrUpdateBranchTimeout is returned when the update-branch command times out.
	// Use errors.Is(err, ErrUpdateBranchTimeout) to check for this condition.
	ErrUpdateBranchTimeout = errors.New("update-branch command timed out")

	// ErrUpdateBranchFailed is returned when the update-branch command fails.
	// Use errors.Is(err, ErrUpdateBranchFailed) to check for this condition.
	ErrUpdateBranchFailed = errors.New("update-branch command failed")

	// ErrUpdateBranchCancelled is returned when the operation was cancelled by the caller.
	// Use errors.Is(err, ErrUpdateBranchCancelled) to check for this condition.
	ErrUpdateBranchCancelled = errors.New("update-branch command cancelled")

	// ErrInvalidArgument is returned when owner, repo, or prNumber are invalid.
	// Use errors.Is(err, ErrInvalidArgument) to check for this condition.
	ErrInvalidArgument = errors.New("invalid argument")
)

// updateBranchError is a composite error that wraps both the sentinel error and the
// underlying execution error. Callers can use errors.Is(err, ErrUpdateBranchFailed)
// and also inspect the underlying error with errors.Unwrap or errors.As.
type updateBranchError struct {
	sentinel error  // one of the sentinel errors
	output   string // combined stdout/stderr from the command
	err      error  // underlying execution error
}

func (e *updateBranchError) Error() string {
	msg := e.sentinel.Error()
	if e.output != "" {
		msg += ": " + e.output
	} else if e.err != nil {
		msg += ": " + e.err.Error()
	}
	return msg
}

// Unwrap returns both errors so callers can use errors.Is/errors.As on either.
// Nil entries are filtered out for cleaner error chain traversal.
func (e *updateBranchError) Unwrap() []error {
	result := make([]error, 0, 2)
	if e.sentinel != nil {
		result = append(result, e.sentinel)
	}
	if e.err != nil {
		result = append(result, e.err)
	}
	return result
}

// truncateOutput strips ANSI escape sequences, truncates output to maxOutputLen,
// and adds ellipsis if truncated. ANSI stripping is done first to ensure clean
// display in the TUI and accurate length calculation.
func truncateOutput(output string) string {
	// Strip ANSI escape sequences first for clean display
	cleaned := StripANSI(output)
	if len(cleaned) <= maxOutputLen {
		return cleaned
	}
	return cleaned[:maxOutputLen] + "..."
}

// UpdateBranch updates a pull request branch by merging the base branch into it.
// This executes `gh pr update-branch <number> --repo <owner/name>`.
//
// Parameters:
//   - ctx: Context for cancellation. If the context has no deadline, a default
//     timeout of 60 seconds is applied. If the context already has a deadline,
//     that deadline is respected (no additional timeout is added).
//   - owner: Repository owner (e.g., "octocat"). Must be non-empty.
//   - repo: Repository name (e.g., "hello-world"). Must be non-empty.
//   - prNumber: Pull request number. Must be greater than 0.
//
// Returns:
//   - ErrInvalidArgument if owner, repo, or prNumber are invalid
//   - ErrGHCLINotFound if gh is not installed
//   - ErrUpdateBranchCancelled if the operation was cancelled by the caller
//   - ErrUpdateBranchTimeout if the command times out
//   - ErrUpdateBranchFailed if the command exits with non-zero status
//   - nil on success
func UpdateBranch(ctx context.Context, owner, repo string, prNumber int) error {
	// Guard against nil context to prevent panic
	if ctx == nil {
		ctx = context.Background()
	}

	// Validate inputs early for clear error messages
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	if owner == "" {
		return &updateBranchError{
			sentinel: ErrInvalidArgument,
			output:   "owner cannot be empty",
		}
	}
	if repo == "" {
		return &updateBranchError{
			sentinel: ErrInvalidArgument,
			output:   "repo cannot be empty",
		}
	}
	if prNumber <= 0 {
		return &updateBranchError{
			sentinel: ErrInvalidArgument,
			output:   fmt.Sprintf("prNumber must be positive, got %d", prNumber),
		}
	}

	// First verify gh CLI is installed
	if _, err := exec.LookPath("gh"); err != nil {
		return &updateBranchError{
			sentinel: ErrGHCLINotFound,
			err:      err,
		}
	}

	// Apply default timeout only if the context has no deadline
	// This respects caller-provided longer deadlines
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, updateBranchTimeout)
		defer cancel()
	}

	// Build the repository identifier
	repoArg := fmt.Sprintf("%s/%s", owner, repo)

	// Execute: gh pr update-branch <number> --repo <owner/name>
	cmd := exec.CommandContext(ctx, "gh", "pr", "update-branch",
		fmt.Sprintf("%d", prNumber),
		"--repo", repoArg,
	)
	output, err := cmd.CombinedOutput()

	if err != nil {
		trimmedOutput := truncateOutput(strings.TrimSpace(string(output)))

		// Check if the error was due to context cancellation
		if ctx.Err() == context.Canceled {
			return &updateBranchError{
				sentinel: ErrUpdateBranchCancelled,
				output:   trimmedOutput,
				err:      ctx.Err(),
			}
		}

		// Check if the error was due to context timeout/deadline
		if ctx.Err() == context.DeadlineExceeded {
			return &updateBranchError{
				sentinel: ErrUpdateBranchTimeout,
				output:   trimmedOutput,
				err:      ctx.Err(), // Use ctx.Err() to preserve context.DeadlineExceeded in chain
			}
		}

		// Command failed with non-zero exit code
		return &updateBranchError{
			sentinel: ErrUpdateBranchFailed,
			output:   trimmedOutput,
			err:      err,
		}
	}

	return nil
}

// ErrOpenBrowserFailed is returned when the gh pr view --web command fails.
var ErrOpenBrowserFailed = errors.New("open browser command failed")

// OpenPRInBrowser opens a pull request in the default browser using gh CLI.
// This executes `gh pr view --web <number> --repo <owner/name>`.
//
// Parameters:
//   - owner: Repository owner (e.g., "octocat"). Must be non-empty.
//   - repo: Repository name (e.g., "hello-world"). Must be non-empty.
//   - prNumber: Pull request number. Must be greater than 0.
//
// Returns:
//   - ErrInvalidArgument if owner, repo, or prNumber are invalid
//   - ErrGHCLINotFound if gh is not installed
//   - ErrOpenBrowserFailed if the command exits with non-zero status
//   - nil on success
func OpenPRInBrowser(owner, repo string, prNumber int) error {
	// Validate inputs
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	if owner == "" {
		return &updateBranchError{
			sentinel: ErrInvalidArgument,
			output:   "owner cannot be empty",
		}
	}
	if repo == "" {
		return &updateBranchError{
			sentinel: ErrInvalidArgument,
			output:   "repo cannot be empty",
		}
	}
	if prNumber <= 0 {
		return &updateBranchError{
			sentinel: ErrInvalidArgument,
			output:   fmt.Sprintf("prNumber must be positive, got %d", prNumber),
		}
	}

	// Verify gh CLI is installed
	if _, err := exec.LookPath("gh"); err != nil {
		return &updateBranchError{
			sentinel: ErrGHCLINotFound,
			err:      err,
		}
	}

	// Build the repository identifier
	repoArg := fmt.Sprintf("%s/%s", owner, repo)

	// Execute: gh pr view --web <number> --repo <owner/name>
	cmd := exec.Command("gh", "pr", "view",
		fmt.Sprintf("%d", prNumber),
		"--web",
		"--repo", repoArg,
	)
	output, err := cmd.CombinedOutput()

	if err != nil {
		trimmedOutput := truncateOutput(strings.TrimSpace(string(output)))
		return &updateBranchError{
			sentinel: ErrOpenBrowserFailed,
			output:   trimmedOutput,
			err:      err,
		}
	}

	return nil
}
