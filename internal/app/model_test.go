package app

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wingitman/listicles/internal/config"
)

// ─── Test helpers ─────────────────────────────────────────────────────────────

// newModel creates a Model rooted at dir using default config.
func newModel(t *testing.T, dir string) *Model {
	t.Helper()
	cfg := config.Default()
	m, err := New(cfg, dir, "", "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	m.width = 120
	m.height = 40
	return m
}

// newModelWithDirs creates a temp dir containing the given subdirectory names
// and returns a Model rooted there.
func newModelWithDirs(t *testing.T, dirs ...string) (*Model, string) {
	t.Helper()
	root := t.TempDir()
	for _, d := range dirs {
		if err := os.Mkdir(filepath.Join(root, d), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	return newModel(t, root), root
}

// newModelWithDirsAndFiles creates a temp dir with subdirs and files.
func newModelWithDirsAndFiles(t *testing.T, dirs []string, files []string) (*Model, string) {
	t.Helper()
	root := t.TempDir()
	for _, d := range dirs {
		if err := os.Mkdir(filepath.Join(root, d), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(root, f), []byte("content"), 0644); err != nil {
			t.Fatalf("create %s: %v", f, err)
		}
	}
	return newModel(t, root), root
}

// sendKey sends a key press to the model and returns the updated model.
func sendKey(m *Model, key string) *Model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	if mm, ok := updated.(*Model); ok {
		return mm
	}
	// Value receiver returns Model not *Model
	mm := updated.(Model)
	return &mm
}

// sendSpecialKey sends a named special key (e.g. "up", "down", "enter").
func sendSpecialKey(m *Model, keyType tea.KeyType) *Model {
	updated, _ := m.Update(tea.KeyMsg{Type: keyType})
	if mm, ok := updated.(*Model); ok {
		return mm
	}
	mm := updated.(Model)
	return &mm
}

// ─── New / initTree ───────────────────────────────────────────────────────────

func TestNew_InitialisesTree(t *testing.T) {
	m, _ := newModelWithDirs(t, "alpha", "beta", "gamma")

	if len(m.nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(m.nodes))
	}
	for i, n := range m.nodes {
		if n.Depth != 0 {
			t.Errorf("node[%d] depth = %d, want 0", i, n.Depth)
		}
		if n.Expanded {
			t.Errorf("node[%d] should not be expanded on init", i)
		}
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestNew_EmptyDir(t *testing.T) {
	m, _ := newModelWithDirs(t) // no subdirs
	if len(m.nodes) != 0 {
		t.Errorf("expected 0 nodes in empty dir, got %d", len(m.nodes))
	}
}

func TestNew_FilesOnlyDir_DirsMode(t *testing.T) {
	_, root := newModelWithDirsAndFiles(t, nil, []string{"a.txt", "b.txt"})
	fm := newModel(t, root) // default is dirs-only
	if len(fm.nodes) != 0 {
		t.Errorf("expected 0 nodes in files-only dir (dirs mode), got %d", len(fm.nodes))
	}
}

// ─── expandNode / collapseNode ────────────────────────────────────────────────

func TestExpandNode_LoadsChildren(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	os.Mkdir(parent, 0755)
	os.Mkdir(filepath.Join(parent, "child1"), 0755)
	os.Mkdir(filepath.Join(parent, "child2"), 0755)

	m := newModel(t, root)
	if len(m.nodes) != 1 {
		t.Fatalf("expected 1 root node, got %d", len(m.nodes))
	}

	if err := m.expandNode(0); err != nil {
		t.Fatalf("expandNode: %v", err)
	}

	if !m.nodes[0].Expanded {
		t.Error("node[0] should be expanded")
	}
	if len(m.nodes) != 3 { // parent + 2 children
		t.Errorf("expected 3 nodes after expand, got %d", len(m.nodes))
	}
	if m.nodes[1].Depth != 1 || m.nodes[2].Depth != 1 {
		t.Error("children should be at depth 1")
	}
}

func TestExpandNode_AlreadyExpanded_Collapses(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "dir"), 0755)
	os.Mkdir(filepath.Join(root, "dir", "child"), 0755)

	m := newModel(t, root)
	_ = m.expandNode(0)
	if len(m.nodes) != 2 {
		t.Fatalf("expected 2 nodes after first expand, got %d", len(m.nodes))
	}

	// Second expand = collapse
	_ = m.expandNode(0)
	if m.nodes[0].Expanded {
		t.Error("node should be collapsed after second expand")
	}
	if len(m.nodes) != 1 {
		t.Errorf("expected 1 node after collapse, got %d", len(m.nodes))
	}
}

func TestExpandNode_EmptyDir(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "empty"), 0755)

	m := newModel(t, root)
	_ = m.expandNode(0)

	if !m.nodes[0].Expanded {
		t.Error("empty dir should still be marked Expanded")
	}
	if len(m.nodes) != 1 {
		t.Errorf("expected 1 node (no children), got %d", len(m.nodes))
	}
}

