package config

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
			name:     "gh auth status with colors",
			input:    "\x1b[32m✓\x1b[0m Logged in to github.com as \x1b[1muser\x1b[0m",
			expected: "✓ Logged in to github.com as user",
		},
		{
			name:     "gh auth error with colors",
			input:    "\x1b[31mX\x1b[0m Not logged in to github.com",
			expected: "X Not logged in to github.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripANSI(tt.input)
			if result != tt.expected {
				t.Errorf("stripANSI(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
