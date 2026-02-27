package stats

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harvx/harvx/internal/tui/filetree"
	"github.com/harvx/harvx/internal/tui/tuimsg"
)

func defaultOpts() Options {
	return Options{
		MaxTokens:     200000,
		ProfileName:   "default",
		TargetName:    "claude",
		TokenizerName: "cl100k_base",
		Width:         35,
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	m := New(defaultOpts())

	assert.Equal(t, 200000, m.maxTokens)
	assert.Equal(t, "default", m.profileName)
	assert.Equal(t, "claude", m.targetName)
	assert.Equal(t, "cl100k_base", m.tokenizerName)
	assert.Equal(t, 35, m.width)
	assert.False(t, m.calculating)
	assert.Equal(t, 0, m.totalTokens)
}

func TestNew_DefaultWidth(t *testing.T) {
	t.Parallel()

	m := New(Options{MaxTokens: 100000})
	assert.Equal(t, DefaultWidth, m.width)
}

func TestModel_Init(t *testing.T) {
	t.Parallel()

	m := New(defaultOpts())
	cmd := m.Init()
	assert.Nil(t, cmd, "Init should return nil cmd")
}

func TestModel_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ tea.Model = Model{}
}

func TestModel_FileToggledMsg_IncrementsGeneration(t *testing.T) {
	t.Parallel()

	m := New(defaultOpts())
	assert.Equal(t, uint64(0), m.generation)
	assert.False(t, m.calculating)

	// First toggle.
	updated, cmd := m.Update(tuimsg.FileToggledMsg{Path: "main.go", Included: true})
	m1 := updated.(Model)

	assert.Equal(t, uint64(1), m1.generation)
	assert.True(t, m1.calculating)
	assert.NotNil(t, cmd, "should return debounce tick cmd")

	// Second rapid toggle.
	updated, cmd = m1.Update(tuimsg.FileToggledMsg{Path: "util.go", Included: true})
	m2 := updated.(Model)

	assert.Equal(t, uint64(2), m2.generation)
	assert.True(t, m2.calculating)
	assert.NotNil(t, cmd)
}

func TestModel_DebounceGeneration_StaleTickIgnored(t *testing.T) {
	t.Parallel()

	m := New(defaultOpts())

	// Simulate two rapid toggles.
	updated, _ := m.Update(tuimsg.FileToggledMsg{Path: "a.go", Included: true})
	m1 := updated.(Model)
	assert.Equal(t, uint64(1), m1.generation)

	updated, _ = m1.Update(tuimsg.FileToggledMsg{Path: "b.go", Included: true})
	m2 := updated.(Model)
	assert.Equal(t, uint64(2), m2.generation)

	// Stale tick from first toggle arrives.
	updated, cmd := m2.Update(recalcTickMsg{generation: 1})
	m3 := updated.(Model)

	// Should be ignored (stale).
	assert.Nil(t, cmd, "stale tick should not trigger recalc")
	assert.True(t, m3.calculating, "should still be calculating")
}

func TestModel_DebounceGeneration_MatchingTickTriggersRecalc(t *testing.T) {
	t.Parallel()

	m := New(defaultOpts())
	root := filetree.NewNode("", "root", true)
	m.treeRoot = root

	// Simulate toggle.
	updated, _ := m.Update(tuimsg.FileToggledMsg{Path: "a.go", Included: true})
	m1 := updated.(Model)

	// Matching tick arrives.
	updated, cmd := m1.Update(recalcTickMsg{generation: 1})
	_ = updated.(Model)

	assert.NotNil(t, cmd, "matching tick should trigger calculateTokens cmd")
}

