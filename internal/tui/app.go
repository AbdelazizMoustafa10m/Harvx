package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/pipeline"
)

// Compile-time interface compliance check.
var _ tea.Model = Model{}

// Model is the root Bubble Tea model for the Harvx interactive TUI.
// It composes sub-models for the file tree, stats panel, profile selector,
// and help overlay, dispatching messages to each in Update.
type Model struct {
	// Sub-models for each panel.
	fileTree        fileTreeModel
	statsPanel      statsPanelModel
	profileSelector profileSelectorModel
	helpOverlay     helpOverlayModel

	// External dependencies.
	cfg      *config.ResolvedConfig
	pipeline *pipeline.Pipeline

	// Global state.
	keys     KeyMap
	width    int
	height   int
	ready    bool
	quitting bool
	err      error
}

// New creates a new root TUI model with the given resolved configuration
// and pipeline reference. The pipeline must not be nil.
func New(cfg *config.ResolvedConfig, p *pipeline.Pipeline) (Model, error) {
	if p == nil {
		return Model{}, fmt.Errorf("tui: pipeline must not be nil")
	}
	if cfg == nil {
		return Model{}, fmt.Errorf("tui: config must not be nil")
	}

	return Model{
		cfg:             cfg,
		pipeline:        p,
		keys:            DefaultKeyMap(),
		fileTree:        newFileTreeModel(),
		statsPanel:      newStatsPanelModel(),
		profileSelector: newProfileSelectorModel(cfg.ProfileName),
		helpOverlay:     newHelpOverlayModel(),
	}, nil
}

// Init implements tea.Model. It returns no initial command; the TUI waits for
// a WindowSizeMsg from the runtime before rendering.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. It handles global key bindings and dispatches
// messages to the appropriate sub-models.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global key handling takes priority when help is not showing.
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.helpOverlay.visible = !m.helpOverlay.visible
			return m, nil

		case key.Matches(msg, m.keys.Generate):
			if !m.helpOverlay.visible {
				return m, func() tea.Msg { return GenerateRequestedMsg{} }
			}

		case key.Matches(msg, m.keys.ProfileTab):
			if !m.helpOverlay.visible {
				m.profileSelector = m.profileSelector.next()
				return m, func() tea.Msg {
					return ProfileChangedMsg{ProfileName: m.profileSelector.current}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		// Propagate to all sub-models.
		m.fileTree = m.fileTree.updateSize(msg.Width, msg.Height)
		m.statsPanel = m.statsPanel.updateSize(msg.Width, msg.Height)
		return m, nil

	case FileToggledMsg:
		m.fileTree = m.fileTree.handleToggle(msg)
		return m, nil

	case TokenCountUpdatedMsg:
		m.statsPanel = m.statsPanel.handleTokenUpdate(msg)
		return m, nil

	case ProfileChangedMsg:
		m.profileSelector = m.profileSelector.handleChange(msg)
		return m, nil

	case ErrorMsg:
		m.err = msg.Err
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model. It composes the sub-model views into a
// multi-panel layout with the file tree on the left and stats on the right.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if !m.ready {
		return "Initializing..."
	}

	// Help overlay takes over the full screen.
	if m.helpOverlay.visible {
		return m.helpOverlay.view(m.keys, m.width, m.height)
	}

	// Error display.
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	// Calculate panel widths: 60% file tree, 40% stats.
	leftWidth := m.width * 60 / 100
	rightWidth := m.width - leftWidth - 1 // -1 for separator

	// Compose panels.
	leftPanel := m.fileTree.view(leftWidth, m.height-2) // -2 for status bar
	rightPanel := m.statsPanel.view(rightWidth, m.height-2)

	// Join panels side by side.
	separator := lipgloss.NewStyle().
		Width(1).
		Height(m.height - 2).
		Render(strings.Repeat("|\n", m.height-3) + "|")

	mainView := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, separator, rightPanel)

	// Status bar at the bottom.
	statusBar := m.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, mainView, statusBar)
}

