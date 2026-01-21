package github

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// createFakeGH creates a fake gh executable in a temp directory that exits with the given code.
// Returns the temp directory path (to prepend to PATH) and a cleanup function.
func createFakeGH(t *testing.T, exitCode int) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "fake-gh-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	var ghPath string
	var script string

	if runtime.GOOS == "windows" {
		ghPath = filepath.Join(tmpDir, "gh.bat")
		script = "@echo off\nexit /b " + strconv.Itoa(exitCode)
	} else {
		ghPath = filepath.Join(tmpDir, "gh")
		script = "#!/bin/sh\nexit " + strconv.Itoa(exitCode)
	}

	if err := os.WriteFile(ghPath, []byte(script), 0755); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to write fake gh: %v", err)
	}

	return tmpDir, func() { os.RemoveAll(tmpDir) }
}

// createFakeGHWithOutput creates a fake gh executable that outputs a message and exits with the given code.
// Output is written to a temporary file and cat'd to avoid shell quoting issues.
func createFakeGHWithOutput(t *testing.T, exitCode int, output string) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "fake-gh-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	var ghPath string
	var script string

	if runtime.GOOS == "windows" {
		// Write output to a file and type it to avoid quoting issues
		outputFile := filepath.Join(tmpDir, "output.txt")
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to write output file: %v", err)
		}
		ghPath = filepath.Join(tmpDir, "gh.bat")
		script = "@echo off\ntype \"" + outputFile + "\"\nexit /b " + strconv.Itoa(exitCode)
	} else {
		// Write output to a file and cat it to avoid shell quoting issues
		outputFile := filepath.Join(tmpDir, "output.txt")
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to write output file: %v", err)
		}
		ghPath = filepath.Join(tmpDir, "gh")
		script = "#!/bin/sh\ncat \"" + outputFile + "\"\nexit " + strconv.Itoa(exitCode)
	}

	if err := os.WriteFile(ghPath, []byte(script), 0755); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to write fake gh: %v", err)
	}

	return tmpDir, func() { os.RemoveAll(tmpDir) }
}

// createSlowFakeGH creates a fake gh executable that sleeps before exiting.
// Uses exec to replace the shell process so that signals are properly received.
func createSlowFakeGH(t *testing.T, sleepDuration time.Duration) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "fake-gh-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	var ghPath string
	var script string

	sleepSeconds := int(sleepDuration.Seconds())
	if sleepSeconds < 1 {
		sleepSeconds = 1
	}

	if runtime.GOOS == "windows" {
		ghPath = filepath.Join(tmpDir, "gh.bat")
		// Windows: use ping for delay (timeout command doesn't work in batch without /T)
		script = "@echo off\nping -n " + strconv.Itoa(sleepSeconds+1) + " 127.0.0.1 > nul\nexit /b 0"
	} else {
		ghPath = filepath.Join(tmpDir, "gh")
		// Use exec to replace the shell process with sleep, so that signals
		// (like SIGKILL from context cancellation) are properly received by sleep.
		// Without exec, the shell receives the signal but sleep continues running.
		// Use `sleep` via PATH (not hardcoded /bin/sleep) for portability across systems.
		script = "#!/bin/sh\nexec sleep " + strconv.Itoa(sleepSeconds)
	}

	if err := os.WriteFile(ghPath, []byte(script), 0755); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to write fake gh: %v", err)
	}

	return tmpDir, func() { os.RemoveAll(tmpDir) }
}

// withPATH temporarily prepends the given path to PATH for the duration of the test.
// This ensures that system utilities (cat, sleep, etc.) remain available while
// prioritizing our fake gh binary.
func withPATH(t *testing.T, prependPath string) func() {
	t.Helper()
	oldPath := os.Getenv("PATH")
	newPath := prependPath + string(os.PathListSeparator) + oldPath
	if err := os.Setenv("PATH", newPath); err != nil {
		t.Fatalf("failed to set PATH: %v", err)
	}
	return func() {
		os.Setenv("PATH", oldPath)
	}
}