func TestModel_TokenCountResult(t *testing.T) {
	t.Parallel()

	m := New(defaultOpts())
	m.calculating = true

	result := tokenCountResult{
		TotalTokens:   89420,
		SelectedFiles: 42,
		TotalFiles:    390,
		TierBreakdown: map[int]int{0: 5, 1: 20, 2: 17},
		TierTokens:    map[int]int{0: 12400, 1: 34200, 2: 42820},
		SecretsFound:  3,
	}

	updated, cmd := m.Update(result)
	m1 := updated.(Model)

	assert.Equal(t, 89420, m1.totalTokens)
	assert.Equal(t, 42, m1.selectedFiles)
	assert.Equal(t, 390, m1.totalFiles)
	assert.Equal(t, 3, m1.secretsFound)
	assert.False(t, m1.calculating, "should no longer be calculating")
	assert.Nil(t, cmd)

	// Budget should be computed: 89420/200000 * 100 = 44.71
	assert.InDelta(t, 44.71, m1.budgetUsed, 0.01)

	// Tier breakdown.
	assert.Equal(t, 5, m1.tierBreakdown[0])
	assert.Equal(t, 20, m1.tierBreakdown[1])
	assert.Equal(t, 17, m1.tierBreakdown[2])
}

func TestModel_TokenCountResult_ZeroBudget(t *testing.T) {
	t.Parallel()

	m := New(Options{MaxTokens: 0})
	m.calculating = true

	result := tokenCountResult{
		TotalTokens:   1000,
		SelectedFiles: 5,
		TotalFiles:    10,
		TierBreakdown: make(map[int]int),
		TierTokens:    make(map[int]int),
	}

	updated, _ := m.Update(result)
	m1 := updated.(Model)

	assert.Equal(t, 0.0, m1.budgetUsed, "zero budget should result in 0% used")
}

func TestModel_ProfileChangedMsg(t *testing.T) {
	t.Parallel()

	m := New(defaultOpts())
	updated, cmd := m.Update(tuimsg.ProfileChangedMsg{ProfileName: "finvault"})
	m1 := updated.(Model)

	assert.Equal(t, "finvault", m1.profileName)
	assert.Nil(t, cmd)
}

func TestModel_TokenCountUpdatedMsg(t *testing.T) {
	t.Parallel()

	m := New(defaultOpts())
	m.calculating = true

	updated, cmd := m.Update(tuimsg.TokenCountUpdatedMsg{
		TotalTokens: 5000,
		FileCount:   42,
		BudgetUsed:  75.5,
	})
	m1 := updated.(Model)

	assert.Equal(t, 5000, m1.totalTokens)
	assert.Equal(t, 42, m1.selectedFiles)
	assert.InDelta(t, 75.5, m1.budgetUsed, 0.001)
	assert.False(t, m1.calculating)
	assert.Nil(t, cmd)
}

func TestModel_WindowSizeMsg(t *testing.T) {
	t.Parallel()

	m := New(defaultOpts())
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 40, Height: 30})
	m1 := updated.(Model)

	assert.Equal(t, 40, m1.width)
	assert.Equal(t, 30, m1.height)
	assert.Nil(t, cmd)
}

func TestModel_Accessors(t *testing.T) {
	t.Parallel()

	m := New(Options{
		MaxTokens: 100000,
		Width:     40,
	})
	m.totalTokens = 5000
	m.selectedFiles = 10
	m.budgetUsed = 5.0
	m.calculating = true
	m.height = 30

	assert.Equal(t, 5000, m.TotalTokens())
	assert.Equal(t, 10, m.SelectedFiles())
	assert.InDelta(t, 5.0, m.BudgetUsed(), 0.001)
	assert.True(t, m.Calculating())
	assert.Equal(t, 40, m.Width())
	assert.Equal(t, 30, m.Height())
}

func TestModel_SetTreeRoot(t *testing.T) {
	t.Parallel()

	m := New(defaultOpts())
	root := filetree.NewNode("", "root", true)
	m.SetTreeRoot(root)

	assert.Equal(t, root, m.treeRoot)
}

