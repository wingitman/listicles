package app

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/wingitman/listicles/internal/config"
	"github.com/wingitman/listicles/internal/fs"
)

// ─── Test helpers ─────────────────────────────────────────────────────────────

// newViewModel builds a minimal Model ready for rendering tests.
// width/height are set to comfortable values.
func newViewModel(cfg *config.Config) Model {
	if cfg == nil {
		cfg = config.Default()
	}
	m := Model{
		cfg:    cfg,
		width:  80,
		height: 40,
		keys:   resolveKeys(cfg.Keybinds),
	}
	return m
}

// makeEntry creates a test fs.Entry.
func makeEntry(name string, isDir bool) fs.Entry {
	entType := fs.EntryFile
	if isDir {
		entType = fs.EntryDir
	}
	return fs.Entry{
		Name: name,
		Path: "/test/" + name,
		Type: entType,
		Size: 1024,
	}
}

// ─── clamp ────────────────────────────────────────────────────────────────────

func TestClamp(t *testing.T) {
	cases := []struct{ v, lo, hi, want int }{
		{5, 1, 10, 5},
		{0, 1, 10, 1},
		{11, 1, 10, 10},
		{-5, 0, 5, 0},
		{100, 0, 5, 5},
	}
	for _, c := range cases {
		got := clamp(c.v, c.lo, c.hi)
		if got != c.want {
			t.Errorf("clamp(%d,%d,%d) = %d, want %d", c.v, c.lo, c.hi, got, c.want)
		}
	}
}

// ─── renderHeader ─────────────────────────────────────────────────────────────

func TestRenderHeader_ShowsRootDir(t *testing.T) {
	m := newViewModel(nil)
	m.rootDir = "/home/wing/Work"
	out := m.renderHeader()
	if !strings.Contains(out, "/home/wing/Work") {
		t.Errorf("header should contain rootDir, got:\n%s", out)
	}
}

func TestRenderHeader_ShowsFilesBadge(t *testing.T) {
	m := newViewModel(nil)
	m.rootDir = "/tmp"
	m.listMode = ListDirsAndFiles
	out := m.renderHeader()
	if !strings.Contains(out, "[files]") {
		t.Errorf("header should show [files] badge, got:\n%s", out)
	}
}

func TestRenderHeader_ShowsHiddenBadge(t *testing.T) {
	m := newViewModel(nil)
	m.rootDir = "/tmp"
	m.showHidden = true
	out := m.renderHeader()
	if !strings.Contains(out, "[hidden]") {
		t.Errorf("header should show [hidden] badge, got:\n%s", out)
	}
}

func TestRenderHeader_ShowsDigitBuffer(t *testing.T) {
	m := newViewModel(nil)
	m.rootDir = "/tmp"
	m.digitBuffer = "19"
	out := m.renderHeader()
	if !strings.Contains(out, "19") {
		t.Errorf("header should show digit buffer '19', got:\n%s", out)
	}
}

func TestRenderHeader_NoVimBadge(t *testing.T) {
	// Vim mode is removed — [VIM] badge should never appear
	m := newViewModel(nil)
	m.rootDir = "/tmp"
	out := m.renderHeader()
	if strings.Contains(out, "[VIM]") {
		t.Errorf("header must not contain [VIM] badge (vim mode removed)")
	}
}

func TestRenderHeader_TruncatesLongPath(t *testing.T) {
	m := newViewModel(nil)
	m.width = 40
	m.rootDir = "/very/long/path/that/exceeds/the/terminal/width/definitely"
	out := m.renderHeader()
	// Should contain ellipsis
	if !strings.Contains(out, "…") {
		t.Errorf("long path should be truncated with …, got:\n%s", out)
	}
}

// ─── renderNode ───────────────────────────────────────────────────────────────

func TestRenderNode_NumberLabel_FocusedDepth(t *testing.T) {
	m := newViewModel(nil)
	m.width = 80
	node := TreeNode{Entry: makeEntry("mydir", true), Depth: 0}
	out := m.renderNode(0, node, 0, 0)
	if !strings.Contains(out, " 1 ") {
		t.Errorf("node at focused depth with siblingIdx=0 should show ' 1 ', got:\n%s", out)
	}
}

func TestRenderNode_NumberLabel_WrongDepth(t *testing.T) {
	m := newViewModel(nil)
	m.width = 80
	// Node at depth 1, but focusedDepth is 0 → should show dot
	node := TreeNode{Entry: makeEntry("child", true), Depth: 1}
	out := m.renderNode(0, node, 0, 0)
	if !strings.Contains(out, " · ") {
		t.Errorf("node not at focused depth should show ' · ', got:\n%s", out)
	}
}

