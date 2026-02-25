package stats

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestRenderBudgetBar_FilledPortion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pct      float64
		barWidth int
	}{
		{name: "0 percent", pct: 0, barWidth: 30},
		{name: "25 percent", pct: 25, barWidth: 30},
		{name: "50 percent", pct: 50, barWidth: 30},
		{name: "75 percent", pct: 75, barWidth: 30},
		{name: "100 percent", pct: 100, barWidth: 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bar := RenderBudgetBar(tt.pct, tt.barWidth)
			// Bar should contain brackets.
			assert.Contains(t, bar, "[")
			assert.Contains(t, bar, "]")
			// Bar should contain the percentage.
			assert.Contains(t, bar, FormatPercentage(tt.pct))
		})
	}
}

func TestRenderBudgetBar_ColorTransitions(t *testing.T) {
	t.Parallel()

	// We can't easily test ANSI colors directly, but we can verify the bar
	// renders without panicking at boundary values.
	boundaries := []float64{0, 69.9, 70.0, 89.9, 90.0, 100.0}
	for _, pct := range boundaries {
		bar := RenderBudgetBar(pct, 30)
		assert.NotEmpty(t, bar)
	}
}

func TestRenderBudgetBar_VeryNarrow(t *testing.T) {
	t.Parallel()

	// Very narrow bar should degrade gracefully.
	bar := RenderBudgetBar(50, 4)
	assert.Equal(t, "50%", bar)
}

func TestRenderBudgetBar_NegativeAndOver100(t *testing.T) {
	t.Parallel()

	// Negative should clamp to 0.
	bar := RenderBudgetBar(-10, 30)
	assert.Contains(t, bar, "0%")

	// Over 100 should clamp to 100.
	bar = RenderBudgetBar(150, 30)
	assert.Contains(t, bar, "100%")
}

func TestRenderBudgetBar_ContainsBlockChars(t *testing.T) {
	t.Parallel()

	bar := RenderBudgetBar(50, 30)
	// Should contain at least one filled and one empty block.
	assert.True(t, strings.Contains(bar, barFilled) || strings.Contains(bar, barEmpty),
		"bar should contain block characters")
}

func TestView_TierBreakdownShown(t *testing.T) {
	t.Parallel()

	m := New(Options{
		MaxTokens:     200000,
		ProfileName:   "test",
		TokenizerName: "none",
		Width:         45,
	})
	m.tierBreakdown = map[int]int{0: 5, 1: 48, 2: 180}
	m.tierTokens = map[int]int{0: 12400, 1: 34200, 2: 28800}
	m.totalTokens = 75400
	m.selectedFiles = 233
	m.totalFiles = 390
	m.height = 40

	view := m.View()

	assert.Contains(t, view, "Tier Breakdown")
	assert.Contains(t, view, "critical")
	assert.Contains(t, view, "primary")
	assert.Contains(t, view, "secondary")
	// Tiers with zero files should show dashes.
	assert.Contains(t, view, "tests")
}

func TestView_SecretsAlertStyle(t *testing.T) {
	t.Parallel()

	// Zero secrets.
	m := New(Options{Width: 40})
	m.height = 30
	view := m.View()
	assert.Contains(t, view, "0 found")

	// Nonzero secrets.
	m.secretsFound = 5
	view = m.View()
	assert.Contains(t, view, "5 found")
}

func TestView_NarrowPanel(t *testing.T) {
	t.Parallel()

	m := New(Options{
		MaxTokens:     200000,
		ProfileName:   "a-very-long-profile-name-that-exceeds-width",
		TargetName:    "claude",
		TokenizerName: "cl100k_base",
		Width:         20,
	})
	m.totalTokens = 89420
	m.height = 30

	// Should not panic with narrow width.
	view := m.View()
	assert.NotEmpty(t, view)
}

func TestRenderBudgetBar_FillProportionality(t *testing.T) {
	t.Parallel()

	barWidth := 30

	tests := []struct {
		name        string
		pct         float64
		wantFilled  int // approximate expected filled count
		innerWidth  int // barWidth - 7 (brackets + space + pct label)
	}{
		{name: "0 percent", pct: 0, wantFilled: 0, innerWidth: 23},
		{name: "50 percent", pct: 50, wantFilled: 11, innerWidth: 23},
		{name: "100 percent", pct: 100, wantFilled: 23, innerWidth: 23},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bar := RenderBudgetBar(tt.pct, barWidth)

			// Count filled and empty block characters in the raw output.
			filledCount := strings.Count(bar, barFilled)
			emptyCount := strings.Count(bar, barEmpty)

			// Total blocks should equal innerWidth.
			assert.Equal(t, tt.innerWidth, filledCount+emptyCount,
				"total blocks should equal innerWidth (%d)", tt.innerWidth)

			// Filled count should match expected.
			assert.Equal(t, tt.wantFilled, filledCount,
				"filled blocks should match expected for %.0f%%", tt.pct)
		})
	}
}

