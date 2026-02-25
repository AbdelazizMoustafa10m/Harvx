package stats

import (
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/harvx/harvx/internal/tui/filetree"
	"github.com/harvx/harvx/internal/tui/tuimsg"
)

// DefaultWidth is the default panel width in characters.
const DefaultWidth = 35

// Compile-time interface compliance check.
var _ tea.Model = Model{}

// Model is the stats sidebar panel Bubble Tea model. It displays live token
// counts, a visual budget bar, file counts, size estimates, compression
// savings, redaction count, tier breakdown, and profile/tokenizer information.
//
// The model debounces token recalculation after file toggles to prevent UI
// jank from rapid changes.
type Model struct {
	// Stats state.
	totalTokens   int
	selectedFiles int
	totalFiles    int
	budgetUsed    float64
	tierBreakdown map[int]int // tier -> file count
	tierTokens    map[int]int // tier -> total tokens
	secretsFound  int

	// Configuration.
	maxTokens      int
	profileName    string
	targetName     string
	tokenizerName  string
	compression    bool
	compressionPct float64 // reduction percentage

	// Layout.
	width  int
	height int

	// Debounce state.
	generation   uint64 // incremented on each file toggle
	calculating  bool   // true while a token count is in progress

	// File tree reference for recalculation.
	treeRoot *filetree.Node

	logger *slog.Logger
}

// Options configures the stats panel model.
type Options struct {
	// MaxTokens is the token budget cap from the profile.
	MaxTokens int

	// ProfileName is the name of the active profile.
	ProfileName string

	// TargetName is the LLM target (e.g., "claude", "chatgpt").
	TargetName string

	// TokenizerName is the name of the tokenizer encoding (e.g., "cl100k_base").
	TokenizerName string

	// Compression indicates whether compression is enabled in the profile.
	Compression bool

	// Width is the initial panel width. Uses DefaultWidth if zero.
	Width int
}

// New creates a new stats panel Model with the given options.
func New(opts Options) Model {
	width := opts.Width
	if width <= 0 {
		width = DefaultWidth
	}

	return Model{
		maxTokens:     opts.MaxTokens,
		profileName:   opts.ProfileName,
		targetName:    opts.TargetName,
		tokenizerName: opts.TokenizerName,
		compression:   opts.Compression,
		width:         width,
		tierBreakdown: make(map[int]int, 6),
		tierTokens:    make(map[int]int, 6),
		logger:        slog.Default().With("component", "stats"),
	}
}

// Init implements tea.Model. The stats panel has no initialization command.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. It handles file toggle messages (with
// debouncing), profile changes, token count results, and size updates.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tuimsg.FileToggledMsg:
		return m.handleFileToggled(msg)

	case tuimsg.ProfileChangedMsg:
		return m.handleProfileChanged(msg)

	case tuimsg.TokenCountUpdatedMsg:
		return m.handleTokenCountUpdated(msg)

	case recalcTickMsg:
		return m.handleRecalcTick(msg)

	case tokenCountResult:
		return m.handleTokenCountResult(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	return m, nil
}

// SetTreeRoot sets the file tree root node for token recalculation.
func (m *Model) SetTreeRoot(root *filetree.Node) {
	m.treeRoot = root
}

// SetSize updates the panel dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetCompressionPct sets the compression reduction percentage.
func (m *Model) SetCompressionPct(pct float64) {
	m.compressionPct = pct
}

// TotalTokens returns the current total token count.
func (m Model) TotalTokens() int {
	return m.totalTokens
}

// SelectedFiles returns the number of currently selected files.
func (m Model) SelectedFiles() int {
	return m.selectedFiles
}

// BudgetUsed returns the current budget utilization percentage (0-100).
func (m Model) BudgetUsed() float64 {
	return m.budgetUsed
}

// Calculating returns true if token recalculation is in progress.
func (m Model) Calculating() bool {
	return m.calculating
}

// Width returns the panel width.
func (m Model) Width() int {
	return m.width
}

// Height returns the panel height.
func (m Model) Height() int {
	return m.height
}

// handleFileToggled increments the debounce generation counter and schedules
// a debounced recalculation tick.
func (m Model) handleFileToggled(_ tuimsg.FileToggledMsg) (tea.Model, tea.Cmd) {
	m.generation++
	m.calculating = true

	m.logger.Debug("file toggled, scheduling recalc",
		"generation", m.generation,
	)

	return m, scheduleDebounce(m.generation)
}

// handleProfileChanged updates the profile name displayed in the panel.
func (m Model) handleProfileChanged(msg tuimsg.ProfileChangedMsg) (tea.Model, tea.Cmd) {
	m.profileName = msg.ProfileName

	m.logger.Debug("profile changed",
		"profile", msg.ProfileName,
	)

	return m, nil
}

// handleTokenCountUpdated handles an external token count update (e.g., from
// the headless pipeline). This provides an alternative to tree-based recalc.
func (m Model) handleTokenCountUpdated(msg tuimsg.TokenCountUpdatedMsg) (tea.Model, tea.Cmd) {
	m.totalTokens = msg.TotalTokens
	m.selectedFiles = msg.FileCount
	m.budgetUsed = msg.BudgetUsed
	m.calculating = false

	return m, nil
}

// handleRecalcTick fires when the debounce timer expires. If the generation
// counter matches the tick's generation, the recalculation is triggered. If
// another toggle arrived since the tick was scheduled, this tick is stale and
// discarded.
func (m Model) handleRecalcTick(msg recalcTickMsg) (tea.Model, tea.Cmd) {
	if msg.generation != m.generation {
		// Stale tick: a newer toggle superseded this one.
		m.logger.Debug("stale recalc tick, ignoring",
			"tickGen", msg.generation,
			"currentGen", m.generation,
		)
		return m, nil
	}

	m.logger.Debug("recalc tick matched, calculating",
		"generation", m.generation,
	)

	return m, calculateTokens(m.treeRoot)
}

// handleTokenCountResult processes the result of an async token count
// calculation, updating all display statistics.
func (m Model) handleTokenCountResult(msg tokenCountResult) (tea.Model, tea.Cmd) {
	m.totalTokens = msg.TotalTokens
	m.selectedFiles = msg.SelectedFiles
	m.totalFiles = msg.TotalFiles
	m.tierBreakdown = msg.TierBreakdown
	m.tierTokens = msg.TierTokens
	m.secretsFound = msg.SecretsFound
	m.calculating = false

	// Calculate budget utilization.
	if m.maxTokens > 0 {
		m.budgetUsed = float64(m.totalTokens) / float64(m.maxTokens) * 100
	} else {
		m.budgetUsed = 0
	}

	m.logger.Debug("token count updated",
		"tokens", m.totalTokens,
		"files", m.selectedFiles,
		"budget", m.budgetUsed,
	)

	return m, nil
}