func TestRenderNode_ExpandedDirIcon(t *testing.T) {
	m := newViewModel(nil)
	m.width = 80
	node := TreeNode{Entry: makeEntry("expanded", true), Depth: 0, Expanded: true}
	out := m.renderNode(0, node, 0, 0)
	if !strings.Contains(out, "▼") {
		t.Errorf("expanded dir should show ▼ icon, got:\n%s", out)
	}
}

func TestRenderNode_CollapsedDirIcon(t *testing.T) {
	m := newViewModel(nil)
	m.width = 80
	node := TreeNode{Entry: makeEntry("collapsed", true), Depth: 0, Expanded: false}
	out := m.renderNode(0, node, 0, 0)
	if !strings.Contains(out, "▶") {
		t.Errorf("collapsed dir should show ▶ icon, got:\n%s", out)
	}
}

func TestRenderNode_FileNoDirectionIcon(t *testing.T) {
	m := newViewModel(nil)
	m.width = 80
	node := TreeNode{Entry: makeEntry("file.txt", false), Depth: 0}
	out := m.renderNode(0, node, 0, 0)
	// Files should have neither ▶ nor ▼
	if strings.Contains(out, "▶") || strings.Contains(out, "▼") {
		t.Errorf("file node should not have direction icons, got:\n%s", out)
	}
}

func TestRenderNode_ClipboardTag_Copy(t *testing.T) {
	m := newViewModel(nil)
	m.width = 80
	entry := makeEntry("myfile.txt", false)
	m.clipboardPath = entry.Path
	m.clipboardOp = ClipCopy
	node := TreeNode{Entry: entry, Depth: 0}
	out := m.renderNode(0, node, 0, 0)
	if !strings.Contains(out, "[copy]") {
		t.Errorf("yanked item should show [copy], got:\n%s", out)
	}
}

func TestRenderNode_ClipboardTag_Cut(t *testing.T) {
	m := newViewModel(nil)
	m.width = 80
	entry := makeEntry("myfile.txt", false)
	m.clipboardPath = entry.Path
	m.clipboardOp = ClipCut
	node := TreeNode{Entry: entry, Depth: 0}
	out := m.renderNode(0, node, 0, 0)
	if !strings.Contains(out, "[cut]") {
		t.Errorf("cut item should show [cut], got:\n%s", out)
	}
}

func TestRenderNode_SelectedHighlight_PaddedToWidth(t *testing.T) {
	m := newViewModel(nil)
	m.width = 80
	m.cursor = 0
	node := TreeNode{Entry: makeEntry("selected", true), Depth: 0}
	out := m.renderNode(0, node, 0, 0)
	// Selected row should be padded to full width (lipgloss fills it)
	// We can't check exact ANSI, but we can verify the node renders without panic
	if out == "" {
		t.Error("selected node rendered as empty string")
	}
}

// ─── renderDetail ─────────────────────────────────────────────────────────────

func TestRenderDetail_None(t *testing.T) {
	m := newViewModel(nil)
	m.detailLevel = DetailNone
	out := m.renderDetail(makeEntry("x", false))
	if out != "" {
		t.Errorf("DetailNone should return empty, got %q", out)
	}
}

func TestRenderDetail_Size_File(t *testing.T) {
	m := newViewModel(nil)
	m.detailLevel = DetailSize
	e := fs.Entry{Name: "file", Path: "/tmp/file", Type: fs.EntryFile, Size: 2048}
	out := m.renderDetail(e)
	if out != "2.0 KB" {
		t.Errorf("DetailSize for 2048 bytes = %q, want 2.0 KB", out)
	}
}

func TestRenderDetail_FullPath(t *testing.T) {
	m := newViewModel(nil)
	m.detailLevel = DetailFullPath
	e := makeEntry("mydir", true)
	out := m.renderDetail(e)
	if out != e.Path {
		t.Errorf("DetailFullPath = %q, want %q", out, e.Path)
	}
}

// ─── renderStatusBar ─────────────────────────────────────────────────────────

func TestRenderStatusBar_ContainsConfiguredKeys(t *testing.T) {
	m := newViewModel(nil)
	m.width = 500 // wide enough not to truncate any hints
	m.nodes = []TreeNode{{Entry: makeEntry("dir", true), Depth: 0}}
	out := m.renderStatusBar()

	// All key hints should appear in Nano-style [key]Action format
	for _, hint := range []string{
		"[enter]", // confirm
		"[pgup/",  // page_up
		"pgdown]", // page_down
		"[home/",  // jump_top
		"end]",    // jump_bottom
		"[y]",     // yank
		"[x]",     // cut
		"[p]",     // paste
		"[Y]",     // copy_path
		"[f]",     // toggle_list
		"[i]",     // details
		"[a]",     // add
		"[d]",     // delete
		"[r]",     // rename
		"[e]",     // edit
		"[o]",     // options
	} {
		if !strings.Contains(out, hint) {
			t.Errorf("status bar missing key hint %q\nfull bar: %s", hint, out)
		}
	}
}