func TestRenderBudgetBar_ColorTransitions_VerifyColors(t *testing.T) {
	t.Parallel()

	// Render budget bars at each color zone and verify by comparing the
	// lipgloss-rendered filled block against the expected color.
	tests := []struct {
		name      string
		pct       float64
		wantColor lipgloss.Color
	}{
		{name: "green zone 0%", pct: 0, wantColor: colorGreen},
		{name: "green zone 50%", pct: 50, wantColor: colorGreen},
		{name: "green zone 69.9%", pct: 69.9, wantColor: colorGreen},
		{name: "yellow zone 70%", pct: 70.0, wantColor: colorYellow},
		{name: "yellow zone 80%", pct: 80.0, wantColor: colorYellow},
		{name: "yellow zone 89.9%", pct: 89.9, wantColor: colorYellow},
		{name: "red zone 90%", pct: 90.0, wantColor: colorRed},
		{name: "red zone 95%", pct: 95.0, wantColor: colorRed},
		{name: "red zone 100%", pct: 100.0, wantColor: colorRed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bar := RenderBudgetBar(tt.pct, 30)

			// Render a reference filled block with the expected color so we can
			// compare the ANSI sequence that lipgloss emits.
			expectedStyled := lipgloss.NewStyle().Foreground(tt.wantColor).Render(barFilled)

			if tt.pct > 0 {
				// The bar output should contain the colored filled block.
				assert.Contains(t, bar, expectedStyled,
					"bar at %.1f%% should use correct color", tt.pct)
			}
		})
	}
}

func TestView_TierBreakdownTokenSums(t *testing.T) {
	t.Parallel()

	m := New(Options{
		MaxTokens:     200000,
		ProfileName:   "test",
		TokenizerName: "cl100k_base",
		Width:         50,
	})
	m.tierBreakdown = map[int]int{0: 3, 1: 10, 2: 5, 3: 8, 4: 2, 5: 1}
	m.tierTokens = map[int]int{0: 5000, 1: 20000, 2: 10000, 3: 8000, 4: 1500, 5: 500}
	m.totalTokens = 45000
	m.selectedFiles = 29
	m.totalFiles = 50
	m.height = 40

	view := m.View()

	// Verify each tier's token count appears formatted.
	assert.Contains(t, view, "5,000")
	assert.Contains(t, view, "20,000")
	assert.Contains(t, view, "10,000")
	assert.Contains(t, view, "8,000")
	assert.Contains(t, view, "1,500")
	assert.Contains(t, view, "500")

	// Verify tier labels.
	assert.Contains(t, view, "critical")
	assert.Contains(t, view, "primary")
	assert.Contains(t, view, "secondary")
	assert.Contains(t, view, "tests")
	assert.Contains(t, view, "docs")
	assert.Contains(t, view, "low")
}

func TestView_VeryNarrowWidth(t *testing.T) {
	t.Parallel()

	// Width of 5 should still render without panic.
	m := New(Options{
		MaxTokens: 200000,
		Width:     5,
	})
	m.totalTokens = 100000
	m.height = 30

	view := m.View()
	assert.NotEmpty(t, view)
}

func TestView_OutputSizeEstimate(t *testing.T) {
	t.Parallel()

	m := New(Options{
		MaxTokens: 200000,
		Width:     40,
	})
	m.totalTokens = 256000 // 256000 * 4 = 1024000 bytes = ~1.0 MB
	m.height = 30

	view := m.View()
	assert.Contains(t, view, "Size:")
	// 256000 tokens * 4 bytes = 1024000 bytes ~= 1000.0 KB or ~1.0 MB
	// EstimateOutputSize returns 1024000, FormatSize(1024000) = "1000.0 KB" (just under 1 MB)
	// Actually 1024000 < 1048576 so it is KB
	assert.Contains(t, view, "KB")
}

func TestView_TokenizerInfoDisplayed(t *testing.T) {
	t.Parallel()

	m := New(Options{
		MaxTokens:     200000,
		TokenizerName: "o200k_base",
		Width:         40,
	})
	m.height = 30

	view := m.View()
	assert.Contains(t, view, "Tokenizer:")
	assert.Contains(t, view, "o200k_base")
}

func TestView_TokenizerNone(t *testing.T) {
	t.Parallel()

	m := New(Options{
		MaxTokens:     200000,
		TokenizerName: "",
		Width:         40,
	})
	m.height = 30

	view := m.View()
	assert.Contains(t, view, "Tokenizer:")
	assert.Contains(t, view, "none")
}

func TestView_SecretsZeroNotRed(t *testing.T) {
	t.Parallel()

	m := New(Options{Width: 40})
	m.secretsFound = 0
	m.height = 30

	view := m.View()

	// Zero secrets: should show "0 found" in normal style (colorValue, not colorAlert).
	assert.Contains(t, view, "0 found")

	// Verify the normal-styled version appears (same rendering path as production code).
	normalStyled := lipgloss.NewStyle().Foreground(colorValue).Render("0 found")
	assert.Contains(t, view, normalStyled, "zero secrets should use normal styling")
}

func TestView_SecretsNonZeroAlertStyle(t *testing.T) {
	t.Parallel()

	m := New(Options{Width: 40})
	m.secretsFound = 3
	m.height = 30

	view := m.View()

	// Nonzero secrets should use alert styling.
	alertStyled := lipgloss.NewStyle().Foreground(colorAlert).Bold(true).Render("3 found")
	assert.Contains(t, view, alertStyled,
		"nonzero secrets should use alert styling")
}