// renderStatusBar renders the bottom status bar with key hints.
func (m Model) renderStatusBar() string {
	profile := m.profileSelector.current
	status := fmt.Sprintf(
		" Profile: %s | q: quit | ?: help | enter: generate | tab: profile | space: toggle",
		profile,
	)

	style := lipgloss.NewStyle().
		Width(m.width).
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)

	return style.Render(status)
}

// --- Stub sub-models ---
// These will be fully implemented in subsequent tasks (T-080 to T-085).

// fileTreeModel is a stub for the file tree panel.
type fileTreeModel struct {
	files    []string
	selected map[string]bool
	cursor   int
	width    int
	height   int
}

func newFileTreeModel() fileTreeModel {
	return fileTreeModel{
		selected: make(map[string]bool),
	}
}

func (m fileTreeModel) updateSize(w, h int) fileTreeModel {
	m.width = w * 60 / 100
	m.height = h - 2
	return m
}

func (m fileTreeModel) handleToggle(msg FileToggledMsg) fileTreeModel {
	m.selected[msg.Path] = msg.Included
	return m
}

func (m fileTreeModel) view(width, height int) string {
	style := lipgloss.NewStyle().
		Width(width).
		Height(height)

	if len(m.files) == 0 {
		return style.Render("  File tree (loading...)")
	}

	var b strings.Builder
	for i, f := range m.files {
		if i >= height {
			break
		}
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		check := "[ ]"
		if m.selected[f] {
			check = "[x]"
		}
		fmt.Fprintf(&b, "%s%s %s\n", prefix, check, f)
	}
	return style.Render(b.String())
}

// statsPanelModel is a stub for the stats/token panel.
type statsPanelModel struct {
	totalTokens int
	fileCount   int
	budgetUsed  float64
	width       int
	height      int
}

func newStatsPanelModel() statsPanelModel {
	return statsPanelModel{}
}

func (m statsPanelModel) updateSize(w, h int) statsPanelModel {
	m.width = w - (w * 60 / 100) - 1
	m.height = h - 2
	return m
}

func (m statsPanelModel) handleTokenUpdate(msg TokenCountUpdatedMsg) statsPanelModel {
	m.totalTokens = msg.TotalTokens
	m.fileCount = msg.FileCount
	m.budgetUsed = msg.BudgetUsed
	return m
}

func (m statsPanelModel) view(width, height int) string {
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(0, 1)

	content := fmt.Sprintf(
		"Stats\n-----\nFiles: %d\nTokens: %d\nBudget: %.1f%%",
		m.fileCount, m.totalTokens, m.budgetUsed,
	)

	return style.Render(content)
}

// profileSelectorModel is a stub for the profile selector.
type profileSelectorModel struct {
	current  string
	profiles []string
	index    int
}

func newProfileSelectorModel(currentProfile string) profileSelectorModel {
	return profileSelectorModel{
		current:  currentProfile,
		profiles: []string{currentProfile},
	}
}

func (m profileSelectorModel) next() profileSelectorModel {
	if len(m.profiles) <= 1 {
		return m
	}
	m.index = (m.index + 1) % len(m.profiles)
	m.current = m.profiles[m.index]
	return m
}

func (m profileSelectorModel) handleChange(msg ProfileChangedMsg) profileSelectorModel {
	m.current = msg.ProfileName
	return m
}

// helpOverlayModel is a stub for the help overlay.
type helpOverlayModel struct {
	visible bool
}

func newHelpOverlayModel() helpOverlayModel {
	return helpOverlayModel{}
}

func (m helpOverlayModel) view(keys KeyMap, width, height int) string {
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center).
		AlignVertical(lipgloss.Center)

	var b strings.Builder
	b.WriteString("Harvx Interactive Mode\n")
	b.WriteString("======================\n\n")

	for _, group := range keys.FullHelp() {
		for _, k := range group {
			fmt.Fprintf(&b, "  %-12s %s\n", k.Help().Key, k.Help().Desc)
		}
		b.WriteString("\n")
	}

	b.WriteString("\nPress ? to close help")

	return style.Render(b.String())
}
