package filetree

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		path  string
		nname string
		isDir bool
	}{
		{
			name:  "file node",
			path:  "src/main.go",
			nname: "main.go",
			isDir: false,
		},
		{
			name:  "directory node",
			path:  "src",
			nname: "src",
			isDir: true,
		},
		{
			name:  "backslash path normalized",
			path:  "src\\main.go",
			nname: "main.go",
			isDir: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n := NewNode(tt.path, tt.nname, tt.isDir)
			assert.Equal(t, tt.nname, n.Name)
			assert.Equal(t, tt.isDir, n.IsDir)
			assert.Equal(t, Excluded, n.Included)
			assert.False(t, n.Expanded)
			assert.Nil(t, n.Children)
			assert.Nil(t, n.Parent)
		})
	}
}

func TestNode_AddChild(t *testing.T) {
	t.Parallel()

	parent := NewNode("src", "src", true)
	child := NewNode("src/main.go", "main.go", false)

	parent.AddChild(child)

	require.Len(t, parent.Children, 1)
	assert.Equal(t, child, parent.Children[0])
	assert.Equal(t, parent, child.Parent)
	assert.Equal(t, 1, child.Depth())
}

func TestNode_AddChild_SetsDepthRecursively(t *testing.T) {
	t.Parallel()

	root := NewNode("", "root", true)
	dir := NewNode("a", "a", true)
	file := NewNode("a/b.go", "b.go", false)

	root.AddChild(dir)
	dir.AddChild(file)

	assert.Equal(t, 0, root.Depth())
	assert.Equal(t, 1, dir.Depth())
	assert.Equal(t, 2, file.Depth())
}

func TestNode_SortChildren(t *testing.T) {
	t.Parallel()

	parent := NewNode("", "root", true)
	parent.AddChild(NewNode("z.go", "z.go", false))
	parent.AddChild(NewNode("a.go", "a.go", false))
	parent.AddChild(NewNode("src", "src", true))
	parent.AddChild(NewNode("lib", "lib", true))
	parent.AddChild(NewNode("m.go", "m.go", false))

	parent.SortChildren()

	names := make([]string, len(parent.Children))
	for i, c := range parent.Children {
		names[i] = c.Name
	}

	// Directories first (alphabetical), then files (alphabetical).
	assert.Equal(t, []string{"lib", "src", "a.go", "m.go", "z.go"}, names)
}

func TestNode_SortChildren_CaseInsensitive(t *testing.T) {
	t.Parallel()

	parent := NewNode("", "root", true)
	parent.AddChild(NewNode("Bfile.go", "Bfile.go", false))
	parent.AddChild(NewNode("afile.go", "afile.go", false))

	parent.SortChildren()

	assert.Equal(t, "afile.go", parent.Children[0].Name)
	assert.Equal(t, "Bfile.go", parent.Children[1].Name)
}

func TestNode_Toggle_File(t *testing.T) {
	t.Parallel()

	file := NewNode("main.go", "main.go", false)
	assert.Equal(t, Excluded, file.Included)

	file.Toggle()
	assert.Equal(t, Included, file.Included)

	file.Toggle()
	assert.Equal(t, Excluded, file.Included)
}

func TestNode_Toggle_Directory_SetsAllDescendants(t *testing.T) {
	t.Parallel()

	root := NewNode("", "root", true)
	dir := NewNode("src", "src", true)
	f1 := NewNode("src/a.go", "a.go", false)
	f2 := NewNode("src/b.go", "b.go", false)
	sub := NewNode("src/sub", "sub", true)
	f3 := NewNode("src/sub/c.go", "c.go", false)

	root.AddChild(dir)
	dir.AddChild(f1)
	dir.AddChild(f2)
	dir.AddChild(sub)
	sub.AddChild(f3)

	// Toggle dir to Included.
	dir.Toggle()
	assert.Equal(t, Included, dir.Included)
	assert.Equal(t, Included, f1.Included)
	assert.Equal(t, Included, f2.Included)
	assert.Equal(t, Included, sub.Included)
	assert.Equal(t, Included, f3.Included)
	assert.Equal(t, Included, root.Included, "root should be included when its only child is included")

	// Toggle dir to Excluded.
	dir.Toggle()
	assert.Equal(t, Excluded, dir.Included)
	assert.Equal(t, Excluded, f1.Included)
	assert.Equal(t, Excluded, f2.Included)
	assert.Equal(t, Excluded, sub.Included)
	assert.Equal(t, Excluded, f3.Included)
	assert.Equal(t, Excluded, root.Included)
}

func TestNode_Toggle_PartialToIncluded(t *testing.T) {
	t.Parallel()

	dir := NewNode("src", "src", true)
	f1 := NewNode("src/a.go", "a.go", false)
	f2 := NewNode("src/b.go", "b.go", false)
	dir.AddChild(f1)
	dir.AddChild(f2)

	// Include only one file to make dir partial.
	f1.Toggle()
	assert.Equal(t, Included, f1.Included)
	assert.Equal(t, Partial, dir.Included)

	// Toggle the partial directory -- should set all to Included.
	dir.Toggle()
	assert.Equal(t, Included, dir.Included)
	assert.Equal(t, Included, f1.Included)
	assert.Equal(t, Included, f2.Included)
}

