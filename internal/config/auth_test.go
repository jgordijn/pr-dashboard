package config

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
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

// withPATH temporarily sets PATH to the given value for the duration of the test.
func withPATH(t *testing.T, newPath string) func() {
	t.Helper()
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", newPath); err != nil {
		t.Fatalf("failed to set PATH: %v", err)
	}
	return func() {
		os.Setenv("PATH", oldPath)
	}
}

func TestCheckGHCLI_Installed(t *testing.T) {
	// Create a fake gh binary that exits successfully
	tmpDir, cleanup := createFakeGH(t, 0)
	defer cleanup()

	// Prepend our temp dir to PATH
	oldPath := os.Getenv("PATH")
	restorePath := withPATH(t, tmpDir+string(os.PathListSeparator)+oldPath)
	defer restorePath()

	err := CheckGHCLI()
	if err != nil {
		t.Errorf("expected no error when gh is in PATH, got: %v", err)
	}
}

func TestCheckGHCLI_NotInstalled(t *testing.T) {
	// Set PATH to empty/nonexistent directory so gh is not found
	tmpDir, err := os.MkdirTemp("", "empty-path-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	restorePath := withPATH(t, tmpDir)
	defer restorePath()

	err = CheckGHCLI()
	if err == nil {
		t.Error("expected error when gh is not in PATH, got nil")
	}
	if !errors.Is(err, ErrGHCLINotFound) {
		t.Errorf("expected ErrGHCLINotFound, got: %v", err)
	}
}

func TestCheckGHCLI_ErrorType(t *testing.T) {
	// Verify the error type is correct when gh is not found
	if ErrGHCLINotFound == nil {
		t.Error("ErrGHCLINotFound should not be nil")
	}

	expectedMsg := "gh CLI not found. Install from https://cli.github.com"
	if ErrGHCLINotFound.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, ErrGHCLINotFound.Error())
	}
}

func TestCheckGHAuth_Authenticated(t *testing.T) {
	// Create a fake gh binary that exits 0 (authenticated)
	tmpDir, cleanup := createFakeGH(t, 0)
	defer cleanup()

	// Set PATH to only our temp dir
	restorePath := withPATH(t, tmpDir)
	defer restorePath()

	err := CheckGHAuth()
	if err != nil {
		t.Errorf("expected no error when gh auth succeeds, got: %v", err)
	}
}

func TestCheckGHAuth_NotAuthenticated(t *testing.T) {
	// Create a fake gh binary that exits 1 (not authenticated)
	tmpDir, cleanup := createFakeGH(t, 1)
	defer cleanup()

	// Set PATH to only our temp dir
	restorePath := withPATH(t, tmpDir)
	defer restorePath()

	err := CheckGHAuth()
	if err == nil {
		t.Error("expected error when gh auth fails, got nil")
	}
	if !errors.Is(err, ErrGHNotAuthenticated) {
		t.Errorf("expected ErrGHNotAuthenticated, got: %v", err)
	}
}

func TestCheckGHAuth_NotAuthenticated_PreservesUnderlyingError(t *testing.T) {
	// Create a fake gh binary that exits 1 (not authenticated)
	tmpDir, cleanup := createFakeGH(t, 1)
	defer cleanup()

	// Set PATH to only our temp dir
	restorePath := withPATH(t, tmpDir)
	defer restorePath()

	err := CheckGHAuth()
	if err == nil {
		t.Error("expected error when gh auth fails, got nil")
	}

	// errors.Is should still work for the sentinel
	if !errors.Is(err, ErrGHNotAuthenticated) {
		t.Errorf("errors.Is should match ErrGHNotAuthenticated, got: %v", err)
	}

	// The error message should contain more than just the sentinel
	if err.Error() == ErrGHNotAuthenticated.Error() {
		t.Error("wrapped error should contain additional context from underlying error")
	}

	// The underlying exec.ExitError should be accessible via errors.As
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Error("underlying exec.ExitError should be accessible via errors.As")
	}
}

func TestCheckGHAuth_GHNotInstalled(t *testing.T) {
	// Set PATH to empty directory so gh is not found
	tmpDir, err := os.MkdirTemp("", "empty-path-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	restorePath := withPATH(t, tmpDir)
	defer restorePath()

	err = CheckGHAuth()
	if err == nil {
		t.Error("expected error when gh is not installed, got nil")
	}
	if !errors.Is(err, ErrGHCLINotFound) {
		t.Errorf("expected ErrGHCLINotFound, got: %v", err)
	}
}

func TestCheckGHAuth_ErrorType(t *testing.T) {
	// Verify the error type is correct
	if ErrGHNotAuthenticated == nil {
		t.Error("ErrGHNotAuthenticated should not be nil")
	}

	expectedMsg := "gh CLI not authenticated. Run `gh auth login` first"
	if ErrGHNotAuthenticated.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, ErrGHNotAuthenticated.Error())
	}
}

func TestSentinelErrors_CanUseErrorsIs(t *testing.T) {
	// Verify that errors.Is works correctly with our sentinel errors
	if !errors.Is(ErrGHCLINotFound, ErrGHCLINotFound) {
		t.Error("errors.Is should return true for ErrGHCLINotFound == ErrGHCLINotFound")
	}
	if !errors.Is(ErrGHNotAuthenticated, ErrGHNotAuthenticated) {
		t.Error("errors.Is should return true for ErrGHNotAuthenticated == ErrGHNotAuthenticated")
	}

	// They should not be equal to each other
	if errors.Is(ErrGHCLINotFound, ErrGHNotAuthenticated) {
		t.Error("ErrGHCLINotFound should not match ErrGHNotAuthenticated")
	}
	if errors.Is(ErrGHNotAuthenticated, ErrGHCLINotFound) {
		t.Error("ErrGHNotAuthenticated should not match ErrGHCLINotFound")
	}
}
