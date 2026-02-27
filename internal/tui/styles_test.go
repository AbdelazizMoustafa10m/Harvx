package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestComputeLayout_Boundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		width  int
		height int
		want   LayoutMode
	}{
		{
			name:   "full layout at 120x40",
			width:  120,
			height: 40,
			want:   LayoutFull,
		},
		{
			name:   "full layout boundary at width=100",
			width:  100,
			height: 40,
			want:   LayoutFull,
		},
		{
			name:   "compressed layout boundary at width=99",
			width:  99,
			height: 40,
			want:   LayoutCompressed,
		},
		{
			name:   "compressed layout at 80x30",
			width:  80,
			height: 30,
			want:   LayoutCompressed,
		},
		{
			name:   "compressed layout boundary at width=60",
			width:  60,
			height: 30,
			want:   LayoutCompressed,
		},
		{
			name:   "single panel boundary at width=59",
			width:  59,
			height: 30,
			want:   LayoutSinglePanel,
		},
		{
			name:   "single panel at 50x20",
			width:  50,
			height: 20,
			want:   LayoutSinglePanel,
		},
		{
			name:   "single panel minimum at 40x12",
			width:  40,
			height: 12,
			want:   LayoutSinglePanel,
		},
		{
			name:   "too small below min width at 39x30",
			width:  39,
			height: 30,
			want:   LayoutTooSmall,
		},
		{
			name:   "too small below min height at 80x11",
			width:  80,
			height: 11,
			want:   LayoutTooSmall,
		},
		{
			name:   "too small both below at 30x10",
			width:  30,
			height: 10,
			want:   LayoutTooSmall,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ComputeLayout(tt.width, tt.height)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewStyles_DarkTheme(t *testing.T) {
	t.Parallel()

	s := NewStyles(true, 120, 40)

	assert.Equal(t, lipgloss.Color("252"), s.Colors.Foreground, "dark theme foreground should be bright (252)")
	assert.Equal(t, lipgloss.Color("235"), s.Colors.Background, "dark theme background should be dark (235)")
	assert.Equal(t, lipgloss.Color("117"), s.Colors.Accent, "dark theme accent should be 117")
}

func TestNewStyles_LightTheme(t *testing.T) {
	t.Parallel()

	s := NewStyles(false, 120, 40)

	assert.Equal(t, lipgloss.Color("237"), s.Colors.Foreground, "light theme foreground should be dark (237)")
	assert.Equal(t, lipgloss.Color("253"), s.Colors.Background, "light theme background should be light (253)")
	assert.Equal(t, lipgloss.Color("33"), s.Colors.Accent, "light theme accent should be 33")
}

func TestNewStyles_PanelWidths_Full(t *testing.T) {
	t.Parallel()

	s := NewStyles(true, 120, 40)

	assert.Equal(t, LayoutFull, s.Layout, "120x40 should produce LayoutFull")
	assert.Equal(t, 78, s.LeftPanelWidth, "left panel should be 65%% of 120 = 78")
	assert.Equal(t, 41, s.RightPanelWidth, "right panel should be 120 - 78 - 1 = 41")
	assert.Equal(t, 38, s.ContentHeight, "content height should be 40 - 2 = 38")
}

func TestNewStyles_PanelWidths_Compressed(t *testing.T) {
	t.Parallel()

	s := NewStyles(true, 80, 30)

	assert.Equal(t, LayoutCompressed, s.Layout, "80x30 should produce LayoutCompressed")
	assert.Equal(t, 48, s.LeftPanelWidth, "left panel should be 60%% of 80 = 48")
	assert.Equal(t, 31, s.RightPanelWidth, "right panel should be 80 - 48 - 1 = 31")
	assert.Equal(t, 28, s.ContentHeight, "content height should be 30 - 2 = 28")
}

func TestNewStyles_PanelWidths_SinglePanel(t *testing.T) {
	t.Parallel()

	s := NewStyles(true, 50, 20)

	assert.Equal(t, LayoutSinglePanel, s.Layout, "50x20 should produce LayoutSinglePanel")
	assert.Equal(t, 50, s.LeftPanelWidth, "left panel should be 100%% of width = 50")
	assert.Equal(t, 0, s.RightPanelWidth, "right panel should be 0 in single panel mode")
	assert.Equal(t, 18, s.ContentHeight, "content height should be 20 - 2 = 18")
}

func TestNewStyles_ResizeRecalculates(t *testing.T) {
	t.Parallel()

	small := NewStyles(true, 80, 30)
	large := NewStyles(true, 120, 40)

	assert.NotEqual(t, small.LeftPanelWidth, large.LeftPanelWidth,
		"different terminal widths should produce different left panel widths")
	assert.NotEqual(t, small.RightPanelWidth, large.RightPanelWidth,
		"different terminal widths should produce different right panel widths")
	assert.NotEqual(t, small.ContentHeight, large.ContentHeight,
		"different terminal heights should produce different content heights")
	assert.NotEqual(t, small.Layout, large.Layout,
		"80-wide and 120-wide should have different layout modes")
}