func TestCollapseNode_RemovesDescendants(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "dir"), 0755)
	os.Mkdir(filepath.Join(root, "dir", "c1"), 0755)
	os.Mkdir(filepath.Join(root, "dir", "c2"), 0755)

	m := newModel(t, root)
	_ = m.expandNode(0)
	if len(m.nodes) != 3 {
		t.Fatalf("expected 3 nodes after expand, got %d", len(m.nodes))
	}

	m.collapseNode(0)
	if m.nodes[0].Expanded {
		t.Error("node should be collapsed")
	}
	if len(m.nodes) != 1 {
		t.Errorf("expected 1 node after collapse, got %d", len(m.nodes))
	}
}

func TestCollapseNode_NestedExpansions(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "outer", "inner", "deep"), 0755)

	m := newModel(t, root)
	_ = m.expandNode(0) // expand outer
	_ = m.expandNode(1) // expand inner
	// nodes: outer, inner, deep
	if len(m.nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(m.nodes))
	}

	// Collapsing outer should remove both inner and deep
	m.collapseNode(0)
	if len(m.nodes) != 1 {
		t.Errorf("expected 1 node after collapsing outer, got %d", len(m.nodes))
	}
}

// ─── findNodeByPath / parentNodeIdx ──────────────────────────────────────────

func TestFindNodeByPath(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "a"), 0755)
	os.Mkdir(filepath.Join(root, "b"), 0755)

	m := newModel(t, root)
	idx := m.findNodeByPath(filepath.Join(root, "b"))
	if idx != 1 {
		t.Errorf("findNodeByPath(b) = %d, want 1", idx)
	}
	if m.findNodeByPath("/nonexistent/path") != -1 {
		t.Error("nonexistent path should return -1")
	}
}

func TestParentNodeIdx(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "parent"), 0755)
	os.Mkdir(filepath.Join(root, "parent", "child"), 0755)

	m := newModel(t, root)
	_ = m.expandNode(0)
	// nodes: [0]=parent [1]=child

	if m.parentNodeIdx(1) != 0 {
		t.Errorf("parentNodeIdx(child) = %d, want 0", m.parentNodeIdx(1))
	}
	if m.parentNodeIdx(0) != -1 {
		t.Errorf("parentNodeIdx(root-depth node) = %d, want -1", m.parentNodeIdx(0))
	}
}

// ─── currentOperationDir ─────────────────────────────────────────────────────

func TestCurrentOperationDir_OnDir(t *testing.T) {
	m, root := newModelWithDirs(t, "mydir")
	dir := m.currentOperationDir()
	if dir != filepath.Join(root, "mydir") {
		t.Errorf("currentOperationDir on dir = %q, want %q", dir, filepath.Join(root, "mydir"))
	}
}

func TestCurrentOperationDir_OnFile(t *testing.T) {
	_, root := newModelWithDirsAndFiles(t, nil, []string{"file.txt"})
	cfg := config.Default()
	cfg.Display.DefaultListMode = "dirs_and_files"
	m2, err := New(cfg, root, "", "")
	if err != nil {
		t.Fatal(err)
	}
	m2.width = 120
	m2.height = 40
	m2.cursor = 0 // file.txt

	dir := m2.currentOperationDir()
	if dir != root {
		t.Errorf("currentOperationDir on file = %q, want %q", dir, root)
	}
}