// withEmptyPATH temporarily sets PATH to only the given path (no system utilities).
// Use this for tests that specifically need to test "gh not found" scenarios.
func withEmptyPATH(t *testing.T, newPath string) func() {
	t.Helper()
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", newPath); err != nil {
		t.Fatalf("failed to set PATH: %v", err)
	}
	return func() {
		os.Setenv("PATH", oldPath)
	}
}

// createFakeGHWithArgvCapture creates a fake gh executable that captures its arguments
// to a file and exits with the given code. Returns the temp directory path and the
// path to the argv capture file.
func createFakeGHWithArgvCapture(t *testing.T, exitCode int) (string, string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "fake-gh-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	argvFile := filepath.Join(tmpDir, "argv.txt")
	var ghPath string
	var script string

	if runtime.GOOS == "windows" {
		ghPath = filepath.Join(tmpDir, "gh.bat")
		// Windows: echo all args to file
		script = "@echo off\necho %* > \"" + argvFile + "\"\nexit /b " + strconv.Itoa(exitCode)
	} else {
		ghPath = filepath.Join(tmpDir, "gh")
		// Unix: echo all args to file (use $@ for all args)
		script = "#!/bin/sh\necho \"$@\" > \"" + argvFile + "\"\nexit " + strconv.Itoa(exitCode)
	}

	if err := os.WriteFile(ghPath, []byte(script), 0755); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to write fake gh: %v", err)
	}

	return tmpDir, argvFile, func() { os.RemoveAll(tmpDir) }
}

func TestUpdateBranch_Actions_Success(t *testing.T) {
	// Create a fake gh binary that exits 0 (success)
	tmpDir, cleanup := createFakeGH(t, 0)
	defer cleanup()

	// Set PATH to only our temp dir
	restorePath := withPATH(t, tmpDir)
	defer restorePath()

	err := UpdateBranch(context.Background(), "octocat", "hello-world", 123)
	if err != nil {
		t.Errorf("expected no error when gh succeeds, got: %v", err)
	}
}

func TestUpdateBranch_Actions_InvokesCorrectCommand(t *testing.T) {
	// Create a fake gh binary that captures its arguments
	tmpDir, argvFile, cleanup := createFakeGHWithArgvCapture(t, 0)
	defer cleanup()

	// Set PATH to only our temp dir
	restorePath := withPATH(t, tmpDir)
	defer restorePath()

	err := UpdateBranch(context.Background(), "octocat", "hello-world", 456)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read captured arguments
	argvBytes, err := os.ReadFile(argvFile)
	if err != nil {
		t.Fatalf("failed to read argv file: %v", err)
	}
	argv := strings.TrimSpace(string(argvBytes))

	// Verify the correct subcommand and arguments were passed
	// Expected: "pr update-branch 456 --repo octocat/hello-world"
	expectedParts := []string{"pr", "update-branch", "456", "--repo", "octocat/hello-world"}
	for _, part := range expectedParts {
		if !strings.Contains(argv, part) {
			t.Errorf("argv should contain %q, got: %q", part, argv)
		}
	}
}

func TestUpdateBranch_Actions_Failure(t *testing.T) {
	// Create a fake gh binary that exits 1 (failure)
	tmpDir, cleanup := createFakeGHWithOutput(t, 1, "PR is not behind the base branch")
	defer cleanup()

	// Set PATH to only our temp dir
	restorePath := withPATH(t, tmpDir)
	defer restorePath()

	err := UpdateBranch(context.Background(), "octocat", "hello-world", 123)
	if err == nil {
		t.Error("expected error when gh fails, got nil")
	}
	if !errors.Is(err, ErrUpdateBranchFailed) {
		t.Errorf("expected ErrUpdateBranchFailed, got: %v", err)
	}

	// Verify error message contains the output
	if err != nil && !strings.Contains(err.Error(), "PR is not behind the base branch") {
		t.Errorf("error message should contain gh output, got: %v", err)
	}
}

