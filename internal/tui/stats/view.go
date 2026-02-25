package stats

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/harvx/harvx/internal/relevance"
)

// Block characters for the budget bar.
const (
	barFilled = "█"
	barEmpty  = "░"
)

// Color thresholds for the budget bar utilization.
const (
	thresholdYellow = 70.0
	thresholdRed    = 90.0
)

// Budget bar colors.
var (
	colorGreen  = lipgloss.Color("42")  // green for < 70%
	colorYellow = lipgloss.Color("226") // yellow for 70-90%
	colorRed    = lipgloss.Color("196") // red for > 90%
	colorDim    = lipgloss.Color("240") // dim gray for empty portion
	colorAlert  = lipgloss.Color("196") // red for alerts (secrets)
	colorLabel  = lipgloss.Color("252") // light gray for labels
	colorValue  = lipgloss.Color("255") // white for values
	colorHeader = lipgloss.Color("117") // light blue for headers
)

// View implements tea.Model. It renders the stats panel as a fixed-width
// sidebar with token counts, budget bar, file stats, and tier breakdown.
func (m Model) View() string {
	w := m.width
	if w <= 0 {
		w = DefaultWidth
	}

	style := lipgloss.NewStyle().
		Width(w).
		Padding(0, 1)

	var b strings.Builder

	// Header: profile info.
	m.renderHeader(&b, w)

	// Token count and budget bar.
	m.renderTokenBudget(&b, w)

	// File count.
	m.renderFileCount(&b, w)

	// Estimated output size.
	m.renderOutputSize(&b, w)

	// Compression savings (if enabled).
	if m.compression {
		m.renderCompression(&b, w)
	}

	// Redaction/secrets count.
	m.renderSecrets(&b, w)

	// Tier breakdown.
	m.renderTierBreakdown(&b, w)

	// Tokenizer info.
	m.renderTokenizerInfo(&b, w)

	return style.Render(b.String())
}

// renderHeader writes the profile and target info header.
func (m Model) renderHeader(b *strings.Builder, width int) {
	headerStyle := lipgloss.NewStyle().
		Foreground(colorHeader).
		Bold(true)

	b.WriteString(headerStyle.Render("Stats"))
	b.WriteByte('\n')

	// Separator line.
	sepWidth := width - 2
	if sepWidth < 0 {
		sepWidth = 0
	}
	sep := strings.Repeat("─", sepWidth)
	b.WriteString(lipgloss.NewStyle().Foreground(colorDim).Render(sep))
	b.WriteByte('\n')

	// Profile line.
	label := lipgloss.NewStyle().Foreground(colorLabel)
	value := lipgloss.NewStyle().Foreground(colorValue)

	b.WriteString(label.Render("Profile: ") + value.Render(m.profileName))
	if m.targetName != "" {
		b.WriteString(label.Render(" | Target: ") + value.Render(m.targetName))
	}
	b.WriteByte('\n')
	b.WriteByte('\n')
}

// renderTokenBudget writes the token count and budget utilization bar.
func (m Model) renderTokenBudget(b *strings.Builder, width int) {
	label := lipgloss.NewStyle().Foreground(colorLabel)
	value := lipgloss.NewStyle().Foreground(colorValue)

	// Calculating indicator.
	if m.calculating {
		b.WriteString(label.Render("Tokens: "))
		b.WriteString(lipgloss.NewStyle().Foreground(colorYellow).Render("calculating..."))
		b.WriteByte('\n')
	} else {
		tokenStr := FormatThousands(m.totalTokens)
		budgetStr := FormatThousands(m.maxTokens)
		b.WriteString(label.Render("Tokens: "))
		b.WriteString(value.Render(tokenStr))
		b.WriteString(label.Render(" / "))
		b.WriteString(value.Render(budgetStr))
		b.WriteByte('\n')
	}

	// Budget bar.
	bar := RenderBudgetBar(m.budgetUsed, width-2)
	b.WriteString(bar)
	b.WriteByte('\n')
	b.WriteByte('\n')
}

// RenderBudgetBar renders a visual budget utilization bar as a string.
// The bar uses block characters and changes color based on utilization level:
// green (<70%), yellow (70-90%), red (>90%).
func RenderBudgetBar(pct float64, barWidth int) string {
	if barWidth < 5 {
		return FormatPercentage(pct)
	}

	// Clamp percentage.
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	// Reserve space for brackets and percentage label: "[ ... ] XX%"
	innerWidth := barWidth - 7 // 2 brackets + 1 space + up to 4 chars for "100%"
	if innerWidth < 1 {
		innerWidth = 1
	}

	filledCount := int(pct / 100 * float64(innerWidth))
	if filledCount > innerWidth {
		filledCount = innerWidth
	}
	emptyCount := innerWidth - filledCount

	// Select color based on utilization.
	barColor := colorGreen
	switch {
	case pct >= thresholdRed:
		barColor = colorRed
	case pct >= thresholdYellow:
		barColor = colorYellow
	}

	filledStyle := lipgloss.NewStyle().Foreground(barColor)
	emptyStyle := lipgloss.NewStyle().Foreground(colorDim)
	bracketStyle := lipgloss.NewStyle().Foreground(colorDim)
	pctStyle := lipgloss.NewStyle().Foreground(barColor)

	filled := filledStyle.Render(strings.Repeat(barFilled, filledCount))
	empty := emptyStyle.Render(strings.Repeat(barEmpty, emptyCount))
	pctLabel := pctStyle.Render(" " + FormatPercentage(pct))

	return bracketStyle.Render("[") + filled + empty + bracketStyle.Render("]") + pctLabel
}

