## MODIFIED Requirements
### Requirement: Help Display
The system SHALL provide a compact help screen showing keybindings and the status-symbol legend.

#### Scenario: Show help
- **WHEN** user presses '?'
- **THEN** a modal SHALL display all keybindings and the CI/review/merge symbol legend
- **AND** the content SHALL fit an 80x24 terminal

#### Scenario: Dismiss help
- **WHEN** help modal is displayed and user presses 'q', Escape, or Enter
- **THEN** the help modal SHALL close
