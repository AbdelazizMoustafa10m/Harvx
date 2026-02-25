package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/discovery"
	"github.com/harvx/harvx/internal/pipeline"
	"github.com/harvx/harvx/internal/tui/filetree"
	"github.com/harvx/harvx/internal/tui/stats"
)

// Compile-time interface compliance check.
var _ tea.Model = Model{}

// Model is the root Bubble Tea model for the Harvx interactive TUI.
// It composes sub-models for the file tree, stats panel, profile selector,
// and help overlay, dispatching messages to each in Update.
type Model struct {
	// Sub-models for each panel.
	fileTree        filetree.Model
	statsPanel      stats.Model
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

// Options holds optional configuration for the root TUI model.
type Options struct {
	// RootDir is the directory to browse. Defaults to "." if empty.
	RootDir string

	// Ignorer filters files from the tree. May be nil for no filtering.
	Ignorer discovery.Ignorer
}

// New creates a new root TUI model with the given resolved configuration
// and pipeline reference. The pipeline must not be nil.
func New(cfg *config.ResolvedConfig, p *pipeline.Pipeline, opts ...Options) (Model, error) {
	if p == nil {
		return Model{}, fmt.Errorf("tui: pipeline must not be nil")
	}
	if cfg == nil {
		return Model{}, fmt.Errorf("tui: config must not be nil")
	}

	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.RootDir == "" {
		o.RootDir = "."
	}

	ft := filetree.New(o.RootDir, o.Ignorer)

	sp := stats.New(stats.Options{
		MaxTokens:     cfg.Profile.MaxTokens,
		ProfileName:   cfg.ProfileName,
		TargetName:    cfg.Profile.Target,
		TokenizerName: cfg.Profile.Tokenizer,
		Compression:   cfg.Profile.Compression,
	})
	sp.SetTreeRoot(ft.Root())

	return Model{
		cfg:             cfg,
		pipeline:        p,
		keys:            DefaultKeyMap(),
		fileTree:        ft,
		statsPanel:      sp,
		profileSelector: newProfileSelectorModel(cfg.ProfileName),
		helpOverlay:     newHelpOverlayModel(),
	}, nil
}

// Init implements tea.Model. It returns the file tree's init command to begin
// scanning the root directory.
func (m Model) Init() tea.Cmd {
	return m.fileTree.Init()
}

// FileTree returns the file tree sub-model. This is useful for testing.
func (m Model) FileTree() filetree.Model {
	return m.fileTree
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

		// Forward key events to file tree when help is not showing.
		if !m.helpOverlay.visible {
			var cmd tea.Cmd
			m.fileTree, cmd = m.fileTree.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		// Propagate to file tree and stats panel.
		leftWidth := msg.Width * 60 / 100
		rightWidth := msg.Width - leftWidth - 1
		m.fileTree.SetSize(leftWidth, msg.Height-2)
		m.statsPanel.SetSize(rightWidth, msg.Height-2)
		return m, nil

	case filetree.DirLoadedMsg:
		var cmd tea.Cmd
		m.fileTree, cmd = m.fileTree.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case FileToggledMsg:
		// FileToggledMsg comes from the file tree; forward to stats for
		// debounced token recalculation.
		m.statsPanel.SetTreeRoot(m.fileTree.Root())
		var statsCmd tea.Cmd
		var updated tea.Model
		updated, statsCmd = m.statsPanel.Update(msg)
		m.statsPanel = updated.(stats.Model)
		if statsCmd != nil {
			cmds = append(cmds, statsCmd)
		}
		return m, tea.Batch(cmds...)

	case TokenCountUpdatedMsg:
		var statsCmd tea.Cmd
		var updated tea.Model
		updated, statsCmd = m.statsPanel.Update(msg)
		m.statsPanel = updated.(stats.Model)
		if statsCmd != nil {
			cmds = append(cmds, statsCmd)
		}
		return m, tea.Batch(cmds...)

	case ProfileChangedMsg:
		m.profileSelector = m.profileSelector.handleChange(msg)
		// Also forward to stats panel.
		var statsCmd tea.Cmd
		var updated tea.Model
		updated, statsCmd = m.statsPanel.Update(msg)
		m.statsPanel = updated.(stats.Model)
		if statsCmd != nil {
			cmds = append(cmds, statsCmd)
		}
		return m, tea.Batch(cmds...)

	case ErrorMsg:
		m.err = msg.Err
		return m, nil

	default:
		// Forward unrecognised messages to the stats panel. This handles
		// stats-internal messages (recalcTickMsg, tokenCountResult) that
		// are returned as tea.Cmd results and re-dispatched by Bubble Tea.
		var statsCmd tea.Cmd
		var updated tea.Model
		updated, statsCmd = m.statsPanel.Update(msg)
		m.statsPanel = updated.(stats.Model)
		if statsCmd != nil {
			cmds = append(cmds, statsCmd)
		}
		return m, tea.Batch(cmds...)
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

	// Compose panels.
	leftStyle := lipgloss.NewStyle().
		Width(leftWidth).
		Height(m.height - 2)
	leftPanel := leftStyle.Render(m.fileTree.View())
	rightPanel := m.statsPanel.View()

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
		" Profile: %s | q: quit | ?: help | ctrl+g: generate | tab: profile | space: toggle",
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
// Profile selector and help overlay remain as stubs.
// They will be fully implemented in subsequent tasks (T-083 to T-085).

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