func TestNode_PropagateUp_DeepNesting(t *testing.T) {
	t.Parallel()

	// Build: root -> a -> b -> c -> file
	root := NewNode("", "root", true)
	a := NewNode("a", "a", true)
	b := NewNode("a/b", "b", true)
	c := NewNode("a/b/c", "c", true)
	file := NewNode("a/b/c/f.go", "f.go", false)

	root.AddChild(a)
	a.AddChild(b)
	b.AddChild(c)
	c.AddChild(file)

	// Toggle the leaf file.
	file.Toggle()
	assert.Equal(t, Included, file.Included)
	assert.Equal(t, Included, c.Included, "single child included = dir included")
	assert.Equal(t, Included, b.Included)
	assert.Equal(t, Included, a.Included)
	assert.Equal(t, Included, root.Included)

	// Toggle it back.
	file.Toggle()
	assert.Equal(t, Excluded, file.Included)
	assert.Equal(t, Excluded, c.Included)
	assert.Equal(t, Excluded, b.Included)
	assert.Equal(t, Excluded, a.Included)
	assert.Equal(t, Excluded, root.Included)
}

func TestNode_PropagateUp_MixedChildren(t *testing.T) {
	t.Parallel()

	root := NewNode("", "root", true)
	f1 := NewNode("a.go", "a.go", false)
	f2 := NewNode("b.go", "b.go", false)
	f3 := NewNode("c.go", "c.go", false)
	root.AddChild(f1)
	root.AddChild(f2)
	root.AddChild(f3)

	// Include one file.
	f1.Toggle()
	assert.Equal(t, Partial, root.Included)

	// Include all files.
	f2.Toggle()
	f3.Toggle()
	assert.Equal(t, Included, root.Included)

	// Exclude one file.
	f2.Toggle()
	assert.Equal(t, Partial, root.Included)

	// Exclude all files.
	f1.Toggle()
	f3.Toggle()
	assert.Equal(t, Excluded, root.Included)
}

func TestNode_VisibleNodes_AllCollapsed(t *testing.T) {
	t.Parallel()

	root := NewNode("", "root", true)
	root.Expanded = true
	dir := NewNode("src", "src", true)
	f1 := NewNode("main.go", "main.go", false)
	root.AddChild(dir)
	root.AddChild(f1)

	// dir is collapsed, so its children should not appear.
	sub := NewNode("src/a.go", "a.go", false)
	dir.AddChild(sub)

	visible := root.VisibleNodes()
	assert.Len(t, visible, 2)
	assert.Equal(t, "src", visible[0].Name)
	assert.Equal(t, "main.go", visible[1].Name)
}

func TestNode_VisibleNodes_Expanded(t *testing.T) {
	t.Parallel()

	root := NewNode("", "root", true)
	root.Expanded = true
	dir := NewNode("src", "src", true)
	dir.Expanded = true
	f1 := NewNode("src/a.go", "a.go", false)
	f2 := NewNode("main.go", "main.go", false)
	root.AddChild(dir)
	root.AddChild(f2)
	dir.AddChild(f1)

	visible := root.VisibleNodes()
	assert.Len(t, visible, 3)
	assert.Equal(t, "src", visible[0].Name)
	assert.Equal(t, "a.go", visible[1].Name)
	assert.Equal(t, "main.go", visible[2].Name)
}

func TestNode_VisibleNodes_Empty(t *testing.T) {
	t.Parallel()

	root := NewNode("", "root", true)
	root.Expanded = true

	visible := root.VisibleNodes()
	assert.Empty(t, visible)
}

func TestNode_FindByPath(t *testing.T) {
	t.Parallel()

	root := NewNode("", "root", true)
	dir := NewNode("src", "src", true)
	file := NewNode("src/main.go", "main.go", false)
	root.AddChild(dir)
	dir.AddChild(file)

	tests := []struct {
		name   string
		path   string
		wantNil bool
	}{
		{name: "find root", path: "", wantNil: false},
		{name: "find directory", path: "src", wantNil: false},
		{name: "find file", path: "src/main.go", wantNil: false},
		{name: "not found", path: "nonexistent.go", wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			found := root.FindByPath(tt.path)
			if tt.wantNil {
				assert.Nil(t, found)
			} else {
				require.NotNil(t, found)
				assert.Equal(t, tt.path, found.Path)
			}
		})
	}
}

func TestNode_IncludedFiles(t *testing.T) {
	t.Parallel()

	root := NewNode("", "root", true)
	f1 := NewNode("a.go", "a.go", false)
	f2 := NewNode("b.go", "b.go", false)
	dir := NewNode("src", "src", true)
	f3 := NewNode("src/c.go", "c.go", false)
	root.AddChild(f1)
	root.AddChild(f2)
	root.AddChild(dir)
	dir.AddChild(f3)

	// Include a.go and c.go.
	f1.Toggle()
	f3.Toggle()

	included := root.IncludedFiles()
	assert.Len(t, included, 2)
	assert.Contains(t, included, "a.go")
	assert.Contains(t, included, "src/c.go")
}

func TestNode_IncludedFiles_None(t *testing.T) {
	t.Parallel()

	root := NewNode("", "root", true)
	root.AddChild(NewNode("a.go", "a.go", false))
	root.AddChild(NewNode("b.go", "b.go", false))

	included := root.IncludedFiles()
	assert.Empty(t, included)
}

func TestInclusionState_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state InclusionState
		want  string
	}{
		{Excluded, "excluded"},
		{Included, "included"},
		{Partial, "partial"},
		{InclusionState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.state.String())
		})
	}
}