func TestModel_View_ContainsKeyInfo(t *testing.T) {
	t.Parallel()

	m := New(Options{
		MaxTokens:     200000,
		ProfileName:   "finvault",
		TargetName:    "claude",
		TokenizerName: "o200k_base",
		Width:         40,
	})
	m.totalTokens = 89420
	m.selectedFiles = 42
	m.totalFiles = 390
	m.secretsFound = 3
	m.height = 30

	view := m.View()

	assert.Contains(t, view, "Stats")
	assert.Contains(t, view, "finvault")
	assert.Contains(t, view, "claude")
	assert.Contains(t, view, "o200k_base")
	assert.Contains(t, view, "89,420")
	assert.Contains(t, view, "200,000")
	assert.Contains(t, view, "42")
	assert.Contains(t, view, "390")
	assert.Contains(t, view, "3 found")
}

func TestModel_View_CalculatingIndicator(t *testing.T) {
	t.Parallel()

	m := New(defaultOpts())
	m.calculating = true
	m.width = 40
	m.height = 30

	view := m.View()
	assert.Contains(t, view, "calculating...")
}

func TestModel_View_CompressionShown(t *testing.T) {
	t.Parallel()

	m := New(Options{
		MaxTokens:   200000,
		Compression: true,
		Width:       40,
	})
	m.compressionPct = 52.0
	m.height = 30

	view := m.View()
	assert.Contains(t, view, "Compressed")
	assert.Contains(t, view, "52%")
}

func TestModel_View_CompressionHiddenWhenDisabled(t *testing.T) {
	t.Parallel()

	m := New(Options{
		MaxTokens:   200000,
		Compression: false,
		Width:       40,
	})
	m.height = 30

	view := m.View()
	assert.NotContains(t, view, "Compressed")
}

// --- calculateTokens integration test ---

func TestCalculateTokens_WithTree(t *testing.T) {
	t.Parallel()

	root := buildTestTree()

	cmd := calculateTokens(root)
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(tokenCountResult)
	require.True(t, ok)

	// 3 included files: main.go (100), util.go (50), readme.md (20)
	assert.Equal(t, 170, result.TotalTokens)
	assert.Equal(t, 3, result.SelectedFiles)
	assert.Equal(t, 4, result.TotalFiles) // 4 total files (including excluded)
	assert.Equal(t, 1, result.SecretsFound)

	// Tier breakdown.
	assert.Equal(t, 2, result.TierBreakdown[1]) // main.go + util.go
	assert.Equal(t, 1, result.TierBreakdown[4]) // readme.md
	assert.Equal(t, 150, result.TierTokens[1])
	assert.Equal(t, 20, result.TierTokens[4])
}

func TestCalculateTokens_NilRoot(t *testing.T) {
	t.Parallel()

	cmd := calculateTokens(nil)
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(tokenCountResult)
	require.True(t, ok)

	assert.Equal(t, 0, result.TotalTokens)
	assert.Equal(t, 0, result.SelectedFiles)
	assert.Equal(t, 0, result.TotalFiles)
}

func TestCalculateTokens_EmptyTree(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)

	cmd := calculateTokens(root)
	msg := cmd()
	result := msg.(tokenCountResult)

	assert.Equal(t, 0, result.TotalTokens)
	assert.Equal(t, 0, result.SelectedFiles)
}

// --- Debounce full flow tests ---