func TestRenderStatusBar_ShowsStatusMsg(t *testing.T) {
	m := newViewModel(nil)
	m.width = 80
	m.statusMsg = "path copied to clipboard"
	out := m.renderStatusBar()
	if !strings.Contains(out, "path copied to clipboard") {
		t.Errorf("status bar should show statusMsg, got:\n%s", out)
	}
}

func TestRenderStatusBar_ShowsListMode(t *testing.T) {
	m := newViewModel(nil)
	m.width = 400 // wide enough for all hints including new keybinds
	m.listMode = ListDirsAndFiles
	out := m.renderStatusBar()
	if !strings.Contains(out, "Files:all") {
		t.Errorf("status bar should show 'Files:all', got:\n%s", out)
	}

	m.listMode = ListDirsOnly
	out = m.renderStatusBar()
	if !strings.Contains(out, "Files:dirs") {
		t.Errorf("status bar should show 'Files:dirs', got:\n%s", out)
	}
}

func TestRenderStatusBar_NoVimHint(t *testing.T) {
	// vim mode toggle key was removed — "v vim" should not appear
	m := newViewModel(nil)
	m.width = 200
	out := m.renderStatusBar()
	if strings.Contains(out, " vim") {
		t.Errorf("status bar must not contain ' vim' hint (vim mode removed), got:\n%s", out)
	}
}

// ─── renderClipboardBar ───────────────────────────────────────────────────────

func TestRenderClipboardBar_Copy(t *testing.T) {
	m := newViewModel(nil)
	m.clipboardPath = "/home/wing/Documents/notes.txt"
	m.clipboardOp = ClipCopy
	out := m.renderClipboardBar()
	if !strings.Contains(out, "[copy]") {
		t.Errorf("clipboard bar should show [copy], got:\n%s", out)
	}
	if !strings.Contains(out, "notes.txt") {
		t.Errorf("clipboard bar should show filename, got:\n%s", out)
	}
}

func TestRenderClipboardBar_Cut(t *testing.T) {
	m := newViewModel(nil)
	m.clipboardPath = "/tmp/file.go"
	m.clipboardOp = ClipCut
	out := m.renderClipboardBar()
	if !strings.Contains(out, "[cut]") {
		t.Errorf("clipboard bar should show [cut], got:\n%s", out)
	}
}

func TestRenderClipboardBar_ShowsPasteKey(t *testing.T) {
	m := newViewModel(nil)
	m.clipboardPath = "/tmp/x"
	m.clipboardOp = ClipCopy
	out := m.renderClipboardBar()
	if !strings.Contains(out, "[p]Paste") { // default paste key in Nano-style format
		t.Errorf("clipboard bar should show '[p]Paste', got:\n%s", out)
	}
}

// ─── renderParentCrumbs ───────────────────────────────────────────────────────

func TestRenderParentCrumbs_ParentDepthZero(t *testing.T) {
	cfg := config.Default()
	cfg.Display.ParentDepth = 0
	m := newViewModel(cfg)
	m.rootDir = "/home/wing"
	out := m.renderParentCrumbs()
	if out != "" {
		t.Errorf("ParentDepth=0 should render nothing, got:\n%s", out)
	}
}

func TestRenderParentCrumbs_Depth1_AtRootLevel(t *testing.T) {
	cfg := config.Default()
	cfg.Display.ParentDepth = 1
	m := newViewModel(cfg)
	m.rootDir = "/home/wing/Work"
	// No nodes — cursor is at root level
	out := m.renderParentCrumbs()
	// Should show the parent of rootDir
	if !strings.Contains(out, "wing") && !strings.Contains(out, "home") {
		t.Errorf("crumbs should contain parent dir names, got:\n%s", out)
	}
}

func TestRenderParentCrumbs_Depth1_InsideTree(t *testing.T) {
	cfg := config.Default()
	cfg.Display.ParentDepth = 1
	m := newViewModel(cfg)
	m.rootDir = "/home/wing"

	parentEntry := makeEntry("Work", true)
	parentEntry.Path = "/home/wing/Work"
	childEntry := makeEntry("listicle", true)
	childEntry.Path = "/home/wing/Work/listicle"

	m.nodes = []TreeNode{
		{Entry: parentEntry, Depth: 0, Expanded: true},
		{Entry: childEntry, Depth: 1},
	}
	m.cursor = 1 // cursor on the depth-1 child

	out := m.renderParentCrumbs()
	// Should show "Work" as the immediate parent crumb
	if !strings.Contains(out, "Work") {
		t.Errorf("crumbs should show immediate parent 'Work', got:\n%s", out)
	}
}

// ─── renderSearchBar ─────────────────────────────────────────────────────────