func TestCurrentOperationDir_NoNodes(t *testing.T) {
	m, root := newModelWithDirs(t) // empty dir
	if m.currentOperationDir() != root {
		t.Errorf("expected rootDir for empty model, got %q", m.currentOperationDir())
	}
}

// ─── navigateLeft / goToParentDir ─────────────────────────────────────────────

func TestNavigateLeft_ExpandedDir_Collapses(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "dir"), 0755)
	os.Mkdir(filepath.Join(root, "dir", "child"), 0755)

	m := newModel(t, root)
	_ = m.expandNode(0)
	m.cursor = 0 // on the expanded dir

	m.navigateLeft()
	if m.nodes[0].Expanded {
		t.Error("navigateLeft on expanded dir should collapse it")
	}
	if m.cursor != 0 {
		t.Errorf("cursor should stay at 0 after collapse, got %d", m.cursor)
	}
}

func TestNavigateLeft_CollapsedAtRoot_GoesUp(t *testing.T) {
	// Create: /tmp/root/subdir
	parentDir := t.TempDir()
	root := filepath.Join(parentDir, "root")
	os.Mkdir(root, 0755)
	os.Mkdir(filepath.Join(root, "subdir"), 0755)

	m := newModel(t, root)
	m.cursor = 0

	m.navigateLeft()
	// Should now be rooted at parentDir
	if m.rootDir != parentDir {
		t.Errorf("rootDir = %q, want %q", m.rootDir, parentDir)
	}
}

func TestNavigateLeft_EmptyDir_GoesUp(t *testing.T) {
	parentDir := t.TempDir()
	emptyChild := filepath.Join(parentDir, "empty")
	os.Mkdir(emptyChild, 0755)

	m := newModel(t, emptyChild) // start in empty dir
	m.navigateLeft()

	if m.rootDir != parentDir {
		t.Errorf("rootDir = %q, want %q", m.rootDir, parentDir)
	}
}

func TestNavigateLeft_NestedNode_CollapsesParent(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "parent"), 0755)
	os.Mkdir(filepath.Join(root, "parent", "child"), 0755)

	m := newModel(t, root)
	_ = m.expandNode(0) // expand parent
	m.cursor = 1        // cursor on child

	m.navigateLeft()
	// Parent should be collapsed, cursor should jump to it
	if m.nodes[0].Expanded {
		t.Error("parent should be collapsed after navigateLeft from child")
	}
	if m.cursor != 0 {
		t.Errorf("cursor should be at parent (0), got %d", m.cursor)
	}
}

func TestGoToParentDir_AtFilesystemRoot(t *testing.T) {
	m, _ := newModelWithDirs(t, "a")
	m.rootDir = "/"
	m.goToParentDir()
	// Should be a no-op
	if m.rootDir != "/" {
		t.Error("goToParentDir at / should be a no-op")
	}
}

// ─── resolveDigitBuffer ───────────────────────────────────────────────────────

func TestResolveDigitBuffer_ValidN(t *testing.T) {
	m, _ := newModelWithDirs(t, "alpha", "beta", "gamma")
	m.digitBuffer = "2"
	m.resolveDigitBuffer()

	// Item 2 = index 1 (beta)
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (beta)", m.cursor)
	}
}

func TestResolveDigitBuffer_MultiDigit(t *testing.T) {
	m, _ := newModelWithDirs(t,
		"d01", "d02", "d03", "d04", "d05",
		"d06", "d07", "d08", "d09", "d10", "d11", "d12",
	)
	m.digitBuffer = "11"
	m.resolveDigitBuffer()
	// Item 11 = index 10 (d11)
	if m.cursor != 10 {
		t.Errorf("cursor = %d, want 10", m.cursor)
	}
}

