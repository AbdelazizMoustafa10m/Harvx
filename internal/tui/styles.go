package tui

import "github.com/charmbracelet/lipgloss"

// LayoutMode describes the terminal layout based on available dimensions.
type LayoutMode int

const (
	// LayoutTooSmall indicates the terminal is too small to render
	// (width < 40 or height < 12).
	LayoutTooSmall LayoutMode = iota + 1

	// LayoutSinglePanel uses a single full-width panel (width 40-59).
	LayoutSinglePanel

	// LayoutCompressed shows both panels in a compressed layout (width 60-99).
	LayoutCompressed

	// LayoutFull shows both panels at full size (width >= 100).
	LayoutFull
)

// Minimum terminal dimensions below which the TUI refuses to render.
const (
	MinTermWidth  = 40
	MinTermHeight = 12
)

// ThemeColors holds the ANSI 256 color palette for a single theme variant.
type ThemeColors struct {
	Background     lipgloss.Color
	Foreground     lipgloss.Color
	Accent         lipgloss.Color
	BorderActive   lipgloss.Color
	BorderInactive lipgloss.Color
	TitleBarBg     lipgloss.Color
	TitleBarFg     lipgloss.Color
	StatusBarBg    lipgloss.Color
	StatusBarFg    lipgloss.Color

	// Tier colors for file tree rendering.
	TierGold    lipgloss.Color // Tier 0
	TierGreen   lipgloss.Color // Tier 1
	TierBlue    lipgloss.Color // Tier 2
	TierCyan    lipgloss.Color // Tier 3
	TierMagenta lipgloss.Color // Tier 4
	TierDim     lipgloss.Color // Tier 5

	// File tree specific.
	IncludedGreen lipgloss.Color
	ExcludedGray  lipgloss.Color
	PartialYellow lipgloss.Color
	CursorBg      lipgloss.Color
	SecretRed     lipgloss.Color
	TokenCountDim lipgloss.Color
}

// darkColors returns the dark-theme color palette using ANSI 256 codes.
func darkColors() ThemeColors {
	return ThemeColors{
		Background:     lipgloss.Color("235"),
		Foreground:     lipgloss.Color("252"),
		Accent:         lipgloss.Color("117"),
		BorderActive:   lipgloss.Color("117"),
		BorderInactive: lipgloss.Color("240"),
		TitleBarBg:     lipgloss.Color("117"),
		TitleBarFg:     lipgloss.Color("235"),
		StatusBarBg:    lipgloss.Color("236"),
		StatusBarFg:    lipgloss.Color("252"),

		TierGold:    lipgloss.Color("220"),
		TierGreen:   lipgloss.Color("34"),
		TierBlue:    lipgloss.Color("33"),
		TierCyan:    lipgloss.Color("36"),
		TierMagenta: lipgloss.Color("133"),
		TierDim:     lipgloss.Color("240"),

		IncludedGreen: lipgloss.Color("34"),
		ExcludedGray:  lipgloss.Color("240"),
		PartialYellow: lipgloss.Color("220"),
		CursorBg:      lipgloss.Color("238"),
		SecretRed:     lipgloss.Color("196"),
		TokenCountDim: lipgloss.Color("242"),
	}
}

// lightColors returns the light-theme color palette using ANSI 256 codes.
func lightColors() ThemeColors {
	return ThemeColors{
		Background:     lipgloss.Color("253"),
		Foreground:     lipgloss.Color("237"),
		Accent:         lipgloss.Color("33"),
		BorderActive:   lipgloss.Color("33"),
		BorderInactive: lipgloss.Color("246"),
		TitleBarBg:     lipgloss.Color("33"),
		TitleBarFg:     lipgloss.Color("253"),
		StatusBarBg:    lipgloss.Color("246"),
		StatusBarFg:    lipgloss.Color("237"),

		TierGold:    lipgloss.Color("172"),
		TierGreen:   lipgloss.Color("28"),
		TierBlue:    lipgloss.Color("27"),
		TierCyan:    lipgloss.Color("30"),
		TierMagenta: lipgloss.Color("127"),
		TierDim:     lipgloss.Color("246"),

		IncludedGreen: lipgloss.Color("28"),
		ExcludedGray:  lipgloss.Color("246"),
		PartialYellow: lipgloss.Color("172"),
		CursorBg:      lipgloss.Color("254"),
		SecretRed:     lipgloss.Color("160"),
		TokenCountDim: lipgloss.Color("244"),
	}
}