func TestUpdateBranch_Actions_Failure_PreservesExitError(t *testing.T) {
	// Create a fake gh binary that exits 1 (failure)
	tmpDir, cleanup := createFakeGH(t, 1)
	defer cleanup()

	// Set PATH to only our temp dir
	restorePath := withPATH(t, tmpDir)
	defer restorePath()

	err := UpdateBranch(context.Background(), "octocat", "hello-world", 123)
	if err == nil {
		t.Error("expected error when gh fails, got nil")
	}

	// errors.Is should work for the sentinel
	if !errors.Is(err, ErrUpdateBranchFailed) {
		t.Errorf("errors.Is should match ErrUpdateBranchFailed, got: %v", err)
	}

	// The underlying exec.ExitError should be accessible via errors.As
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Error("underlying exec.ExitError should be accessible via errors.As")
	}
}

func TestUpdateBranch_Actions_GHNotInstalled(t *testing.T) {
	// Set PATH to empty directory so gh is not found
	tmpDir, err := os.MkdirTemp("", "empty-path-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use withEmptyPATH to exclude system utilities (we need gh to truly be missing)
	restorePath := withEmptyPATH(t, tmpDir)
	defer restorePath()

	err = UpdateBranch(context.Background(), "octocat", "hello-world", 123)
	if err == nil {
		t.Error("expected error when gh is not installed, got nil")
	}
	if !errors.Is(err, ErrGHCLINotFound) {
		t.Errorf("expected ErrGHCLINotFound, got: %v", err)
	}
}

func TestUpdateBranch_Actions_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test in short mode")
	}

	// Create a fake gh binary that sleeps for longer than the timeout
	tmpDir, cleanup := createSlowFakeGH(t, 10*time.Second)
	defer cleanup()

	// Set PATH to only our temp dir
	restorePath := withPATH(t, tmpDir)
	defer restorePath()

	// Create a context with a very short timeout (shorter than the sleep duration)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := UpdateBranch(ctx, "octocat", "hello-world", 123)
	elapsed := time.Since(start)

	// Should complete in roughly 500ms due to timeout, not wait for the full 10s sleep
	if elapsed > 3*time.Second {
		t.Errorf("command should have timed out quickly, took %v", elapsed)
	}

	if err == nil {
		t.Error("expected error when command times out, got nil")
	}

	// The parent context timing out will be captured as DeadlineExceeded
	// and wrapped in ErrUpdateBranchTimeout
	if !errors.Is(err, ErrUpdateBranchTimeout) {
		t.Errorf("expected ErrUpdateBranchTimeout, got: %v", err)
	}
}

func TestUpdateBranch_Actions_RespectsCallerDeadline(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping deadline test in short mode")
	}

	// Create a fake gh binary that sleeps for 10 seconds (longer than our caller deadline)
	tmpDir, cleanup := createSlowFakeGH(t, 10*time.Second)
	defer cleanup()

	// Set PATH to only our temp dir
	restorePath := withPATH(t, tmpDir)
	defer restorePath()

	// Create a context with a 1-second deadline (much shorter than default 30s)
	// The function should respect this caller-provided deadline and timeout at ~1s
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	start := time.Now()
	err := UpdateBranch(ctx, "octocat", "hello-world", 123)
	elapsed := time.Since(start)

	// Should timeout at ~1 second (caller deadline), not wait for 10s sleep
	if elapsed > 3*time.Second {
		t.Errorf("command should have timed out at caller deadline, took %v (expected ~1s)", elapsed)
	}

	// Should return timeout error
	if err == nil {
		t.Error("expected timeout error, got nil")
	} else if !errors.Is(err, ErrUpdateBranchTimeout) {
		t.Errorf("expected ErrUpdateBranchTimeout, got: %v", err)
	}
}

func TestUpdateBranch_Actions_Timeout_PreservesDeadlineExceeded(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test in short mode")
	}

	// Create a fake gh binary that sleeps
	tmpDir, cleanup := createSlowFakeGH(t, 10*time.Second)
	defer cleanup()

	restorePath := withPATH(t, tmpDir)
	defer restorePath()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := UpdateBranch(ctx, "octocat", "hello-world", 123)
	if err == nil {
		t.Error("expected timeout error, got nil")
		return
	}

	// The underlying context.DeadlineExceeded should be accessible
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Error("underlying context.DeadlineExceeded should be accessible via errors.Is")
	}
}