func TestResolveDigitBuffer_OutOfRange(t *testing.T) {
	m, _ := newModelWithDirs(t, "only")
	m.cursor = 0
	m.digitBuffer = "99"
	m.resolveDigitBuffer() // should not crash
	// cursor should remain unchanged since there's no 99th item
	if m.cursor != 0 {
		t.Errorf("cursor should remain 0, got %d", m.cursor)
	}
}

func TestResolveDigitBuffer_ExpandsDir(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "mydir"), 0755)
	os.Mkdir(filepath.Join(root, "mydir", "child"), 0755)

	m := newModel(t, root)
	m.digitBuffer = "1"
	m.resolveDigitBuffer()

	// Node 0 (mydir) should now be expanded and cursor on its first child
	if !m.nodes[0].Expanded {
		t.Error("resolveDigitBuffer should expand the dir")
	}
	if m.cursor != 1 {
		t.Errorf("cursor should be on first child (1), got %d", m.cursor)
	}
}

// ─── calcPageJump ─────────────────────────────────────────────────────────────

func TestCalcPageJump(t *testing.T) {
	cases := []struct {
		n    int
		want int
	}{
		{1, 1},
		{2, 1},
		{4, 2},
		{8, 3},
		{16, 4},
		{32, 5},
		{64, 6},
		{128, 7},
	}
	for _, c := range cases {
		got := calcPageJump(c.n)
		if got != c.want {
			t.Errorf("calcPageJump(%d) = %d, want %d", c.n, got, c.want)
		}
	}
}

// ─── nthSiblingAtDepth ────────────────────────────────────────────────────────

func TestNthSiblingAtDepth(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "a"), 0755)
	os.Mkdir(filepath.Join(root, "b"), 0755)
	os.Mkdir(filepath.Join(root, "c"), 0755)

	m := newModel(t, root)

	// 0-based: 0=a, 1=b, 2=c
	if idx := nthSiblingAtDepth(m.nodes, 0, 0, 0); idx != 0 {
		t.Errorf("nthSiblingAtDepth(0) = %d, want 0", idx)
	}
	if idx := nthSiblingAtDepth(m.nodes, 0, 0, 2); idx != 2 {
		t.Errorf("nthSiblingAtDepth(2) = %d, want 2", idx)
	}
	if idx := nthSiblingAtDepth(m.nodes, 0, 0, 5); idx != -1 {
		t.Errorf("out of range should return -1, got %d", idx)
	}
}

// ─── clampCursor ─────────────────────────────────────────────────────────────

