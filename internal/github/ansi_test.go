package github

import "testing"

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no ANSI codes",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "red color code",
			input:    "\x1b[31mError\x1b[0m",
			expected: "Error",
		},
		{
			name:     "green color code",
			input:    "\x1b[32mSuccess\x1b[0m",
			expected: "Success",
		},
		{
			name:     "bold text",
			input:    "\x1b[1mBold\x1b[0m",
			expected: "Bold",
		},
		{
			name:     "multiple color codes",
			input:    "\x1b[31mRed\x1b[0m and \x1b[32mGreen\x1b[0m",
			expected: "Red and Green",
		},
		{
			name:     "256 color code",
			input:    "\x1b[38;5;196mBright Red\x1b[0m",
			expected: "Bright Red",
		},
		{
			name:     "24-bit RGB color",
			input:    "\x1b[38;2;255;0;0mTrue Red\x1b[0m",
			expected: "True Red",
		},
		{
			name:     "cursor movement",
			input:    "\x1b[2JCleared\x1b[H",
			expected: "Cleared",
		},
		{
			name:     "complex gh CLI error output",
			input:    "\x1b[31mX\x1b[0m \x1b[1mPull request #123\x1b[0m is not behind the base branch",
			expected: "X Pull request #123 is not behind the base branch",
		},
		{
			name:     "gh auth status output with colors",
			input:    "\x1b[32m✓\x1b[0m Logged in to github.com as \x1b[1muser\x1b[0m",
			expected: "✓ Logged in to github.com as user",
		},
		{
			name:     "escape with single char",
			input:    "text\x1bMmore",
			expected: "textmore",
		},
		{
			name:     "preserves unicode",
			input:    "\x1b[32m✓\x1b[0m Success: 日本語 and émojis 🎉",
			expected: "✓ Success: 日本語 and émojis 🎉",
		},
		{
			name:     "OSC sequence (terminal title)",
			input:    "\x1b]0;Terminal Title\x07Content",
			expected: "Content",
		},
		{
			name:     "nested/sequential escapes",
			input:    "\x1b[1m\x1b[31mBold Red\x1b[0m\x1b[0m",
			expected: "Bold Red",
		},
		{
			name:     "real gh error with spinner cleared",
			input:    "\x1b[?25l\x1b[?25h\x1b[31mError: PR has conflicts\x1b[0m",
			expected: "Error: PR has conflicts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripANSI(tt.input)
			if result != tt.expected {
				t.Errorf("StripANSI(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStripANSI_Idempotent(t *testing.T) {
	// Stripping twice should give the same result
	input := "\x1b[31mError\x1b[0m: something went wrong"
	first := StripANSI(input)
	second := StripANSI(first)

	if first != second {
		t.Errorf("StripANSI is not idempotent: first=%q, second=%q", first, second)
	}
}
