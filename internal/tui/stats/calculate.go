package stats

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/harvx/harvx/internal/tui/filetree"
)

// debounceDelay is the duration to wait after the last file toggle before
// triggering a token recalculation. This prevents UI jank from rapid toggles.
const debounceDelay = 200 * time.Millisecond

// recalcTickMsg is sent when the debounce timer fires. It carries the
// generation counter at the time the tick was created so that stale ticks
// (from earlier toggles that were superseded) can be discarded.
type recalcTickMsg struct {
	generation uint64
}

// tokenCountResult carries the result of an asynchronous token count
// calculation back to the stats model.
type tokenCountResult struct {
	// TotalTokens is the sum of token counts across all included files.
	TotalTokens int

	// SelectedFiles is the number of included files.
	SelectedFiles int

	// TotalFiles is the total number of files (included + excluded).
	TotalFiles int

	// TierBreakdown maps tier number to the count of included files in that tier.
	TierBreakdown map[int]int

	// TierTokens maps tier number to the total tokens in that tier.
	TierTokens map[int]int

	// SecretsFound is the number of files with detected secrets.
	SecretsFound int
}

// scheduleDebounce returns a tea.Cmd that fires a recalcTickMsg after the
// debounce delay, tagged with the given generation counter.
func scheduleDebounce(gen uint64) tea.Cmd {
	return tea.Tick(debounceDelay, func(_ time.Time) tea.Msg {
		return recalcTickMsg{generation: gen}
	})
}

// calculateTokens returns a tea.Cmd that synchronously iterates the file tree
// and sums token counts for included files. The result is sent back as a
// tokenCountResult message.
//
// This runs as a tea.Cmd (async from the Bubble Tea runtime's perspective) to
// avoid blocking the Update loop during large tree traversals.
func calculateTokens(root *filetree.Node) tea.Cmd {
	return func() tea.Msg {
		if root == nil {
			return tokenCountResult{}
		}

		result := tokenCountResult{
			TierBreakdown: make(map[int]int, 6),
			TierTokens:    make(map[int]int, 6),
		}

		walkTree(root, &result)

		return result
	}
}

// walkTree recursively traverses the file tree, accumulating statistics for
// included file nodes into result.
func walkTree(node *filetree.Node, result *tokenCountResult) {
	for _, child := range node.Children {
		if child.IsDir {
			walkTree(child, result)
			continue
		}

		// Count all files for the total.
		result.TotalFiles++

		if child.Included != filetree.Included {
			continue
		}

		result.SelectedFiles++
		result.TotalTokens += child.TokenCount
		result.TierBreakdown[child.Tier]++
		result.TierTokens[child.Tier] += child.TokenCount

		if child.HasSecrets {
			result.SecretsFound++
		}
	}
}
