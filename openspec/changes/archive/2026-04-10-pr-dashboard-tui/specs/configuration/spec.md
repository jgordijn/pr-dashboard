## ADDED Requirements

### Requirement: Configuration File
The system SHALL load configuration from `~/.config/pr-dashboard/config.toml`.

#### Scenario: Load existing configuration
- **WHEN** the config file exists at `~/.config/pr-dashboard/config.toml`
- **THEN** the system SHALL parse and apply all configuration values

#### Scenario: Apply default values
- **WHEN** optional configuration values are not specified
- **THEN** the system SHALL apply sensible defaults:
  - refresh_interval: 30 seconds
  - show_drafts: true
  - initial_mode: "full"
  - highlight_changes: true

### Requirement: Configuration Validation
The system SHALL validate configuration values and report clear errors.

#### Scenario: Missing required username
- **WHEN** the config file does not contain a username
- **THEN** the system SHALL exit with error "username is required in config"

#### Scenario: Missing organizations
- **WHEN** the config file contains no [[organizations]] blocks
- **THEN** the system SHALL exit with error "at least one organization is required"

#### Scenario: Invalid refresh interval
- **WHEN** refresh_interval is less than 10 or greater than 300
- **THEN** the system SHALL exit with error "refresh_interval must be between 10 and 300 seconds"

#### Scenario: Invalid display mode
- **WHEN** initial_mode is not "full", "compact", or "minimal"
- **THEN** the system SHALL exit with error "initial_mode must be 'full', 'compact', or 'minimal'"

### Requirement: First-Run Setup Wizard
The system SHALL provide an interactive setup wizard when no configuration exists.

#### Scenario: Detect missing configuration
- **WHEN** the application starts without a config file
- **THEN** the system SHALL launch the interactive setup wizard

#### Scenario: Collect configuration interactively
- **WHEN** the setup wizard is running
- **THEN** the system SHALL prompt for:
  - GitHub username (required)
  - Organization(s) to watch (comma-separated, at least one required)
  - Refresh interval (with default suggestion of 30)
- **AND** generate a valid config file at `~/.config/pr-dashboard/config.toml`

### Requirement: GitHub CLI Detection
The system SHALL verify that the gh CLI is installed.

#### Scenario: gh CLI installed
- **WHEN** `gh` is found in system PATH
- **THEN** the system SHALL proceed to authentication check

#### Scenario: gh CLI not installed
- **WHEN** `gh` is not found in system PATH
- **THEN** the system SHALL display error "gh CLI not found. Install from https://cli.github.com"
- **AND** exit gracefully

### Requirement: GitHub Authentication Check
The system SHALL verify GitHub CLI authentication before operation.

#### Scenario: Authentication present
- **WHEN** `gh auth status` indicates successful authentication
- **THEN** the system SHALL proceed with normal operation

#### Scenario: Authentication missing
- **WHEN** `gh auth status` indicates no authentication
- **THEN** the system SHALL display an error message
- **AND** instruct the user to run `gh auth login`
- **AND** exit gracefully

### Requirement: Multi-Organization Support
The system SHALL support watching multiple GitHub organizations.

#### Scenario: Configure multiple organizations
- **WHEN** the config file contains multiple `[[organizations]]` blocks
- **THEN** the system SHALL fetch PRs from all configured organizations
- **AND** display them grouped by organization

### Requirement: Terminal Requirements
The system SHALL verify terminal requirements before starting.

#### Scenario: Interactive terminal required
- **WHEN** stdin is not a TTY
- **THEN** the system SHALL exit with error "Interactive terminal required"

#### Scenario: Minimum terminal size
- **WHEN** terminal is smaller than 80x24
- **THEN** the system SHALL display "Terminal too small (minimum 80x24)"
