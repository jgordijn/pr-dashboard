package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jgordijn/pr-dashboard/internal/model"
)

// mouseTarget describes an interactive row in the frame as it is displayed by
// Bubble Tea. Y and Width are terminal-cell coordinates; ANSI bytes never
// participate in either value.
type mouseTarget struct {
	Key   string
	Text  string
	Y     int
	Width int
}

// mouseTargets returns targets for only the rows that are currently rendered
// as the PR tree. Bubble Tea v0.27 drops lines from the top when a frame is
// taller than the terminal, so logical row coordinates are shifted by the same
// suffix offset here.
func (m Model) mouseTargets() []mouseTarget {
	if m.ViewMode != ViewDashboard {
		return nil
	}
	if m.availableWidth() < 24 {
		return nil
	}
	if m.Modal.Type != ModalNone || m.IsLoading || m.Error != nil || len(m.Groups) == 0 || m.countVisiblePRs() == 0 {
		return nil
	}

	keys := m.mouseItemKeys()
	list := strings.TrimSuffix(m.renderPRList(), "\n")
	lines := strings.Split(list, "\n")
	// Both values come from the same visible projection, so each semantic key
	// has exactly one physical line.
	// View is: header, tree, blank separator, status. Its line count is also
	// what Bubble Tea's renderer uses when retaining only the terminal-height
	// suffix.
	viewLineCount := len(strings.Split(m.View(), "\n"))
	topDrop := 0
	if m.Height > 0 && viewLineCount > m.Height {
		topDrop = viewLineCount - m.Height
	}

	targets := make([]mouseTarget, 0, len(keys))
	for i, key := range keys {
		y := 1 + i - topDrop // row zero is the application header
		if y < 0 {
			continue
		}
		width := lipgloss.Width(lines[i])
		if m.Width > 0 {
			width = min(width, m.Width)
		}
		targets = append(targets, mouseTarget{Key: key, Text: lines[i], Y: y, Width: width})
	}
	return targets
}

// mouseItemKeys mirrors the physical rows emitted by renderPRList. In the
// organization projection the organization headers are interactive even
// though keyboard movement historically traverses only PR leaves.
func (m Model) mouseItemKeys() []string {
	if m.GroupingMode == model.GroupingModeRepository {
		return m.visibleItemKeys()
	}
	var keys []string
	for _, group := range m.Groups {
		if m.visiblePRCountInGroup(group) == 0 {
			continue
		}
		keys = append(keys, organizationFocusKey(group.Organization))
		if group.Collapsed {
			continue
		}
		for _, pr := range group.PRs {
			if m.isPRDisplayable(pr) {
				keys = append(keys, pr.Key)
			}
		}
	}
	return keys
}

// handleMouseMsg activates only an unambiguous left-button press. Releases,
// drag motion, wheel events, other buttons, and all non-tree rows are inert.
func (m Model) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	if msg.X < 0 || msg.Y < 0 || (m.Width > 0 && msg.X >= m.Width) || (m.Height > 0 && msg.Y >= m.Height) {
		return m, nil
	}

	targets := m.mouseTargets()
	var clicked *mouseTarget
	for i := range targets {
		target := &targets[i]
		if target.Y == msg.Y && msg.X < target.Width {
			clicked = target
			break
		}
	}
	if clicked == nil {
		return m, nil
	}
	return m.activateMouseTarget(*clicked)
}

func (m Model) activateMouseTarget(clicked mouseTarget) (tea.Model, tea.Cmd) {
	if organization, ok := parseOrganizationFocusKey(clicked.Key); ok {
		m.SelectedKey = clicked.Key
		if m.GroupingMode == model.GroupingModeRepository {
			m = m.toggleTreeOrganization(clicked.Key)
		} else {
			m = m.toggleOrganizationViewNode(organizationFocusKey(organization))
		}
		return m, nil
	}

	if _, _, ok := parseRepositoryFocusKey(clicked.Key); ok {
		m.SelectedKey = clicked.Key
		m = m.toggleRepository(clicked.Key)
		return m, nil
	}

	pr := model.FindPRByKey(m.Groups, clicked.Key)
	if pr == nil {
		return m, nil
	}
	m.SelectedKey = clicked.Key
	return m, m.openBrowserCmd(pr)
}
