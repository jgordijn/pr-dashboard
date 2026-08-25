package tui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jgordijn/pr-dashboard/internal/hidden"
	"github.com/jgordijn/pr-dashboard/internal/model"
)

type ViewMode int

const (
	ViewDashboard ViewMode = iota
	ViewHiddenItems
)

type hiddenFilter int

const (
	hiddenFilterAll hiddenFilter = iota
	hiddenFilterRepositories
	hiddenFilterPRs
)

type HiddenManagerState struct {
	Cursor      int
	Filter      hiddenFilter
	Query       string
	Searching   bool
	Message     string
	PreviousKey string
}

func (m *Model) hiddenAccount() string {
	if m.Config == nil {
		return ""
	}
	return m.Config.General.Username
}

func (m *Model) hiddenEntries() []hidden.Entry {
	if m.HiddenState == nil {
		return nil
	}
	entries := m.HiddenState.Entries(m.hiddenAccount())
	query := strings.ToLower(strings.TrimSpace(m.HiddenManager.Query))
	filtered := make([]hidden.Entry, 0, len(entries))
	for _, entry := range entries {
		if m.HiddenManager.Filter == hiddenFilterRepositories && entry.Kind != hidden.KindRepository {
			continue
		}
		if m.HiddenManager.Filter == hiddenFilterPRs && entry.Kind != hidden.KindPullRequest {
			continue
		}
		search := strings.ToLower(entry.Organization + "/" + entry.Repository + " " + entry.Title)
		if entry.Kind == hidden.KindPullRequest {
			search += fmt.Sprintf(" #%d", entry.Number)
		}
		if query != "" && !strings.Contains(search, query) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func (m *Model) hiddenRuleCount() int {
	if m.HiddenState == nil {
		return 0
	}
	return m.HiddenState.RuleCount(m.hiddenAccount())
}

func (m *Model) isPRHidden(pr model.PullRequest) bool {
	return m.HiddenState != nil && m.HiddenState.IsPRHidden(m.hiddenAccount(), pr.Organization, pr.Repository, pr.Number)
}

func (m *Model) isPRDisplayable(pr model.PullRequest) bool {
	return (m.ShowDrafts || !pr.IsDraft) && !m.isPRHidden(pr)
}
func (m *Model) visiblePRCountInGroup(group model.PRGroup) int {
	count := 0
	for _, pr := range group.PRs {
		if m.isPRDisplayable(pr) {
			count++
		}
	}
	return count
}
func (m *Model) projectionExclusions() (hiddenCount, draftCount int) {
	for _, group := range m.Groups {
		for _, pr := range group.PRs {
			if m.isPRHidden(pr) {
				hiddenCount++
			} else if !m.ShowDrafts && pr.IsDraft {
				draftCount++
			}
		}
	}
	return
}

func (m Model) saveHidden(next *hidden.State) error {
	if m.HiddenLoadErr != nil {
		return fmt.Errorf("hidden items unavailable: %w", m.HiddenLoadErr)
	}
	if m.HiddenStore == nil {
		return fmt.Errorf("hidden items persistence is unavailable")
	}
	return m.HiddenStore.Save(next)
}

func (m Model) hideFocusedItem() (tea.Model, tea.Cmd) {
	before := m.visibleItemKeys()
	index := indexOfKey(before, m.SelectedKey)
	next := m.HiddenState.Clone()
	var entry hidden.Entry
	added := false
	if owner, repo, ok := parseRepositoryFocusKey(m.SelectedKey); ok {
		entry = hidden.Entry{Kind: hidden.KindRepository, Organization: owner, Repository: repo, Title: repo, HiddenAt: time.Now()}
		added = next.HideRepository(m.hiddenAccount(), owner, repo, repo, entry.HiddenAt)
	} else if pr := m.SelectedPR(); pr != nil {
		entry = hidden.Entry{Kind: hidden.KindPullRequest, Organization: pr.Organization, Repository: pr.Repository, Number: pr.Number, Title: pr.Title, HiddenAt: time.Now()}
		added = next.HidePR(m.hiddenAccount(), pr.Organization, pr.Repository, pr.Number, pr.Title, entry.HiddenAt)
	} else {
		m.FlashMessage = "Organizations cannot be hidden · focus a repository or pull request"
		return m, nil
	}
	if !added {
		m.FlashMessage = "That item is already hidden"
		return m, nil
	}
	if err := m.saveHidden(next); err != nil {
		return m.visibilityError("Could not save hidden items", err)
	}
	m.HiddenState = next
	m.LastHidden = &entry
	m.FlashMessage = "Hidden " + hiddenEntryLabel(entry) + " · z undo · M manage"
	m.SelectedKey = nearestKeyAt(m.visibleItemKeys(), index)
	return m, nil
}

func (m Model) undoLastHide() (tea.Model, tea.Cmd) {
	if m.LastHidden == nil {
		m.FlashMessage = "Nothing to undo"
		return m, nil
	}
	next := m.HiddenState.Clone()
	entry := *m.LastHidden
	if !next.Unhide(m.hiddenAccount(), entry) {
		m.LastHidden = nil
		m.FlashMessage = "Nothing to undo"
		return m, nil
	}
	if err := m.saveHidden(next); err != nil {
		return m.visibilityError("Could not restore hidden item", err)
	}
	m.HiddenState = next
	m.LastHidden = nil
	m.SelectedKey = hiddenEntryKey(entry)
	m.SelectedKey = m.findNearestVisibleKey()
	m.FlashMessage = "Restored " + hiddenEntryLabel(entry)
	return m, nil
}

func (m Model) openHiddenManager() (tea.Model, tea.Cmd) {
	m.ViewMode = ViewHiddenItems
	m.FlashMessage = ""
	m.HiddenManager = HiddenManagerState{PreviousKey: m.SelectedKey}
	return m, nil
}

func (m Model) closeHiddenManager() (tea.Model, tea.Cmd) {
	previous := m.HiddenManager.PreviousKey
	m.ViewMode = ViewDashboard
	m.HiddenManager = HiddenManagerState{}
	m.SelectedKey = previous
	m.SelectedKey = m.findNearestVisibleKey()
	return m, nil
}

func (m Model) unhideManagerSelection() (tea.Model, tea.Cmd) {
	items := m.hiddenEntries()
	if len(items) == 0 {
		return m, nil
	}
	m.HiddenManager.Cursor = clampCursor(m.HiddenManager.Cursor, len(items))
	entry := items[m.HiddenManager.Cursor]
	next := m.HiddenState.Clone()
	next.Unhide(m.hiddenAccount(), entry)
	if err := m.saveHidden(next); err != nil {
		m.HiddenManager.Message = "Could not restore hidden item · " + err.Error()
		return m, nil
	}
	m.HiddenState = next
	m.LastHidden = nil
	m.HiddenManager.Message = "Restored " + hiddenEntryLabel(entry)
	items = m.hiddenEntries()
	m.HiddenManager.Cursor = clampCursor(m.HiddenManager.Cursor, len(items))
	return m, nil
}

func (m Model) handleHiddenManagerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	selected, hasSelection := m.managerSelection()
	oldCursor := m.HiddenManager.Cursor
	if m.HiddenManager.Searching {
		switch key {
		case "esc":
			m.HiddenManager.Searching = false
			m.HiddenManager.Query = ""
			m.restoreManagerSelection(selected, hasSelection, oldCursor)
		case "enter":
			m.HiddenManager.Searching = false
		case "backspace", "ctrl+h":
			if m.HiddenManager.Query != "" {
				_, size := utf8.DecodeLastRuneInString(m.HiddenManager.Query)
				m.HiddenManager.Query = m.HiddenManager.Query[:len(m.HiddenManager.Query)-size]
				m.restoreManagerSelection(selected, hasSelection, oldCursor)
			}
		default:
			if len(msg.Runes) > 0 && !msg.Alt {
				m.HiddenManager.Query += string(msg.Runes)
				m.restoreManagerSelection(selected, hasSelection, oldCursor)
			}
		}
		return m, nil
	}
	switch key {
	case "q":
		return m.closeHiddenManager()
	case "esc":
		if m.HiddenManager.Query != "" {
			m.HiddenManager.Query = ""
			m.restoreManagerSelection(selected, hasSelection, oldCursor)
			return m, nil
		}
		return m.closeHiddenManager()
	case "/":
		m.HiddenManager.Searching = true
		return m, nil
	case "tab":
		m.HiddenManager.Filter = (m.HiddenManager.Filter + 1) % 3
		m.restoreManagerSelection(selected, hasSelection, oldCursor)
		return m, nil
	case "shift+tab":
		m.HiddenManager.Filter = (m.HiddenManager.Filter + 2) % 3
		m.restoreManagerSelection(selected, hasSelection, oldCursor)
		return m, nil
	case "j", "down":
		items := m.hiddenEntries()
		if m.HiddenManager.Cursor < len(items)-1 {
			m.HiddenManager.Cursor++
		}
		return m, nil
	case "k", "up":
		if m.HiddenManager.Cursor > 0 {
			m.HiddenManager.Cursor--
		}
		return m, nil
	case "g":
		m.HiddenManager.Cursor = 0
		return m, nil
	case "G":
		items := m.hiddenEntries()
		if len(items) > 0 {
			m.HiddenManager.Cursor = len(items) - 1
		}
		return m, nil
	case "u", "enter":
		return m.unhideManagerSelection()
	}
	return m, nil
}

func (m *Model) managerSelection() (hidden.Entry, bool) {
	items := m.hiddenEntries()
	if len(items) == 0 {
		return hidden.Entry{}, false
	}
	cursor := clampCursor(m.HiddenManager.Cursor, len(items))
	return items[cursor], true
}
func (m *Model) restoreManagerSelection(selected hidden.Entry, hasSelection bool, oldCursor int) {
	items := m.hiddenEntries()
	if hasSelection {
		for i, entry := range items {
			if entry.Kind == selected.Kind && hiddenEntryKey(entry) == hiddenEntryKey(selected) {
				m.HiddenManager.Cursor = i
				return
			}
		}
	}
	m.HiddenManager.Cursor = clampCursor(oldCursor, len(items))
}

func (m Model) renderHiddenManager() string {
	width := m.availableWidth()
	items := m.hiddenEntries()
	all, repos, prs := m.hiddenCounts()
	var b strings.Builder
	line := func(text string) { b.WriteString(m.truncateForMode(text, width)); b.WriteByte('\n') }
	styled := func(text string, style lipgloss.Style) {
		b.WriteString(style.Render(m.truncateForMode(text, width)))
		b.WriteByte('\n')
	}

	styled(fmt.Sprintf("Hidden Items · %d rules", all), m.Styles.HeaderStyle)
	styled("Repositories and pull requests excluded from every dashboard view", m.Styles.DimStyle)
	if m.HiddenLoadErr != nil {
		styled("Persistence unavailable · "+m.HiddenLoadErr.Error(), m.Styles.StatusFailingStyle)
	}
	b.WriteByte('\n')
	filterNames := []string{"All", "Repositories", "Pull requests"}
	line(fmt.Sprintf("[%s] %d   %s %d   %s %d   / search", filterNames[m.HiddenManager.Filter], all, filterNames[1], repos, filterNames[2], prs))
	b.WriteByte('\n')
	showSearch := m.HiddenManager.Searching || m.HiddenManager.Query != ""
	if showSearch {
		prompt := "Search hidden: " + m.HiddenManager.Query
		if m.HiddenManager.Searching {
			prompt += "_"
		}
		styled(prompt, m.Styles.StatusPendingStyle)
	}
	if len(items) == 0 {
		if m.HiddenManager.Query != "" {
			line("No hidden items match “" + m.HiddenManager.Query + "”")
			line("Esc clears the search")
		} else {
			line("Nothing is hidden")
			line("Focus a repository or PR and press H.")
		}
	} else {
		m.HiddenManager.Cursor = clampCursor(m.HiddenManager.Cursor, len(items))
		extra := 0
		if m.HiddenLoadErr != nil {
			extra++
		}
		if showSearch {
			extra++
		}
		if m.HiddenManager.Message != "" {
			extra++
		}
		start, end := managerWindow(m.HiddenManager.Cursor, len(items), m.Height-extra)
		for i := start; i < end; i++ {
			entry := items[i]
			gutter := "  "
			if i == m.HiddenManager.Cursor {
				gutter = m.Symbols.Selected + " "
			}
			kind := "[PR]"
			if entry.Kind == hidden.KindRepository {
				kind = "[REPO]"
			}
			text := fmt.Sprintf("%s %-6s %s", gutter, kind, hiddenEntryLabel(entry))
			if entry.Title != "" && entry.Kind == hidden.KindPullRequest {
				text += "  " + entry.Title
			}
			if i == m.HiddenManager.Cursor {
				styled(text, m.Styles.SelectedStyle)
			} else {
				line(text)
			}
		}
	}
	b.WriteByte('\n')
	if m.HiddenManager.Message != "" {
		styled(m.HiddenManager.Message, m.Styles.StatusPassingStyle)
	}
	b.WriteString(m.Styles.DimStyle.Render(m.truncateForMode("j/k move  u/Enter unhide  / search  Tab type  q/Esc close", width)))
	return b.String()
}
func (m *Model) hiddenCounts() (all, repositories, prs int) {
	if m.HiddenState == nil {
		return
	}
	for _, entry := range m.HiddenState.Entries(m.hiddenAccount()) {
		all++
		if entry.Kind == hidden.KindRepository {
			repositories++
		} else {
			prs++
		}
	}
	return
}
func (m Model) visibilityError(title string, err error) (tea.Model, tea.Cmd) {
	m.Modal = ModalState{Type: ModalError, Title: title, Message: err.Error()}
	return m, nil
}
func hiddenEntryLabel(entry hidden.Entry) string {
	if entry.Kind == hidden.KindRepository {
		return entry.Organization + "/" + entry.Repository
	}
	return fmt.Sprintf("%s/%s#%d", entry.Organization, entry.Repository, entry.Number)
}
func hiddenEntryKey(entry hidden.Entry) string {
	if entry.Kind == hidden.KindRepository {
		return repositoryFocusKey(entry.Organization, entry.Repository)
	}
	return hiddenEntryLabel(entry)
}
func indexOfKey(keys []string, key string) int {
	for i, candidate := range keys {
		if candidate == key {
			return i
		}
	}
	return len(keys)
}
func nearestKeyAt(keys []string, index int) string {
	if len(keys) == 0 {
		return ""
	}
	if index >= len(keys) {
		index = len(keys) - 1
	}
	if index < 0 {
		index = 0
	}
	return keys[index]
}
func clampCursor(cursor, length int) int {
	if length == 0 {
		return 0
	}
	if cursor < 0 {
		return 0
	}
	if cursor >= length {
		return length - 1
	}
	return cursor
}
func managerWindow(cursor, length, height int) (int, int) {
	cursor = clampCursor(cursor, length)
	capacity := max(1, height-8)
	if height <= 0 {
		capacity = 12
	}
	if capacity > length {
		capacity = length
	}
	start := cursor - capacity + 1
	if start < 0 {
		start = 0
	}
	end := start + capacity
	return start, end
}