func TestClampCursor_BelowZero(t *testing.T) {
	m, _ := newModelWithDirs(t, "a", "b")
	m.cursor = -1
	m.clampCursor()
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestClampCursor_AboveLen(t *testing.T) {
	m, _ := newModelWithDirs(t, "a", "b", "c")
	m.cursor = 100
	m.clampCursor()
	if m.cursor != 2 {
		t.Errorf("cursor = %d, want 2", m.cursor)
	}
}

// ─── applyLiveFilter ─────────────────────────────────────────────────────────

func TestApplyLiveFilter_BasicMatch(t *testing.T) {
	m, _ := newModelWithDirs(t, "foobar", "bazqux", "foobaz")
	m.prevNodes = make([]TreeNode, len(m.nodes))
	copy(m.prevNodes, m.nodes)
	m.textInput.SetValue("foo")

	m.applyLiveFilter()

	if len(m.searchLiveNodes) != 2 {
		t.Errorf("expected 2 matches for 'foo', got %d", len(m.searchLiveNodes))
	}
}

func TestApplyLiveFilter_CaseInsensitive(t *testing.T) {
	m, _ := newModelWithDirs(t, "Documents", "downloads", "Pictures")
	m.prevNodes = make([]TreeNode, len(m.nodes))
	copy(m.prevNodes, m.nodes)
	m.textInput.SetValue("DOC")

	m.applyLiveFilter()
	if len(m.searchLiveNodes) != 1 {
		t.Errorf("expected 1 case-insensitive match, got %d", len(m.searchLiveNodes))
	}
}

func TestApplyLiveFilter_EmptyQuery_Restores(t *testing.T) {
	m, _ := newModelWithDirs(t, "alpha", "beta")
	m.prevNodes = make([]TreeNode, len(m.nodes))
	copy(m.prevNodes, m.nodes)

	m.textInput.SetValue("")
	m.applyLiveFilter()

	if m.searchLiveNodes != nil {
		t.Error("empty query should set searchLiveNodes to nil")
	}
}

func TestApplyLiveFilter_RespectsMaxResults(t *testing.T) {
	// Create 10 dirs all matching "dir"
	dirs := make([]string, 10)
	for i := range dirs {
		dirs[i] = "dir" + string(rune('a'+i))
	}
	m, _ := newModelWithDirs(t, dirs...)
	m.cfg.Display.SearchMaxResults = 3
	m.prevNodes = make([]TreeNode, len(m.nodes))
	copy(m.prevNodes, m.nodes)
	m.textInput.SetValue("dir")

	m.applyLiveFilter()
	if len(m.searchLiveNodes) != 3 {
		t.Errorf("expected max 3 results, got %d", len(m.searchLiveNodes))
	}
}

func TestApplyLiveFilter_StripsFlagsFromQuery(t *testing.T) {
	m, _ := newModelWithDirs(t, "foobaz", "barqux")
	m.prevNodes = make([]TreeNode, len(m.nodes))
	copy(m.prevNodes, m.nodes)
	m.textInput.SetValue("-r foo")

	m.applyLiveFilter()
	// Should match "foobaz" — the -r flag is stripped
	if len(m.searchLiveNodes) != 1 {
		t.Errorf("expected 1 match after stripping -r, got %d", len(m.searchLiveNodes))
	}
}

// ─── rebuildTree ─────────────────────────────────────────────────────────────

func TestRebuildTree_PreservesExpansion(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "aaa"), 0755)
	os.Mkdir(filepath.Join(root, "aaa", "child"), 0755)
	os.Mkdir(filepath.Join(root, "bbb"), 0755)

	m := newModel(t, root)
	_ = m.expandNode(0) // expand aaa

	if err := m.rebuildTree(); err != nil {
		t.Fatalf("rebuildTree: %v", err)
	}

	// aaa should still be expanded
	aaaIdx := m.findNodeByPath(filepath.Join(root, "aaa"))
	if aaaIdx < 0 {
		t.Fatal("aaa node not found after rebuild")
	}
	if !m.nodes[aaaIdx].Expanded {
		t.Error("aaa should still be expanded after rebuildTree")
	}
}

// ─── Update key handler tests ─────────────────────────────────────────────────

func TestUpdate_QuitKey(t *testing.T) {
	m, _ := newModelWithDirs(t, "a")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("quit key should return a command")
	}
	// Execute the command and check it's tea.Quit
	msg := cmd()
	if msg != tea.Quit() {
		t.Error("quit key command should produce tea.Quit")
	}
}

func TestUpdate_UpDown_MoveCursor(t *testing.T) {
	m, _ := newModelWithDirs(t, "a", "b", "c")
	m.cursor = 1

	m2 := sendSpecialKey(m, tea.KeyUp)
	if m2.cursor != 0 {
		t.Errorf("after up: cursor = %d, want 0", m2.cursor)
	}

	m3 := sendSpecialKey(m, tea.KeyDown)
	if m3.cursor != 2 {
		t.Errorf("after down: cursor = %d, want 2", m3.cursor)
	}
}

func TestUpdate_Right_ExpandsAndMovesCursor(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "dir"), 0755)
	os.Mkdir(filepath.Join(root, "dir", "child"), 0755)

	m := newModel(t, root)
	m.cursor = 0

	m2 := sendSpecialKey(m, tea.KeyRight)

	if !m2.nodes[0].Expanded {
		t.Error("right key should expand directory")
	}
	if m2.cursor != 1 {
		t.Errorf("cursor should move to first child (1), got %d", m2.cursor)
	}
}