func TestModel_DebounceFullFlow_RapidTogglesOnlyOneRecalc(t *testing.T) {
	t.Parallel()

	// Build a tree so calculateTokens has something to work with.
	root := buildTestTree()
	m := New(defaultOpts())
	m.treeRoot = root

	// Simulate 5 rapid file toggles.
	var updated tea.Model
	updated = m
	for i := 0; i < 5; i++ {
		var cmd tea.Cmd
		updated, cmd = updated.(Model).Update(tuimsg.FileToggledMsg{
			Path:     "file.go",
			Included: true,
		})
		assert.NotNil(t, cmd, "each toggle should produce a debounce tick cmd")
	}

	m5 := updated.(Model)
	assert.Equal(t, uint64(5), m5.generation)
	assert.True(t, m5.calculating)

	// Stale ticks from generations 1-4 should be ignored.
	for gen := uint64(1); gen <= 4; gen++ {
		updated, cmd := m5.Update(recalcTickMsg{generation: gen})
		mIgnored := updated.(Model)
		assert.Nil(t, cmd, "stale tick gen %d should be ignored", gen)
		assert.True(t, mIgnored.calculating, "should still be calculating after stale tick gen %d", gen)
	}

	// The matching tick (gen 5) should trigger recalculation.
	updated, cmd := m5.Update(recalcTickMsg{generation: 5})
	_ = updated.(Model)
	assert.NotNil(t, cmd, "matching tick gen 5 should trigger calculateTokens")
}

func TestModel_TokenCountResult_BudgetThresholds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		maxTokens int
		tokens    int
		wantPct   float64
	}{
		{
			name:      "exactly 70 percent",
			maxTokens: 100000,
			tokens:    70000,
			wantPct:   70.0,
		},
		{
			name:      "exactly 90 percent",
			maxTokens: 100000,
			tokens:    90000,
			wantPct:   90.0,
		},
		{
			name:      "over 100 percent",
			maxTokens: 100000,
			tokens:    150000,
			wantPct:   150.0,
		},
		{
			name:      "zero tokens",
			maxTokens: 100000,
			tokens:    0,
			wantPct:   0.0,
		},
		{
			name:      "just under 70 percent",
			maxTokens: 100000,
			tokens:    69999,
			wantPct:   69.999,
		},
		{
			name:      "just over 90 percent",
			maxTokens: 100000,
			tokens:    90001,
			wantPct:   90.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := New(Options{MaxTokens: tt.maxTokens})
			m.calculating = true

			result := tokenCountResult{
				TotalTokens:   tt.tokens,
				SelectedFiles: 10,
				TotalFiles:    20,
				TierBreakdown: make(map[int]int),
				TierTokens:    make(map[int]int),
			}

			updated, _ := m.Update(result)
			m1 := updated.(Model)

			assert.InDelta(t, tt.wantPct, m1.budgetUsed, 0.01)
		})
	}
}

func TestModel_SetSize(t *testing.T) {
	t.Parallel()

	m := New(defaultOpts())
	assert.Equal(t, 35, m.width)
	assert.Equal(t, 0, m.height)

	m.SetSize(50, 40)
	assert.Equal(t, 50, m.width)
	assert.Equal(t, 40, m.height)
}

func TestModel_SetCompressionPct(t *testing.T) {
	t.Parallel()

	m := New(Options{
		Compression: true,
		Width:       40,
	})
	assert.Equal(t, 0.0, m.compressionPct)

	m.SetCompressionPct(52.3)
	assert.InDelta(t, 52.3, m.compressionPct, 0.001)
}

func TestModel_Update_UnknownMsgType(t *testing.T) {
	t.Parallel()

	m := New(defaultOpts())
	m.totalTokens = 42

	// Send an unrecognized message type.
	type unknownMsg struct{}
	updated, cmd := m.Update(unknownMsg{})
	m1 := updated.(Model)

	// Model should be unchanged.
	assert.Equal(t, 42, m1.totalTokens)
	assert.Nil(t, cmd)
}

