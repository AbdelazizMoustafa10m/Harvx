package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/harvx/harvx/internal/buildinfo"
	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/discovery"
	"github.com/harvx/harvx/internal/pipeline"
	"github.com/harvx/harvx/internal/tui/filetree"
	"github.com/harvx/harvx/internal/tui/help"
	"github.com/harvx/harvx/internal/tui/profile"
	"github.com/harvx/harvx/internal/tui/search"
	"github.com/harvx/harvx/internal/tui/stats"
)

// Compile-time interface compliance check.
var _ tea.Model = Model{}

// Model is the root Bubble Tea model for the Harvx interactive TUI.
// It composes sub-models for the file tree, stats panel, profile selector,
// overlay system, toast messages, search, and help overlay, dispatching
// messages to each in Update.
type Model struct {
	// Sub-models for each panel.
	fileTree        filetree.Model
	statsPanel      stats.Model
	profileSelector profile.Model
	helpOverlay     help.Model
	searchModel     search.Model
	overlay         overlayModel
	toast           toastModel

	// External dependencies.
	cfg       *config.ResolvedConfig
	pipeline  *pipeline.Pipeline
	clipboard Clipboarder

	// Global state.
	keys     KeyMap
	styles   Styles
	dir      string
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

	// ProfileNames is the list of all available profile names from config.
	// If empty, only the active profile name is shown.
	ProfileNames []string

	// Clipboard is the clipboard implementation. Defaults to a noop that
	// returns ErrClipboardUnavailable.
	Clipboard Clipboarder
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
	if o.Clipboard == nil {
		o.Clipboard = noopClipboard{}
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

	ps := profile.New(cfg.ProfileName, o.ProfileNames)

	return Model{
		cfg:             cfg,
		pipeline:        p,
		clipboard:       o.Clipboard,
		keys:            DefaultKeyMap(),
		dir:             o.RootDir,
		fileTree:        ft,
		statsPanel:      sp,
		profileSelector: ps,
		helpOverlay:     help.New(),
		searchModel:     search.New(),
		overlay:         newOverlayModel(),
		toast:           newToastModel(),
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

// ProfileSelector returns the profile selector sub-model. Useful for testing.
func (m Model) ProfileSelector() profile.Model {
	return m.profileSelector
}

// Overlay returns the overlay model. Useful for testing.
func (m Model) Overlay() overlayModel {
	return m.overlay
}

// Toast returns the toast model. Useful for testing.
func (m Model) Toast() toastModel {
	return m.toast
}

// Update implements tea.Model. It handles global key bindings and dispatches
// messages to the appropriate sub-models. When an overlay is active, key
// events are routed to the overlay instead of the normal key handlers.
// Priority order: overlay > help overlay > search mode > global keys > filetree.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If an overlay is active, route keys to overlay handling.
		if m.overlay.Active() {
			return m.handleOverlayKey(msg)
		}

		// Help overlay intercepts all keys when visible.
		if m.helpOverlay.Visible {
			var cmd tea.Cmd
			m.helpOverlay, cmd = m.helpOverlay.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// Search mode intercepts all keys when active.
		if m.searchModel.Active() {
			var cmd tea.Cmd
			m.searchModel, cmd = m.searchModel.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			// Live-update the filetree filter as user types.
			m.fileTree.SetSearchFilter(m.searchModel.Query())
			return m, tea.Batch(cmds...)
		}

		// Global key handling.
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.helpOverlay = m.helpOverlay.Toggle()
			return m, nil

		case key.Matches(msg, m.keys.Search):
			// If a filter is already active, a second "/" clears it.
			if m.searchModel.Filtered() {
				var cmd tea.Cmd
				m.searchModel, cmd = m.searchModel.ClearFilter()
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				m.fileTree.SetSearchFilter("")
				return m, tea.Batch(cmds...)
			}
			var cmd tea.Cmd
			m.searchModel, cmd = m.searchModel.Activate()
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.TierView):
			m.fileTree.CycleTierFilter()
			return m, nil

		case key.Matches(msg, m.keys.SelectAll):
			m.fileTree.SelectAllVisible()
			m.statsPanel.SetTreeRoot(m.fileTree.Root())
			return m, nil

		case key.Matches(msg, m.keys.SelectNone):
			m.fileTree.DeselectAllVisible()
			m.statsPanel.SetTreeRoot(m.fileTree.Root())
			return m, nil

		case key.Matches(msg, m.keys.ClearFilter):
			var cmd tea.Cmd
			m.searchModel, cmd = m.searchModel.ClearFilter()
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			m.fileTree.ClearAllFilters()
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.Generate):
			return m.handleGenerate()

		case key.Matches(msg, m.keys.Preview):
			return m.handlePreview()

		case key.Matches(msg, m.keys.Save):
			return m.handleSaveProfile()

		case key.Matches(msg, m.keys.Export):
			return m.handleExportClipboard()

		case key.Matches(msg, m.keys.ProfileTab):
			var cmd tea.Cmd
			m.profileSelector, cmd = m.profileSelector.Next()
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.ProfileBackTab):
			var cmd tea.Cmd
			m.profileSelector, cmd = m.profileSelector.Prev()
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// Forward key events to file tree.
		var cmd tea.Cmd
		m.fileTree, cmd = m.fileTree.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case search.FilterAppliedMsg:
		m.fileTree.SetSearchFilter(msg.Query)
		return m, nil

	case search.FilterClearedMsg:
		m.fileTree.SetSearchFilter("")
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		// Recompute styles for the new dimensions.
		m.styles = NewStyles(lipgloss.HasDarkBackground(), msg.Width, msg.Height)
		// Propagate computed panel sizes to sub-models.
		m.fileTree.SetSize(m.styles.LeftPanelWidth-4, m.styles.ContentHeight-2)
		m.fileTree.SetDark(lipgloss.HasDarkBackground())
		m.statsPanel.SetSize(m.styles.RightPanelWidth-4, m.styles.ContentHeight-2)
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
		// Update profile selector.
		updated, _ := m.profileSelector.Update(msg)
		m.profileSelector = updated.(profile.Model)
		// Also forward to stats panel.
		var statsCmd tea.Cmd
		var statsUpdated tea.Model
		statsUpdated, statsCmd = m.statsPanel.Update(msg)
		m.statsPanel = statsUpdated.(stats.Model)
		if statsCmd != nil {
			cmds = append(cmds, statsCmd)
		}
		return m, tea.Batch(cmds...)

	case generateCompleteMsg:
		return m.handleGenerateComplete(msg)

	case saveProfileCompleteMsg:
		return m.handleSaveProfileComplete(msg)

	case clipboardCompleteMsg:
		return m.handleClipboardComplete(msg)

	case toastDismissMsg:
		m.toast.dismiss(msg)
		return m, nil

	case ErrorMsg:
		m.err = msg.Err
		return m, nil

	default:
		// Forward to overlay if active (handles spinner tick, text input blink).
		if m.overlay.Active() {
			cmd := m.overlay.update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

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
}

// handleOverlayKey routes key events when an overlay is active.
func (m Model) handleOverlayKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch m.overlay.state {
	case overlayGenerating:
		// While generating, only allow quit.
		if key.Matches(msg, m.keys.Quit) {
			m.overlay.close()
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	case overlayPreviewing:
		// Esc or q dismisses the preview.
		if key.Matches(msg, m.keys.Quit) {
			m.overlay.close()
			return m, nil
		}
		return m, nil

	case overlaySavingProfile:
		switch msg.Type {
		case tea.KeyEnter:
			return m.handleSaveProfileConfirm()
		case tea.KeyEscape:
			m.overlay.close()
			return m, nil
		default:
			// Forward to text input.
			cmd := m.overlay.update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// View implements tea.Model. It composes the sub-model views into a
// multi-panel layout with the file tree on the left and stats on the right.
// The layout adapts to terminal size using the pre-computed Styles.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if !m.ready {
		return "Initializing..."
	}

	// Help overlay takes over the full screen.
	if m.helpOverlay.Visible {
		isDark := isDarkTheme(m.styles.Colors)
		return help.View(m.width, m.height, isDark)
	}

	// Overlay takes over the full screen.
	if m.overlay.Active() {
		return m.overlay.view(m.width, m.height)
	}

	// Error display.
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	// Build the title bar and status bar.
	titleBar := RenderTitleBar(buildinfo.Version, m.dir, m.width, m.styles)

	var statusBar string
	if m.toast.visible {
		statusBar = m.toast.view(m.width)
	} else {
		statusBar = RenderStatusBar(m.profileSelector.Current(), m.width, m.styles)
	}

	// Build the file tree view. If search is active, append the search
	// input below the tree content.
	fileTreeView := m.fileTree.View()
	if m.searchModel.Active() {
		fileTreeView = fileTreeView + "\n" + m.searchModel.View()
	}

	// Build the dynamic panel title based on active filters.
	panelTitle := m.buildPanelTitle()

	// Compose via the layout system.
	return RenderLayout(LayoutParams{
		Styles:        m.styles,
		FileTreeView:  fileTreeView,
		StatsView:     m.statsPanel.View(),
		TitleBar:      titleBar,
		StatusBar:     statusBar,
		FileTreeTitle: panelTitle,
		Mode:          m.styles.Layout,
		Width:         m.width,
		Height:        m.height,
	})
}

// buildPanelTitle returns a dynamic panel title based on active filters.
func (m Model) buildPanelTitle() string {
	filter := m.fileTree.Filter()
	title := "Files"

	if filter.HasTierFilter() {
		title += " | Tier: " + filter.TierLabel()
	}
	if filter.HasSearchFilter() {
		title += " | Filter: " + filter.SearchQuery
	}

	return title
}

// Styles returns the current computed styles. Useful for testing.
func (m Model) Styles() Styles {
	return m.styles
}

// SearchModel returns the search sub-model. Useful for testing.
func (m Model) SearchModel() search.Model {
	return m.searchModel
}

// HelpOverlay returns the help overlay model. Useful for testing.
func (m Model) HelpOverlay() help.Model {
	return m.helpOverlay
}