func TestUpdate_Right_AlreadyExpanded_NoMove(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "dir"), 0755)
	os.Mkdir(filepath.Join(root, "dir", "child"), 0755)

	m := newModel(t, root)
	_ = m.expandNode(0)
	m.cursor = 0 // stay on the dir

	m2 := sendSpecialKey(m, tea.KeyRight)
	// Should collapse (toggle), cursor stays on the dir
	if m2.nodes[0].Expanded {
		t.Error("right on already-expanded dir should collapse it")
	}
}

func TestUpdate_Left_ExpandedDir_Collapses(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "dir"), 0755)
	os.Mkdir(filepath.Join(root, "dir", "child"), 0755)

	m := newModel(t, root)
	_ = m.expandNode(0)
	m.cursor = 0

	m2 := sendSpecialKey(m, tea.KeyLeft)
	if m2.nodes[0].Expanded {
		t.Error("left on expanded dir should collapse it")
	}
}

func TestUpdate_PageDown(t *testing.T) {
	// 32 items → calcPageJump(32) = 5
	dirs := make([]string, 32)
	for i := range dirs {
		dirs[i] = "dir" + string(rune('a'+i%26)) + string(rune('0'+i/26))
	}
	m, _ := newModelWithDirs(t, dirs...)
	m.cursor = 0

	m2 := sendSpecialKey(m, tea.KeyPgDown)
	expected := calcPageJump(32)
	if m2.cursor != expected {
		t.Errorf("after pgdown: cursor = %d, want %d", m2.cursor, expected)
	}
}

func TestUpdate_JumpTop(t *testing.T) {
	m, _ := newModelWithDirs(t, "a", "b", "c")
	m.cursor = 2

	m2 := sendSpecialKey(m, tea.KeyHome)
	if m2.cursor != 0 {
		t.Errorf("after home: cursor = %d, want 0", m2.cursor)
	}
}

func TestUpdate_JumpBottom(t *testing.T) {
	m, _ := newModelWithDirs(t, "a", "b", "c")
	m.cursor = 0

	m2 := sendSpecialKey(m, tea.KeyEnd)
	if m2.cursor != 2 {
		t.Errorf("after end: cursor = %d, want 2", m2.cursor)
	}
}

func TestUpdate_ToggleList(t *testing.T) {
	m, _ := newModelWithDirs(t, "a")
	if m.listMode != ListDirsOnly {
		t.Fatal("initial listMode should be ListDirsOnly")
	}

	m2 := sendKey(m, "f")
	if m2.listMode != ListDirsAndFiles {
		t.Error("after f: should be ListDirsAndFiles")
	}

	m3 := sendKey(m2, "f")
	if m3.listMode != ListDirsOnly {
		t.Error("after second f: should be ListDirsOnly")
	}
}

func TestUpdate_ToggleHidden(t *testing.T) {
	m, _ := newModelWithDirs(t, "a")
	if m.showHidden {
		t.Fatal("showHidden should default to false")
	}

	m2 := sendKey(m, ".")
	if !m2.showHidden {
		t.Error("after '.': showHidden should be true")
	}

	m3 := sendKey(m2, ".")
	if m3.showHidden {
		t.Error("after second '.': showHidden should be false")
	}
}

func TestUpdate_DetailsToggle(t *testing.T) {
	m, _ := newModelWithDirs(t, "a")
	levels := []DetailLevel{DetailNone, DetailCount, DetailSize, DetailFullPath, DetailNone}

	for i, expected := range levels[:4] {
		if m.detailLevel != expected {
			t.Errorf("step %d: detailLevel = %v, want %v", i, m.detailLevel, expected)
		}
		m = sendKey(m, "i")
	}
	// After 4 presses, should wrap back to None
	if m.detailLevel != DetailNone {
		t.Errorf("after 4 presses: detailLevel = %v, want DetailNone", m.detailLevel)
	}
}

func TestUpdate_SearchKey_OpensSearchMode(t *testing.T) {
	m, _ := newModelWithDirs(t, "a")
	m2 := sendKey(m, "/")

	if m2.mode != ModeSearch {
		t.Errorf("after '/': mode = %v, want ModeSearch", m2.mode)
	}
}

