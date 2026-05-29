package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wingitman/listicles/internal/config"
	"github.com/wingitman/listicles/internal/fs"
	"github.com/wingitman/listicles/internal/search"
	appupdate "github.com/wingitman/listicles/internal/update"
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
	cfg := config.Default()
	cfg.Display.DefaultListMode = "dirs"
	fm, err := New(cfg, root, "", "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
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

func TestGoToParentDir_AtDriveListRootIsNoOp(t *testing.T) {
	m, _ := newModelWithDirs(t, "a")
	m.rootDir = fs.DriveListRoot
	m.goToParentDir()
	if m.rootDir != fs.DriveListRoot {
		t.Fatalf("rootDir = %q, want drive list root", m.rootDir)
	}
}

func TestInitTree_DriveListRootShowsDrives(t *testing.T) {
	m, _ := newModelWithDirs(t, "a")
	if err := m.initTree(fs.DriveListRoot); err != nil {
		t.Fatalf("initTree drive list: %v", err)
	}
	if m.rootDir != fs.DriveListRoot {
		t.Fatalf("rootDir = %q, want drive list root", m.rootDir)
	}
	if m.nodes == nil {
		t.Fatal("drive list nodes should be an empty slice or entries, not nil")
	}
}

func TestDriveListSelectionOpensDrive(t *testing.T) {
	m, _ := newModelWithDirs(t, "a")
	drive := t.TempDir()
	m.rootDir = fs.DriveListRoot
	m.nodes = []TreeNode{{Entry: fs.Entry{Name: "T:", Path: drive, Type: fs.EntryDir}}}
	m.cursor = 0

	m2 := sendSpecialKey(m, tea.KeyEnter)
	if m2.rootDir != drive {
		t.Fatalf("rootDir = %q, want %q", m2.rootDir, drive)
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

func TestStartLiveZoxideSearch_DisabledClearsResults(t *testing.T) {
	m, _ := newModelWithDirs(t, "alpha")
	m.prevRootDir = m.rootDir
	m.searchTools.HasZoxide = false
	m.searchLiveNodes = []TreeNode{{Entry: fs.Entry{Name: "old", Path: "/old", Type: fs.EntryDir}}}
	m.textInput.SetValue("-z alpha")

	cmd := m.startLiveZoxideSearch()
	if cmd != nil {
		t.Fatal("disabled zoxide should not launch a search command")
	}
	if m.searchLiveNodes != nil {
		t.Fatal("disabled zoxide search should clear stale results")
	}
	if m.searchRunning {
		t.Fatal("disabled zoxide search should not remain running")
	}
}

func TestUpdate_StaleZoxideSearchResultIgnored(t *testing.T) {
	m, _ := newModelWithDirs(t, "alpha")
	m.mode = ModeSearch
	m.searchRequestID = 2

	updated, _ := m.Update(searchResultMsg{
		requestID: 1,
		zoxide:    true,
		results:   []search.Result{{Path: m.rootDir}},
	})
	m2 := updated.(Model)
	if len(m2.searchLiveNodes) != 0 {
		t.Fatal("stale zoxide results should not replace current results")
	}
}

func TestConfirmSearchSelection_DirectoryResultExpandsUnfiltered(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "scrimmy")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatalf("mkdir scrimmy: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "alpha"), 0755); err != nil {
		t.Fatalf("mkdir alpha: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("create notes.txt: %v", err)
	}

	m := newModel(t, root)
	m.mode = ModeSearch
	m.searchInputActive = false
	m.searchQuery = "scrimm"
	m.searchRecursive = false
	m.prevRootDir = root
	m.searchLiveNodes = []TreeNode{{
		Entry: fs.Entry{Name: "scrimmy", Path: dir, Type: fs.EntryDir},
	}}

	updated, cmd := m.confirmSearchSelection()
	if cmd != nil {
		t.Fatal("directory search result should expand in-place without command")
	}
	m2 := updated.(Model)
	if len(m2.searchLiveNodes) != 3 {
		t.Fatalf("expected directory plus two unfiltered children, got %d", len(m2.searchLiveNodes))
	}
	if !m2.searchLiveNodes[0].Expanded {
		t.Fatal("directory result should be marked expanded")
	}
	if m2.cursor != 1 {
		t.Fatalf("cursor = %d, want first child index 1", m2.cursor)
	}
	if got := m2.searchLiveNodes[1].Entry.Name; got != "alpha" {
		t.Fatalf("first child = %q, want alpha", got)
	}
	if got := m2.searchLiveNodes[2].Entry.Name; got != "notes.txt" {
		t.Fatalf("second child = %q, want notes.txt", got)
	}
}

func TestUpdate_SearchNavigationRightExpandsDirectoryResult(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "scrimmy")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatalf("mkdir scrimmy: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "child"), 0755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}

	m := newModel(t, root)
	m.mode = ModeSearch
	m.searchInputActive = false
	m.prevRootDir = root
	m.searchLiveNodes = []TreeNode{{
		Entry: fs.Entry{Name: "scrimmy", Path: dir, Type: fs.EntryDir},
	}}

	m2 := sendSpecialKey(m, tea.KeyRight)
	if len(m2.searchLiveNodes) != 2 {
		t.Fatalf("expected directory plus child, got %d", len(m2.searchLiveNodes))
	}
	if got := m2.searchLiveNodes[1].Entry.Name; got != "child" {
		t.Fatalf("child = %q, want child", got)
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
	if m.listMode != ListDirsAndFiles {
		t.Fatal("initial listMode should be ListDirsAndFiles")
	}

	m2 := sendKey(m, "f")
	if m2.listMode != ListDirsOnly {
		t.Error("after f: should be ListDirsOnly")
	}

	m3 := sendKey(m2, "f")
	if m3.listMode != ListDirsAndFiles {
		t.Error("after second f: should be ListDirsAndFiles")
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
	levels := []DetailLevel{
		DetailNone, DetailCount, DetailSize, DetailFullPath,
		DetailModTime, DetailBirthTime, DetailPermissions, DetailOwner, DetailMimeType,
	}

	for i, expected := range levels {
		if m.detailLevel != expected {
			t.Errorf("step %d: detailLevel = %v, want %v", i, m.detailLevel, expected)
		}
		m = sendKey(m, "i")
	}
	// After 9 presses, should wrap back to None
	if m.detailLevel != DetailNone {
		t.Errorf("after 9 presses: detailLevel = %v, want DetailNone", m.detailLevel)
	}
}

func TestUpdate_SearchKey_OpensSearchMode(t *testing.T) {
	m, _ := newModelWithDirs(t, "a")
	m2 := sendKey(m, "/")

	if m2.mode != ModeSearch {
		t.Errorf("after '/': mode = %v, want ModeSearch", m2.mode)
	}
}

func TestUpdate_UpdateCheckMsgShowsPrompt(t *testing.T) {
	m, _ := newModelWithDirs(t, "a")
	updated, _ := m.Update(updateCheckMsg{info: appupdate.Info{
		RepoPath:      t.TempDir(),
		Branch:        "feature",
		Upstream:      "origin/feature",
		CurrentCommit: "1111111",
		LatestCommit:  "2222222",
		Available: []appupdate.Commit{{
			Hash:    "2222222",
			Short:   "2222222",
			Subject: "add updates",
			Body:    "details",
		}},
	}})
	m2 := updated.(Model)

	if m2.mode != ModeUpdatePrompt {
		t.Fatalf("mode = %v, want ModeUpdatePrompt", m2.mode)
	}
	if m2.updateCursor != 0 {
		t.Fatalf("updateCursor = %d, want 0", m2.updateCursor)
	}
}

func TestUpdate_ShowUpdatesKeyOpensUpdatesMode(t *testing.T) {
	m, _ := newModelWithDirs(t, "a")
	m.updateInfo = appupdate.Info{RepoPath: t.TempDir()}
	m2 := sendKey(m, "U")

	if m2.mode != ModeUpdates {
		t.Fatalf("mode = %v, want ModeUpdates", m2.mode)
	}
}

func TestUpdate_PluginsKeyOpensPluginsMode(t *testing.T) {
	m, _ := newModelWithDirs(t, "a")
	m2 := sendKey(m, "P")

	if m2.mode != ModePlugins {
		t.Fatalf("mode = %v, want ModePlugins", m2.mode)
	}
}

func TestToggleSelectedPlugin_PersistsConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", filepath.Join(t.TempDir(), "AppData", "Roaming"))
	m, _ := newModelWithDirs(t, "a")
	m.installedTools.HasZoxide = true
	m.searchTools = toolsForConfig(m.installedTools, m.cfg)
	m.pluginCursor = 2

	if err := m.toggleSelectedPlugin(); err != nil {
		t.Fatalf("toggleSelectedPlugin: %v", err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Plugins.Zoxide {
		t.Fatal("zoxide plugin should be disabled in persisted config")
	}
	if m.searchTools.HasZoxide {
		t.Fatal("zoxide should no longer be active after disabling")
	}
}

func TestApplyConfig_UpdatesLiveKeybindsAndDisplay(t *testing.T) {
	m, _ := newModelWithDirs(t, "visible")

	cfg := config.Default()
	cfg.Keybinds.Up = "k"
	cfg.Keybinds.Down = "j"
	cfg.Display.ShowHidden = true
	cfg.Display.DefaultListMode = "dirs"

	if err := m.applyConfig(cfg); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}

	if m.keys.up != "k" || m.keys.down != "j" {
		t.Fatalf("keybinds not reloaded: up=%q down=%q", m.keys.up, m.keys.down)
	}
	if !m.showHidden {
		t.Fatal("showHidden should be updated from reloaded config")
	}
	if m.listMode != ListDirsOnly {
		t.Fatalf("listMode = %v, want ListDirsOnly", m.listMode)
	}
	if m.statusMsg != "" {
		t.Fatalf("applyConfig should not set status directly, got %q", m.statusMsg)
	}
}

func TestUpdate_ReloadConfigMsgReloadsConfigFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.MkdirAll(config.ConfigDir(), 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	content := `
[keybinds]
up = "k"
down = "j"

[display]
show_hidden = true
default_list_mode = "dirs"
search_max_results = 20
parent_depth = 1

[apps]
editor = ""
opener = ""
`
	if err := os.WriteFile(config.ConfigPath(), []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	m, _ := newModelWithDirs(t, "visible")
	updated, _ := m.Update(reloadConfigMsg{})
	m2 := updated.(Model)

	if m2.keys.up != "k" || m2.keys.down != "j" {
		t.Fatalf("config reload did not update keys: up=%q down=%q", m2.keys.up, m2.keys.down)
	}
	if !m2.showHidden || m2.listMode != ListDirsOnly {
		t.Fatalf("config reload did not update display settings: showHidden=%v listMode=%v", m2.showHidden, m2.listMode)
	}
	if m2.statusMsg != "config reloaded" {
		t.Fatalf("statusMsg = %q, want config reloaded", m2.statusMsg)
	}
}

func TestUpdate_EnterFileOpensEditorNotExplorer(t *testing.T) {
	if _, err := exec.LookPath("true"); err != nil {
		t.Skip("true command not available")
	}

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	cfg := config.Default()
	cfg.Display.DefaultListMode = "dirs_and_files"
	cfg.Apps.Editor = "true"
	cfg.Apps.Opener = filepath.Join(root, "missing-opener")
	m, err := New(cfg, root, "", "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on file should return an editor command")
	}
	msg := cmd()
	if err, ok := msg.(errorMsg); ok {
		t.Fatalf("enter on file should use the configured editor, not the opener: %s", err)
	}
}

func TestConfirmSearchSelection_TextMatchWritesOpenFileWithLine(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	if err := os.WriteFile(path, []byte("one\ntwo\n"), 0644); err != nil {
		t.Fatalf("create file: %v", err)
	}
	openFile := filepath.Join(root, "open-target")
	m := newModel(t, root)
	m.openFile = openFile
	m.mode = ModeSearch
	m.searchLiveNodes = []TreeNode{{
		Entry:        fs.Entry{Name: "file.txt", Path: path, Type: fs.EntryFile},
		IsTextMatch:  true,
		MatchLineNum: 2,
	}}

	_, cmd := m.confirmSearchSelection()
	if cmd == nil {
		t.Fatal("text match should return an open-file command")
	}
	_ = cmd()
	target, err := os.ReadFile(openFile)
	if err != nil {
		t.Fatalf("read open target: %v", err)
	}
	want := path + "\n2\n"
	if string(target) != want {
		t.Fatalf("open target = %q, want %q", string(target), want)
	}
}

func TestEditorArgs_WithLineNumber(t *testing.T) {
	path := `D:\Projects\listicles\main.go`
	cases := []struct {
		editor string
		want   []string
	}{
		{"nvim", []string{"+23", path}},
		{"code", []string{"--goto", path + ":23"}},
		{"notepad.exe", []string{path}},
	}
	for _, c := range cases {
		got := editorArgs(c.editor, path, 23)
		if len(got) != len(c.want) {
			t.Fatalf("editorArgs(%q) len = %d, want %d: %v", c.editor, len(got), len(c.want), got)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("editorArgs(%q)[%d] = %q, want %q", c.editor, i, got[i], c.want[i])
			}
		}
	}
}

func TestOpenDefaultCmd_CustomOpenerDoesNotWaitForGuiProcess(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh command not available")
	}

	root := t.TempDir()
	target := filepath.Join(root, "target")
	script := filepath.Join(root, "opener.sh")
	done := target + ".done"
	content := "#!/bin/sh\nsleep 1\n: > \"$1.done\"\n"
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatalf("write opener script: %v", err)
	}

	cmd := openDefaultCmd(target, script)
	start := time.Now()
	if _, ok := cmd().(reloadMsg); !ok {
		t.Fatal("custom opener should return reloadMsg after starting")
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("custom opener blocked for %v", elapsed)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(done); err == nil {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("custom opener process did not continue after listicles returned")
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

func TestUpdate_Paste_WithClipboard_OpensInput(t *testing.T) {
	m, _ := newModelWithDirs(t, "src", "dst")
	m.clipboardPath = "/some/path/file.txt"
	m.clipboardOp = ClipCopy

	m2 := sendKey(m, "p")
	if m2.mode != ModeInput {
		t.Errorf("paste with clipboard should open input, got %v", m2.mode)
	}
	if m2.inputAction != InputPasteCopy {
		t.Errorf("inputAction = %v, want InputPasteCopy", m2.inputAction)
	}
	if m2.textInput.Value() != "file.txt" {
		t.Errorf("textInput = %q, want file.txt", m2.textInput.Value())
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
	m.pendingName = "file.txt"

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
	m.pendingName = "file.txt"

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
	q, r, txt, z := parseSearchFlags("foo bar")
	if q != "foo bar" || r || txt || z {
		t.Errorf("got q=%q r=%v txt=%v z=%v", q, r, txt, z)
	}
}

func TestParseSearchFlags_RecursiveFlag(t *testing.T) {
	q, r, _, _ := parseSearchFlags("foo -r")
	if q != "foo" || !r {
		t.Errorf("got q=%q r=%v", q, r)
	}
}

func TestParseSearchFlags_TextFlag(t *testing.T) {
	q, _, txt, _ := parseSearchFlags("-t foo")
	if q != "foo" || !txt {
		t.Errorf("got q=%q txt=%v", q, txt)
	}
}

func TestParseSearchFlags_CombinedRT(t *testing.T) {
	q, r, txt, _ := parseSearchFlags("foo -rt")
	if q != "foo" || !r || !txt {
		t.Errorf("got q=%q r=%v txt=%v", q, r, txt)
	}
}

func TestParseSearchFlags_CombinedTR(t *testing.T) {
	q, r, txt, _ := parseSearchFlags("foo -tr")
	if q != "foo" || !r || !txt {
		t.Errorf("got q=%q r=%v txt=%v", q, r, txt)
	}
}

func TestParseSearchFlags_SeparateRandT(t *testing.T) {
	q, r, txt, _ := parseSearchFlags("foo -r -t")
	if q != "foo" || !r || !txt {
		t.Errorf("got q=%q r=%v txt=%v", q, r, txt)
	}
}

func TestParseSearchFlags_FlagInMiddle(t *testing.T) {
	q, r, txt, _ := parseSearchFlags("hello -r world")
	if q != "hello world" || !r || txt {
		t.Errorf("got q=%q r=%v txt=%v", q, r, txt)
	}
}

func TestParseSearchFlags_ZoxideIgnoresRecursiveAndText(t *testing.T) {
	q, r, txt, z := parseSearchFlags("project -rtz")
	if q != "project" || r || txt || !z {
		t.Errorf("got q=%q r=%v txt=%v z=%v", q, r, txt, z)
	}
}