func TestRenderSearchBar_ShowsHints(t *testing.T) {
	m := newViewModel(nil)
	m.mode = ModeSearch
	m.searchInputActive = true // typing state: shows search flags hint
	out := m.renderSearchBar()
	// Typing state hint: "[Enter]Search  [-r]Recursive  [-t]Content …"
	if !strings.Contains(out, "[Enter]Search") {
		t.Errorf("search bar should mention '[Enter]Search', got:\n%s", out)
	}
	if !strings.Contains(out, "-r") {
		t.Errorf("search bar should mention -r flag, got:\n%s", out)
	}
	if !strings.Contains(out, "-t") {
		t.Errorf("search bar should mention -t flag, got:\n%s", out)
	}
}

func TestRenderSearchBar_ShowsLiveMatchCount(t *testing.T) {
	m := newViewModel(nil)
	m.mode = ModeSearch
	m.searchLiveNodes = []TreeNode{
		{Entry: makeEntry("foo", true), Depth: 0},
		{Entry: makeEntry("foobar", true), Depth: 0},
	}
	out := m.renderSearchBar()
	if !strings.Contains(out, "2 match") {
		t.Errorf("search bar should show live match count, got:\n%s", out)
	}
}

func TestRenderSearchBar_ShowsToolBadge_Fd(t *testing.T) {
	m := newViewModel(nil)
	m.mode = ModeSearch
	m.searchTools.HasFd = true
	m.searchTextMode = false
	out := m.renderSearchBar()
	if !strings.Contains(out, "[fd]") {
		t.Errorf("search bar should show [fd] when fd is available, got:\n%s", out)
	}
}

// ─── renderSearchResultHeader ─────────────────────────────────────────────────

func TestRenderSearchResultHeader_NoResults(t *testing.T) {
	m := newViewModel(nil)
	m.mode = ModeSearchResult
	m.searchQuery = "xyz"
	m.nodes = nil
	out := m.renderSearchResultHeader()
	if !strings.Contains(out, "No results") {
		t.Errorf("should show 'No results', got:\n%s", out)
	}
	if !strings.Contains(out, "xyz") {
		t.Errorf("should show query 'xyz', got:\n%s", out)
	}
}

func TestRenderSearchResultHeader_WithResults(t *testing.T) {
	m := newViewModel(nil)
	m.mode = ModeSearchResult
	m.searchQuery = "main"
	// renderSearchResultHeader now reads m.searchLiveNodes
	m.searchLiveNodes = []TreeNode{
		{Entry: makeEntry("main.go", false), Depth: 0},
		{Entry: makeEntry("main_test.go", false), Depth: 0},
	}
	out := m.renderSearchResultHeader()
	if !strings.Contains(out, "2 result") {
		t.Errorf("should show result count, got:\n%s", out)
	}
	if !strings.Contains(out, "main") {
		t.Errorf("should show query, got:\n%s", out)
	}
}

// ─── View (smoke test) ────────────────────────────────────────────────────────

func TestView_RendersWithoutPanic(t *testing.T) {
	cfg := config.Default()
	m := newViewModel(cfg)
	m.rootDir = "/tmp"
	m.nodes = []TreeNode{
		{Entry: makeEntry("alpha", true), Depth: 0},
		{Entry: makeEntry("beta", true), Depth: 0},
	}
	// Should not panic
	out := m.View()
	if out == "" {
		t.Error("View() returned empty string")
	}
	if !strings.Contains(out, "alpha") {
		t.Errorf("View() should contain entry names, got:\n%s", out)
	}
}

func TestView_SearchMode_ShowsSearchBar(t *testing.T) {
	cfg := config.Default()
	m := newViewModel(cfg)
	m.rootDir = "/tmp"
	m.mode = ModeSearch

	// Need a real textinput for the search bar
	from, _ := New(cfg, "/tmp", "", "")
	from.width = 80
	from.height = 40
	from.mode = ModeSearch
	from.searchInputActive = true

	out := from.View()
	// In typing state the hint shows "[Enter]Search"
	if !strings.Contains(out, "[Enter]Search") {
		t.Errorf("ModeSearch should show search bar hint, got:\n%s", out)
	}
}

func TestView_ClipboardBar_AppearsWhenSet(t *testing.T) {
	cfg := config.Default()
	m := newViewModel(cfg)
	m.rootDir = "/tmp"
	m.clipboardPath = filepath.Join("/tmp", "file.txt")
	m.clipboardOp = ClipCopy
	m.nodes = []TreeNode{
		{Entry: makeEntry("dir", true), Depth: 0},
	}

	out := m.View()
	if !strings.Contains(out, "[copy]") {
		t.Errorf("View() should show clipboard bar when clipboard is set, got:\n%s", out)
	}
}