func TestUpdate_Yank_SetsClipboard(t *testing.T) {
	m, root := newModelWithDirs(t, "mydir")
	m2 := sendKey(m, "y")

	if m2.clipboardPath != filepath.Join(root, "mydir") {
		t.Errorf("clipboardPath = %q, want %q", m2.clipboardPath, filepath.Join(root, "mydir"))
	}
	if m2.clipboardOp != ClipCopy {
		t.Errorf("clipboardOp = %v, want ClipCopy", m2.clipboardOp)
	}
}

func TestUpdate_Yank_SameItem_Clears(t *testing.T) {
	m, _ := newModelWithDirs(t, "mydir")
	m2 := sendKey(m, "y")  // yank
	m3 := sendKey(m2, "y") // yank same item again = clear

	if m3.clipboardPath != "" || m3.clipboardOp != ClipNone {
		t.Errorf("second yank on same item should clear clipboard, got path=%q op=%v",
			m3.clipboardPath, m3.clipboardOp)
	}
}

func TestUpdate_Cut_SetsClipboard(t *testing.T) {
	m, root := newModelWithDirs(t, "mydir")
	m2 := sendKey(m, "x")

	if m2.clipboardOp != ClipCut {
		t.Errorf("clipboardOp = %v, want ClipCut", m2.clipboardOp)
	}
	if m2.clipboardPath != filepath.Join(root, "mydir") {
		t.Errorf("clipboardPath wrong after cut")
	}
}

func TestUpdate_Cut_SameItem_Clears(t *testing.T) {
	m, _ := newModelWithDirs(t, "mydir")
	m2 := sendKey(m, "x")
	m3 := sendKey(m2, "x")

	if m3.clipboardOp != ClipNone || m3.clipboardPath != "" {
		t.Error("second cut on same item should clear clipboard")
	}
}

func TestUpdate_Paste_NoClipboard_NoOp(t *testing.T) {
	m, _ := newModelWithDirs(t, "a")
	m2 := sendKey(m, "p")

	if m2.mode != ModeNormal {
		t.Errorf("paste with no clipboard should stay ModeNormal, got %v", m2.mode)
	}
}

func TestUpdate_Paste_WithClipboard_OpensConfirm(t *testing.T) {
	m, _ := newModelWithDirs(t, "src", "dst")
	m.clipboardPath = "/some/path/file.txt"
	m.clipboardOp = ClipCopy

	m2 := sendKey(m, "p")
	if m2.mode != ModeConfirm {
		t.Errorf("paste with clipboard should open confirm, got %v", m2.mode)
	}
	if m2.confirmAction != ConfirmPasteCopy {
		t.Errorf("confirmAction = %v, want ConfirmPasteCopy", m2.confirmAction)
	}
}

func TestUpdate_Confirm_Y_ExecutesDelete(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "todelete")
	os.Mkdir(target, 0755)

	m := newModel(t, root)
	m.mode = ModeConfirm
	m.confirmAction = ConfirmDelete
	m.pendingPath = target

	m2 := sendKey(m, "y")

	if m2.mode == ModeConfirm {
		t.Error("confirm y should exit confirm mode")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Error("target should be deleted after confirm y")
	}
}

func TestUpdate_Confirm_N_Cancels(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "keepme")
	os.Mkdir(target, 0755)

	m := newModel(t, root)
	m.mode = ModeConfirm
	m.confirmAction = ConfirmDelete
	m.pendingPath = target

	m2 := sendKey(m, "n")

	if _, err := os.Stat(target); err != nil {
		t.Errorf("target should still exist after cancel, got: %v", err)
	}
	if m2.mode != ModeNormal {
		t.Errorf("mode should return to Normal after cancel, got %v", m2.mode)
	}
}