// renderFileCount writes the file count line.
func (m Model) renderFileCount(b *strings.Builder, _ int) {
	label := lipgloss.NewStyle().Foreground(colorLabel)
	value := lipgloss.NewStyle().Foreground(colorValue)

	b.WriteString(label.Render("Files: "))
	b.WriteString(value.Render(FormatThousands(m.selectedFiles)))
	b.WriteString(label.Render(" / "))
	b.WriteString(value.Render(FormatThousands(m.totalFiles)))
	b.WriteString(label.Render(" selected"))
	b.WriteByte('\n')
}

// renderOutputSize writes the estimated output size.
func (m Model) renderOutputSize(b *strings.Builder, _ int) {
	label := lipgloss.NewStyle().Foreground(colorLabel)
	value := lipgloss.NewStyle().Foreground(colorValue)

	sizeBytes := EstimateOutputSize(m.totalTokens)
	b.WriteString(label.Render("Size: ~"))
	b.WriteString(value.Render(FormatSize(sizeBytes)))
	b.WriteByte('\n')
}

// renderCompression writes the compression savings line.
func (m Model) renderCompression(b *strings.Builder, _ int) {
	label := lipgloss.NewStyle().Foreground(colorLabel)
	value := lipgloss.NewStyle().Foreground(colorValue)

	b.WriteString(label.Render("Compressed: "))
	b.WriteString(value.Render(FormatPercentage(m.compressionPct) + " reduction"))
	b.WriteByte('\n')
}

// renderSecrets writes the redaction/secrets count. The count is shown in red
// if secrets were found.
func (m Model) renderSecrets(b *strings.Builder, _ int) {
	label := lipgloss.NewStyle().Foreground(colorLabel)

	b.WriteString(label.Render("Secrets: "))
	if m.secretsFound > 0 {
		alertStyle := lipgloss.NewStyle().Foreground(colorAlert).Bold(true)
		b.WriteString(alertStyle.Render(fmt.Sprintf("%d found", m.secretsFound)))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(colorValue).Render("0 found"))
	}
	b.WriteByte('\n')
	b.WriteByte('\n')
}

// tierLabels maps tier numbers to short display labels.
var tierLabels = map[int]string{
	0: "critical",
	1: "primary",
	2: "secondary",
	3: "tests",
	4: "docs",
	5: "low",
}

// renderTierBreakdown writes the tier breakdown table.
func (m Model) renderTierBreakdown(b *strings.Builder, width int) {
	headerStyle := lipgloss.NewStyle().
		Foreground(colorHeader).
		Bold(true)

	b.WriteString(headerStyle.Render("Tier Breakdown"))
	b.WriteByte('\n')

	sepWidth := width - 2
	if sepWidth < 0 {
		sepWidth = 0
	}
	sep := strings.Repeat("─", sepWidth)
	b.WriteString(lipgloss.NewStyle().Foreground(colorDim).Render(sep))
	b.WriteByte('\n')

	label := lipgloss.NewStyle().Foreground(colorLabel)
	value := lipgloss.NewStyle().Foreground(colorValue)
	dim := lipgloss.NewStyle().Foreground(colorDim)

	// Iterate through tiers 0-5.
	for tier := int(relevance.Tier0Critical); tier <= int(relevance.Tier5Low); tier++ {
		tierLabel, ok := tierLabels[tier]
		if !ok {
			tierLabel = fmt.Sprintf("tier%d", tier)
		}

		fileCount := m.tierBreakdown[tier]
		tokenCount := m.tierTokens[tier]

		tierName := fmt.Sprintf("T%d (%s)", tier, tierLabel)

		if fileCount == 0 {
			b.WriteString(dim.Render(fmt.Sprintf("  %-16s %5s  %8s",
				tierName, "—", "—")))
		} else {
			b.WriteString(label.Render(fmt.Sprintf("  %-16s", tierName)))
			b.WriteString(value.Render(fmt.Sprintf(" %5d", fileCount)))
			b.WriteString(label.Render("  "))
			b.WriteString(value.Render(fmt.Sprintf("%8s", FormatThousands(tokenCount))))
		}
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
}

// renderTokenizerInfo writes the tokenizer encoding name.
func (m Model) renderTokenizerInfo(b *strings.Builder, _ int) {
	label := lipgloss.NewStyle().Foreground(colorLabel)
	value := lipgloss.NewStyle().Foreground(colorValue)

	name := m.tokenizerName
	if name == "" {
		name = "none"
	}

	b.WriteString(label.Render("Tokenizer: "))
	b.WriteString(value.Render(name))
	b.WriteByte('\n')
}
