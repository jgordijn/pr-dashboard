## 1. Account Detection

- [x] 1.1 Add `GHAccount` struct (Login string, Active bool) to `internal/config/auth.go`
- [x] 1.2 Implement `ListGHAccounts()` in `internal/config/auth.go` that parses `gh auth status --hostname github.com` output and returns `[]GHAccount`
- [x] 1.3 Write tests for `ListGHAccounts` covering multiple accounts, single account, and parse failure cases

## 2. Account Switching

- [x] 2.1 Implement `SwitchGHAccount(user string)` in `internal/config/auth.go` that runs `gh auth switch --user <login>`
- [x] 2.2 Write tests for `SwitchGHAccount` covering success and failure cases

## 3. TUI: Keybinding and Messages

- [x] 3.1 Add `SwitchAccount` keybinding (`s`) to `KeyMap` in `internal/tui/keys.go`
- [x] 3.2 Add `ModalAccountPicker` type to `ModalType` in `internal/tui/messages.go`
- [x] 3.3 Add `AccountsLoadedMsg` and `AccountSwitchedMsg` message types in `internal/tui/messages.go`
- [x] 3.4 Add `Accounts []config.GHAccount` field to `Model` in `internal/tui/app.go`

## 4. TUI: Account Picker Logic

- [x] 4.1 Handle `s` keypress in `handleKeyMsg`: if no action in progress, fire command to list accounts
- [x] 4.2 Handle `AccountsLoadedMsg`: show account picker modal if multiple accounts, or "Only one account available" modal if single
- [x] 4.3 Handle number key presses when account picker modal is open to select an account
- [x] 4.4 Handle `AccountSwitchedMsg`: recreate `github.Client`, update `Config.General.Username`, trigger PR refresh
- [x] 4.5 Handle switch errors: show error modal, leave current state unchanged

## 5. TUI: View Updates

- [x] 5.1 Add account picker modal rendering in `renderModal` in `internal/tui/view.go`
- [x] 5.2 Add active account display to `renderStatusBar` (e.g., `Account: jgordijn`)
- [x] 5.3 Add `s` keybinding to help modal in `renderHelpModal`

## 6. Verification

- [x] 6.1 Run all existing tests to confirm no regressions
- [x] 6.2 Run `go vet` to confirm no static analysis issues
