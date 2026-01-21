package github

import "regexp"

// ansiRegex matches ANSI escape sequences including:
// - CSI sequences: ESC [ ... final byte (colors, cursor control, etc.)
// - Private mode CSI: ESC [ ? ... (cursor show/hide, etc.)
// - OSC sequences: ESC ] ... ST (operating system commands)
// - Simple escapes: ESC followed by a single character
// This covers the vast majority of ANSI codes that gh CLI might output.
var ansiRegex = regexp.MustCompile(`\x1b\[\?[0-9;]*[a-zA-Z]|\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*\x07|\x1b.`)

// StripANSI removes ANSI escape sequences from a string.
// This is useful for cleaning gh CLI output before displaying in the TUI,
// as gh may include colored output that would appear as garbled text.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}