// Styles holds all computed lipgloss styles for the TUI layout. It is
// immutable after construction; create a new Styles on terminal resize.
type Styles struct {
	// Theme colors used for this style set.
	Colors ThemeColors

	// TitleBar renders the top bar with inverse colors.
	TitleBar lipgloss.Style

	// StatusBar renders the bottom status line with inverse colors.
	StatusBar lipgloss.Style

	// FileTreePanel renders the left file-tree panel with border and padding.
	FileTreePanel lipgloss.Style

	// StatsPanel renders the right statistics panel with border and padding.
	StatsPanel lipgloss.Style

	// ActiveBorder renders a panel border when the panel is focused.
	ActiveBorder lipgloss.Style

	// InactiveBorder renders a panel border when the panel is not focused.
	InactiveBorder lipgloss.Style

	// PanelTitle renders panel title text inline with the border.
	PanelTitle lipgloss.Style

	// Separator renders the vertical divider between panels.
	Separator lipgloss.Style

	// TooSmall renders the "terminal too small" warning message.
	TooSmall lipgloss.Style

	// Computed layout dimensions.
	LeftPanelWidth  int
	RightPanelWidth int
	ContentHeight   int

	// Layout describes the current layout mode.
	Layout LayoutMode
}

// ComputeLayout determines the layout mode from the terminal dimensions.
func ComputeLayout(width, height int) LayoutMode {
	if width < MinTermWidth || height < MinTermHeight {
		return LayoutTooSmall
	}
	if width < 60 {
		return LayoutSinglePanel
	}
	if width < 100 {
		return LayoutCompressed
	}
	return LayoutFull
}

// NewStyles creates a complete Styles set adapted to the given theme and
// terminal dimensions. The isDark flag selects the color palette. The width
// and height determine panel sizing and layout mode.
func NewStyles(isDark bool, width, height int) Styles {
	colors := lightColors()
	if isDark {
		colors = darkColors()
	}

	layout := ComputeLayout(width, height)

	// Compute panel widths based on layout mode.
	var leftWidth, rightWidth int
	switch layout {
	case LayoutFull:
		leftWidth = width * 65 / 100
		rightWidth = width - leftWidth - 1 // 1 for separator
	case LayoutCompressed:
		leftWidth = width * 60 / 100
		rightWidth = width - leftWidth - 1
	case LayoutSinglePanel:
		leftWidth = width
		rightWidth = 0
	default:
		// LayoutTooSmall -- dimensions are meaningless but avoid zero.
		leftWidth = width
		rightWidth = 0
	}

	// ContentHeight accounts for the title bar (1 line) and status bar (1 line).
	contentHeight := height - 2
	if contentHeight < 0 {
		contentHeight = 0
	}

	border := lipgloss.RoundedBorder()

	titleBar := lipgloss.NewStyle().
		Background(colors.TitleBarBg).
		Foreground(colors.TitleBarFg).
		Bold(true).
		Width(width).
		Padding(0, 1).
		Align(lipgloss.Center)

	statusBar := lipgloss.NewStyle().
		Background(colors.StatusBarBg).
		Foreground(colors.StatusBarFg).
		Width(width).
		Padding(0, 1)

	activeBorder := lipgloss.NewStyle().
		Border(border).
		BorderForeground(colors.BorderActive).
		Padding(0, 1)

	inactiveBorder := lipgloss.NewStyle().
		Border(border).
		BorderForeground(colors.BorderInactive).
		Padding(0, 1)

	// File tree panel uses the active border by default. The caller
	// can swap between ActiveBorder and InactiveBorder as focus changes.
	fileTreePanel := activeBorder.
		Width(leftWidth).
		Height(contentHeight)

	statsPanel := inactiveBorder.
		Width(rightWidth).
		Height(contentHeight)

	panelTitle := lipgloss.NewStyle().
		Foreground(colors.Accent).
		Bold(true).
		Padding(0, 1)

	separator := lipgloss.NewStyle().
		Foreground(colors.BorderInactive).
		Width(1).
		Height(contentHeight)

	tooSmall := lipgloss.NewStyle().
		Foreground(colors.Accent).
		Bold(true).
		Width(width).
		Height(height).
		Align(lipgloss.Center).
		AlignVertical(lipgloss.Center)

	return Styles{
		Colors:          colors,
		TitleBar:        titleBar,
		StatusBar:       statusBar,
		FileTreePanel:   fileTreePanel,
		StatsPanel:      statsPanel,
		ActiveBorder:    activeBorder,
		InactiveBorder:  inactiveBorder,
		PanelTitle:      panelTitle,
		Separator:       separator,
		TooSmall:        tooSmall,
		LeftPanelWidth:  leftWidth,
		RightPanelWidth: rightWidth,
		ContentHeight:   contentHeight,
		Layout:          layout,
	}
}
