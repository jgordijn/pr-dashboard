package main

import (
	"testing"
)

func TestPrintUsage(t *testing.T) {
	// Just verify it doesn't panic
	printUsage()
}

func TestVersionVariable(t *testing.T) {
	// Verify Version is set to a default value
	if Version == "" {
		t.Error("Version should not be empty")
	}
}
