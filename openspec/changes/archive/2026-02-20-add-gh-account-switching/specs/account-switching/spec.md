## ADDED Requirements

### Requirement: List authenticated GitHub accounts
The system SHALL detect all authenticated `gh` CLI accounts for `github.com` by parsing the output of `gh auth status --hostname github.com`. Each account SHALL be represented by its login name and whether it is the currently active account.

#### Scenario: Multiple accounts available
- **WHEN** the user has two or more accounts authenticated in the `gh` CLI
- **THEN** the system SHALL return all account logins and mark exactly one as active

#### Scenario: Single account available
- **WHEN** the user has exactly one account authenticated in the `gh` CLI
- **THEN** the system SHALL return that single account marked as active

#### Scenario: gh CLI not authenticated
- **WHEN** `gh auth status` returns a non-zero exit code or no accounts are found
- **THEN** the system SHALL return an error

### Requirement: Switch active GitHub account
The system SHALL switch the active `gh` CLI account by executing `gh auth switch --user <login>`. After a successful switch, the system SHALL recreate the GitHub GraphQL client and update the in-memory username to match the newly active account.

#### Scenario: Successful account switch
- **WHEN** the user selects a different account from the account picker
- **THEN** the system SHALL run `gh auth switch --user <login>`, recreate the GraphQL client, update the username, and trigger a full PR refresh

#### Scenario: Switch to already active account
- **WHEN** the user selects the account that is already active
- **THEN** the system SHALL dismiss the picker without running any commands or refreshing

#### Scenario: Switch command fails
- **WHEN** `gh auth switch` returns a non-zero exit code
- **THEN** the system SHALL display an error modal with the failure message and leave the current account and client unchanged

### Requirement: Account picker UI
The system SHALL provide a modal overlay triggered by the `s` keybinding that displays all authenticated accounts as a numbered list. The currently active account SHALL be visually marked. The user SHALL select an account by pressing its corresponding number key.

#### Scenario: Open account picker with multiple accounts
- **WHEN** the user presses `s` and multiple accounts are available
- **THEN** the system SHALL display a modal listing all accounts with numbers (e.g., `1. jgordijn (active)`, `2. jgordijn-ah`)

#### Scenario: Open account picker with single account
- **WHEN** the user presses `s` and only one account is available
- **THEN** the system SHALL display a modal with the message "Only one account available" instead of a picker

#### Scenario: Dismiss account picker
- **WHEN** the account picker modal is open and the user presses `q`, `Esc`, or `Enter`
- **THEN** the system SHALL close the modal without switching accounts

#### Scenario: Account picker blocked during action
- **WHEN** the user presses `s` while an action (e.g., branch update) is in progress
- **THEN** the system SHALL ignore the keypress

### Requirement: Active account display
The system SHALL display the currently active GitHub account login in the status bar so the user always knows which account is in use.

#### Scenario: Status bar shows active account
- **WHEN** the dashboard is running
- **THEN** the status bar SHALL include the active account login (e.g., `Account: jgordijn`)

#### Scenario: Status bar updates after switch
- **WHEN** the user successfully switches accounts
- **THEN** the status bar SHALL immediately reflect the new account login