func TestModel_TokenCountResult_TierTokensSumCorrectly(t *testing.T) {
	t.Parallel()

	m := New(Options{MaxTokens: 200000})
	m.calculating = true

	result := tokenCountResult{
		TotalTokens:   75400,
		SelectedFiles: 233,
		TotalFiles:    390,
		TierBreakdown: map[int]int{0: 5, 1: 48, 2: 180},
		TierTokens:    map[int]int{0: 12400, 1: 34200, 2: 28800},
	}

	updated, _ := m.Update(result)
	m1 := updated.(Model)

	// Verify tier tokens sum to total.
	tierSum := 0
	for _, tok := range m1.tierTokens {
		tierSum += tok
	}
	assert.Equal(t, m1.totalTokens, tierSum, "tier token sum should equal totalTokens")

	// Verify tier file count sum.
	fileSum := 0
	for _, count := range m1.tierBreakdown {
		fileSum += count
	}
	assert.Equal(t, m1.selectedFiles, fileSum, "tier file count sum should equal selectedFiles")
}

func TestModel_View_ZeroWidth(t *testing.T) {
	t.Parallel()

	m := New(Options{
		MaxTokens:   200000,
		ProfileName: "test",
		Width:       0,
	})
	m.height = 30
	m.totalTokens = 100

	// Should not panic with zero width; should use DefaultWidth.
	view := m.View()
	assert.NotEmpty(t, view)
}

func TestModel_New_NegativeWidth(t *testing.T) {
	t.Parallel()

	m := New(Options{
		MaxTokens: 100000,
		Width:     -5,
	})
	assert.Equal(t, DefaultWidth, m.width, "negative width should fall back to DefaultWidth")
}

func TestModel_View_AllFilesExcluded(t *testing.T) {
	t.Parallel()

	m := New(Options{
		MaxTokens: 200000,
		Width:     40,
	})
	m.totalTokens = 0
	m.selectedFiles = 0
	m.totalFiles = 100
	m.height = 30

	view := m.View()
	assert.Contains(t, view, "0")
	assert.Contains(t, view, "100")
}

func TestModel_View_CompressionSavingsPercentage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pct     float64
		wantStr string
	}{
		{name: "zero compression", pct: 0, wantStr: "0% reduction"},
		{name: "half compression", pct: 50, wantStr: "50% reduction"},
		{name: "full compression", pct: 100, wantStr: "100% reduction"},
		{name: "high compression", pct: 78, wantStr: "78% reduction"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := New(Options{
				MaxTokens:   200000,
				Compression: true,
				Width:       40,
			})
			m.compressionPct = tt.pct
			m.height = 30

			view := m.View()
			assert.Contains(t, view, tt.wantStr)
		})
	}
}

// buildTestTree creates a small test file tree:
//
//	root/
//	  src/
//	    main.go (included, tier 1, 100 tokens)
//	    util.go (included, tier 1, 50 tokens, has secrets)
//	  docs/
//	    readme.md (included, tier 4, 20 tokens)
//	  .gitignore (excluded, tier 5, 10 tokens)
func buildTestTree() *filetree.Node {
	root := filetree.NewNode("", "root", true)
	root.SetLoaded(true)

	src := filetree.NewNode("src", "src", true)
	src.SetLoaded(true)
	root.AddChild(src)

	mainGo := filetree.NewNode("src/main.go", "main.go", false)
	mainGo.Included = filetree.Included
	mainGo.Tier = 1
	mainGo.TokenCount = 100
	src.AddChild(mainGo)

	utilGo := filetree.NewNode("src/util.go", "util.go", false)
	utilGo.Included = filetree.Included
	utilGo.Tier = 1
	utilGo.TokenCount = 50
	utilGo.HasSecrets = true
	src.AddChild(utilGo)

	docs := filetree.NewNode("docs", "docs", true)
	docs.SetLoaded(true)
	root.AddChild(docs)

	readme := filetree.NewNode("docs/readme.md", "readme.md", false)
	readme.Included = filetree.Included
	readme.Tier = 4
	readme.TokenCount = 20
	docs.AddChild(readme)

	gitignore := filetree.NewNode(".gitignore", ".gitignore", false)
	gitignore.Included = filetree.Excluded
	gitignore.Tier = 5
	gitignore.TokenCount = 10
	root.AddChild(gitignore)

	return root
}
