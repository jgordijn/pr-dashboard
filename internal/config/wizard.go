package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ErrWizardCancelled is returned when the user cancels the wizard (e.g., EOF/Ctrl+D).
var ErrWizardCancelled = errors.New("wizard cancelled by user")

// ErrConfigExists is returned when attempting to write a config file that already exists.
var ErrConfigExists = errors.New("configuration file already exists")

// Wizard handles the setup configuration wizard.
// It prompts the user for configuration values and creates the config file.
type Wizard struct {
	in  io.Reader
	out io.Writer
}

// NewWizard creates a new Wizard instance using stdin and stdout.
func NewWizard() *Wizard {
	return &Wizard{
		in:  os.Stdin,
		out: os.Stdout,
	}
}

// NewWizardWithIO creates a new Wizard instance with custom input/output streams.
// This is primarily useful for testing.
func NewWizardWithIO(in io.Reader, out io.Writer) *Wizard {
	return &Wizard{
		in:  in,
		out: out,
	}
}

// Run executes the setup configuration wizard.
// It checks for gh CLI prerequisites, prompts the user for configuration values,
// creates the config directory and file, and returns the created Config.
// Returns ErrWizardCancelled if the user cancels the wizard (EOF/Ctrl+D).
// Returns ErrConfigExists if the config file already exists.
// Returns ErrGHCLINotFound or ErrGHNotAuthenticated if gh CLI checks fail.
func (w *Wizard) Run() (*Config, error) {
	w.printWelcome()

	// Check gh CLI prerequisites
	if err := w.checkPrerequisites(); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(w.in)
	// Increase buffer size to handle long lines (up to 1MB)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	// Prompt for username
	username, err := w.promptUsername(scanner)
	if err != nil {
		return nil, err
	}

	// Prompt for organizations
	orgs, err := w.promptOrganizations(scanner)
	if err != nil {
		return nil, err
	}

	// Prompt for refresh interval
	refreshInterval, err := w.promptRefreshInterval(scanner)
	if err != nil {
		return nil, err
	}

	// Build the config
	cfg := w.buildConfig(username, orgs, refreshInterval)

	// Write config to file
	if err := w.writeConfig(cfg); err != nil {
		return nil, err
	}

	w.printSuccess()

	return cfg, nil
}

// printWelcome prints the welcome message.
func (w *Wizard) printWelcome() {
	fmt.Fprintln(w.out, "")
	fmt.Fprintln(w.out, "Welcome to pr-dashboard!")
	fmt.Fprintln(w.out, "========================")
	fmt.Fprintln(w.out, "")
	fmt.Fprintln(w.out, "This wizard will help you set up your configuration.")
	fmt.Fprintln(w.out, "Press Ctrl+D (EOF) at any time to cancel.")
	fmt.Fprintln(w.out, "")
}

// checkPrerequisites verifies that gh CLI is installed and authenticated.
func (w *Wizard) checkPrerequisites() error {
	fmt.Fprint(w.out, "Checking prerequisites...")

	if err := CheckGHCLI(); err != nil {
		fmt.Fprintln(w.out, " FAILED")
		fmt.Fprintln(w.out, "")
		fmt.Fprintln(w.out, "The GitHub CLI (gh) is not installed.")
		fmt.Fprintln(w.out, "Please install it from: https://cli.github.com")
		return err
	}

	if err := CheckGHAuth(); err != nil {
		fmt.Fprintln(w.out, " FAILED")
		fmt.Fprintln(w.out, "")
		fmt.Fprintln(w.out, "The GitHub CLI is not authenticated.")
		fmt.Fprintln(w.out, "Please run: gh auth login")
		return err
	}

	fmt.Fprintln(w.out, " OK")
	fmt.Fprintln(w.out, "")
	return nil
}

// promptUsername prompts for and validates the GitHub username.
func (w *Wizard) promptUsername(scanner *bufio.Scanner) (string, error) {
	for {
		fmt.Fprint(w.out, "GitHub username: ")
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return "", fmt.Errorf("error reading input: %w", err)
			}
			return "", ErrWizardCancelled
		}

		username := strings.TrimSpace(scanner.Text())
		if username == "" {
			fmt.Fprintln(w.out, "  Error: Username is required. Please try again.")
			continue
		}

		return username, nil
	}
}

// promptOrganizations prompts for and validates the GitHub organizations.
func (w *Wizard) promptOrganizations(scanner *bufio.Scanner) ([]string, error) {
	fmt.Fprintln(w.out, "")
	fmt.Fprintln(w.out, "Enter the GitHub organizations to monitor.")
	fmt.Fprintln(w.out, "Separate multiple organizations with commas.")

	for {
		fmt.Fprint(w.out, "Organizations: ")
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return nil, fmt.Errorf("error reading input: %w", err)
			}
			return nil, ErrWizardCancelled
		}

		input := scanner.Text()
		orgs := ParseOrganizations(input)

		if len(orgs) == 0 {
			fmt.Fprintln(w.out, "  Error: At least one organization is required. Please try again.")
			continue
		}

		return orgs, nil
	}
}

// promptRefreshInterval prompts for the refresh interval with a default value.
// It loops until valid input (or empty for default) is received.
func (w *Wizard) promptRefreshInterval(scanner *bufio.Scanner) (int, error) {
	fmt.Fprintln(w.out, "")

	for {
		fmt.Fprintf(w.out, "Refresh interval in seconds (%d-%d) [%d]: ",
			MinRefreshInterval, MaxRefreshInterval, DefaultRefreshInterval)

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return 0, fmt.Errorf("error reading input: %w", err)
			}
			return 0, ErrWizardCancelled
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			return DefaultRefreshInterval, nil
		}

		interval, err := ParseRefreshInterval(input)
		if err != nil {
			fmt.Fprintf(w.out, "  Error: %v. Please try again.\n", err)
			continue
		}

		return interval, nil
	}
}