func TestSentinelErrors_Actions_CanUseErrorsIs(t *testing.T) {
	// Verify that errors.Is works correctly with our sentinel errors
	if !errors.Is(ErrGHCLINotFound, ErrGHCLINotFound) {
		t.Error("errors.Is should return true for ErrGHCLINotFound == ErrGHCLINotFound")
	}
	if !errors.Is(ErrUpdateBranchTimeout, ErrUpdateBranchTimeout) {
		t.Error("errors.Is should return true for ErrUpdateBranchTimeout == ErrUpdateBranchTimeout")
	}
	if !errors.Is(ErrUpdateBranchFailed, ErrUpdateBranchFailed) {
		t.Error("errors.Is should return true for ErrUpdateBranchFailed == ErrUpdateBranchFailed")
	}
	if !errors.Is(ErrUpdateBranchCancelled, ErrUpdateBranchCancelled) {
		t.Error("errors.Is should return true for ErrUpdateBranchCancelled == ErrUpdateBranchCancelled")
	}
	if !errors.Is(ErrInvalidArgument, ErrInvalidArgument) {
		t.Error("errors.Is should return true for ErrInvalidArgument == ErrInvalidArgument")
	}

	// They should not be equal to each other
	if errors.Is(ErrGHCLINotFound, ErrUpdateBranchTimeout) {
		t.Error("ErrGHCLINotFound should not match ErrUpdateBranchTimeout")
	}
	if errors.Is(ErrUpdateBranchFailed, ErrGHCLINotFound) {
		t.Error("ErrUpdateBranchFailed should not match ErrGHCLINotFound")
	}
}

func TestUpdateBranch_Actions_InvalidArguments(t *testing.T) {
	// Create a fake gh binary (we shouldn't reach it due to early validation)
	tmpDir, cleanup := createFakeGH(t, 0)
	defer cleanup()
	restorePath := withPATH(t, tmpDir)
	defer restorePath()

	tests := []struct {
		name     string
		owner    string
		repo     string
		prNumber int
		wantMsg  string
	}{
		{"empty owner", "", "repo", 1, "owner cannot be empty"},
		{"whitespace owner", "  ", "repo", 1, "owner cannot be empty"},
		{"empty repo", "owner", "", 1, "repo cannot be empty"},
		{"whitespace repo", "owner", "  ", 1, "repo cannot be empty"},
		{"zero prNumber", "owner", "repo", 0, "prNumber must be positive"},
		{"negative prNumber", "owner", "repo", -1, "prNumber must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UpdateBranch(context.Background(), tt.owner, tt.repo, tt.prNumber)
			if err == nil {
				t.Error("expected error for invalid argument, got nil")
				return
			}
			if !errors.Is(err, ErrInvalidArgument) {
				t.Errorf("expected ErrInvalidArgument, got: %v", err)
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("error should contain %q, got: %v", tt.wantMsg, err)
			}
		})
	}
}

func TestUpdateBranch_Actions_ContextCancelled_ReturnsCancelled(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping context cancellation test in short mode")
	}

	// Create a fake gh binary that sleeps
	tmpDir, cleanup := createSlowFakeGH(t, 10*time.Second)
	defer cleanup()

	// Set PATH to only our temp dir
	restorePath := withPATH(t, tmpDir)
	defer restorePath()

	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	err := UpdateBranch(ctx, "octocat", "hello-world", 123)
	if err == nil {
		t.Error("expected error when context is cancelled, got nil")
		return
	}

	// Should return ErrUpdateBranchCancelled, not ErrUpdateBranchFailed
	if !errors.Is(err, ErrUpdateBranchCancelled) {
		t.Errorf("expected ErrUpdateBranchCancelled, got: %v", err)
	}

	// The underlying context.Canceled should be accessible
	if !errors.Is(err, context.Canceled) {
		t.Error("underlying context.Canceled should be accessible via errors.Is")
	}
}