func TestUpdate_ErrorMode_AnyKeyDismisses(t *testing.T) {
	m, _ := newModelWithDirs(t, "a")
	m.mode = ModeError
	m.errorMsg = "something went wrong"

	m2 := sendKey(m, "x") // any key

	if m2.mode != ModeNormal {
		t.Errorf("any key in error mode should dismiss to Normal, got %v", m2.mode)
	}
	if m2.errorMsg != "" {
		t.Error("errorMsg should be cleared after dismiss")
	}
}

func TestUpdate_ConfirmPasteCopy_CopiesFile(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	dstDir := filepath.Join(root, "dst")
	os.Mkdir(srcDir, 0755)
	os.Mkdir(dstDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("hello"), 0644)

	m := newModel(t, root)
	m.mode = ModeConfirm
	m.confirmAction = ConfirmPasteCopy
	m.pendingPath = filepath.Join(srcDir, "file.txt")
	m.pendingDestDir = dstDir

	sendKey(m, "y")

	// Check copy exists in destination
	if _, err := os.Stat(filepath.Join(dstDir, "file.txt")); err != nil {
		t.Errorf("copied file not found in destination: %v", err)
	}
	// Original should still exist
	if _, err := os.Stat(filepath.Join(srcDir, "file.txt")); err != nil {
		t.Errorf("source file should still exist after copy: %v", err)
	}
}

func TestUpdate_ConfirmPasteMove_MovesFile(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	dstDir := filepath.Join(root, "dst")
	os.Mkdir(srcDir, 0755)
	os.Mkdir(dstDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("hello"), 0644)

	m := newModel(t, root)
	m.mode = ModeConfirm
	m.confirmAction = ConfirmPasteMove
	m.pendingPath = filepath.Join(srcDir, "file.txt")
	m.pendingDestDir = dstDir

	m2 := sendKey(m, "y")

	// File should be in destination
	if _, err := os.Stat(filepath.Join(dstDir, "file.txt")); err != nil {
		t.Errorf("moved file not found in destination: %v", err)
	}
	// Source should be gone
	if _, err := os.Stat(filepath.Join(srcDir, "file.txt")); !os.IsNotExist(err) {
		t.Error("source file should be removed after move")
	}
	// Clipboard should be cleared
	if m2.clipboardPath != "" || m2.clipboardOp != ClipNone {
		t.Error("clipboard should be cleared after move")
	}
}

// ─── parseSearchFlags (in app package) ───────────────────────────────────────

func TestParseSearchFlags_NoFlags(t *testing.T) {
	q, r, txt := parseSearchFlags("foo bar")
	if q != "foo bar" || r || txt {
		t.Errorf("got q=%q r=%v txt=%v", q, r, txt)
	}
}

func TestParseSearchFlags_RecursiveFlag(t *testing.T) {
	q, r, _ := parseSearchFlags("foo -r")
	if q != "foo" || !r {
		t.Errorf("got q=%q r=%v", q, r)
	}
}

func TestParseSearchFlags_TextFlag(t *testing.T) {
	q, _, txt := parseSearchFlags("-t foo")
	if q != "foo" || !txt {
		t.Errorf("got q=%q txt=%v", q, txt)
	}
}

func TestParseSearchFlags_CombinedRT(t *testing.T) {
	q, r, txt := parseSearchFlags("foo -rt")
	if q != "foo" || !r || !txt {
		t.Errorf("got q=%q r=%v txt=%v", q, r, txt)
	}
}

func TestParseSearchFlags_CombinedTR(t *testing.T) {
	q, r, txt := parseSearchFlags("foo -tr")
	if q != "foo" || !r || !txt {
		t.Errorf("got q=%q r=%v txt=%v", q, r, txt)
	}
}

func TestParseSearchFlags_SeparateRandT(t *testing.T) {
	q, r, txt := parseSearchFlags("foo -r -t")
	if q != "foo" || !r || !txt {
		t.Errorf("got q=%q r=%v txt=%v", q, r, txt)
	}
}

func TestParseSearchFlags_FlagInMiddle(t *testing.T) {
	q, r, txt := parseSearchFlags("hello -r world")
	if q != "hello world" || !r || txt {
		t.Errorf("got q=%q r=%v txt=%v", q, r, txt)
	}
}