// buildConfig creates a Config struct from the collected values.
func (w *Wizard) buildConfig(username string, orgs []string, refreshInterval int) *Config {
	cfg := &Config{
		General: GeneralConfig{
			Username:        username,
			RefreshInterval: refreshInterval,
		},
		Organizations: make([]OrganizationConfig, len(orgs)),
		Display: DisplayConfig{
			ShowDrafts:  DefaultShowDrafts,
			InitialMode: DefaultInitialMode,
		},
		Notifications: NotificationsConfig{
			HighlightChanges: DefaultHighlightChanges,
		},
	}

	for i, org := range orgs {
		cfg.Organizations[i] = OrganizationConfig{Login: org}
	}

	return cfg
}

// writeConfig writes the configuration to the default config file.
// It refuses to overwrite an existing file and uses atomic write (temp file + rename).
func (w *Wizard) writeConfig(cfg *Config) error {
	configPath, err := DefaultConfigPath()
	if err != nil {
		return fmt.Errorf("failed to determine config path: %w", err)
	}

	if err := WriteConfigToPath(configPath, cfg); err != nil {
		return err
	}

	fmt.Fprintf(w.out, "\nConfiguration saved to: %s\n", configPath)
	return nil
}

// WriteConfigToPath writes a Config to the specified path.
// It refuses to overwrite an existing file and uses atomic write (temp file + rename).
// The config directory is created with 0700 permissions if it doesn't exist.
// The config file is created with 0600 permissions.
// This function is exported for testability.
func WriteConfigToPath(configPath string, cfg *Config) error {
	// Check if config file already exists - refuse to overwrite
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("%w: %s", ErrConfigExists, configPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check config file %s: %w", configPath, err)
	}

	// Create config directory if it doesn't exist (restrictive permissions)
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	// Generate TOML content
	content := GenerateConfigTOML(cfg)

	// Write to a temporary file first, then rename for atomic operation
	tmpFile, err := os.CreateTemp(configDir, ".pr-dashboard-config-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary config file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on error
	defer func() {
		if tmpPath != "" {
			os.Remove(tmpPath)
		}
	}()

	// Write content to temp file
	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temporary config file: %w", err)
	}

	// Set restrictive permissions before closing
	if err := tmpFile.Chmod(0600); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to set config file permissions: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary config file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, configPath); err != nil {
		return fmt.Errorf("failed to save config file %s: %w", configPath, err)
	}

	// Clear tmpPath so defer doesn't try to remove the final file
	tmpPath = ""

	return nil
}

// printSuccess prints the success message.
func (w *Wizard) printSuccess() {
	fmt.Fprintln(w.out, "")
	fmt.Fprintln(w.out, "Setup complete! You can now run pr-dashboard.")
	fmt.Fprintln(w.out, "")
}

// ParseOrganizations parses a comma-separated list of organization names.
// It trims whitespace from each organization and filters out empty strings.
func ParseOrganizations(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}

	parts := strings.Split(input, ",")
	var orgs []string

	for _, part := range parts {
		org := strings.TrimSpace(part)
		if org != "" {
			orgs = append(orgs, org)
		}
	}

	return orgs
}

// ParseRefreshInterval parses and validates a refresh interval string.
// Returns an error if the value is not a valid integer or is out of range.
func ParseRefreshInterval(input string) (int, error) {
	input = strings.TrimSpace(input)

	val, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("invalid refresh interval %q: must be a number", input)
	}

	if val < MinRefreshInterval {
		return 0, fmt.Errorf("refresh interval %d is too low: minimum is %d", val, MinRefreshInterval)
	}

	if val > MaxRefreshInterval {
		return 0, fmt.Errorf("refresh interval %d is too high: maximum is %d", val, MaxRefreshInterval)
	}

	return val, nil
}

// GenerateConfigTOML generates the TOML content for a Config struct.
func GenerateConfigTOML(cfg *Config) string {
	var sb strings.Builder

	sb.WriteString("# pr-dashboard configuration\n")
	sb.WriteString("# Generated by the setup wizard\n\n")

	sb.WriteString("[general]\n")
	sb.WriteString(fmt.Sprintf("username = %q\n", cfg.General.Username))
	sb.WriteString(fmt.Sprintf("refresh_interval = %d\n", cfg.General.RefreshInterval))
	sb.WriteString("\n")

	for _, org := range cfg.Organizations {
		sb.WriteString("[[organizations]]\n")
		sb.WriteString(fmt.Sprintf("login = %q\n", org.Login))
		sb.WriteString("\n")
	}

	sb.WriteString("[display]\n")
	sb.WriteString(fmt.Sprintf("show_drafts = %t\n", cfg.Display.ShowDrafts))
	sb.WriteString(fmt.Sprintf("initial_mode = %q\n", cfg.Display.InitialMode))
	sb.WriteString("\n")

	sb.WriteString("[notifications]\n")
	sb.WriteString(fmt.Sprintf("highlight_changes = %t\n", cfg.Notifications.HighlightChanges))
	sb.WriteString("\n")

	return sb.String()
}

// RunWizard is a convenience function that creates a new Wizard and runs it.
// This is the primary entry point for the setup wizard.
func RunWizard() (*Config, error) {
	return NewWizard().Run()
}
