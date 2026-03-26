package app

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/wingitman/listicles/internal/config"
	"github.com/wingitman/listicles/internal/fs"
	"github.com/wingitman/listicles/internal/search"
)

// ─── Types ────────────────────────────────────────────────────────────────────

type Mode int

const (
	ModeNormal       Mode = iota
	ModeConfirm           // waiting for y/n
	ModeInput             // text input (add/rename)
	ModeError             // showing an error
	ModeSearch            // live filter bar
	ModeSearchResult      // committed search results
)

type InputAction int

const (
	InputAdd    InputAction = iota
	InputRename InputAction = iota
)

type DetailLevel int

const (
	DetailNone DetailLevel = iota
	DetailCount
	DetailSize
	DetailFullPath
)

type ListMode int

const (
	ListDirsOnly     ListMode = iota
	ListDirsAndFiles ListMode = iota
)

type ConfirmAction int

const (
	ConfirmDelete    ConfirmAction = iota
	ConfirmRename    ConfirmAction = iota
	ConfirmPasteCopy ConfirmAction = iota
	ConfirmPasteMove ConfirmAction = iota
)

// ClipOp is the type of pending clipboard operation.
type ClipOp int

const (
	ClipNone ClipOp = iota
	ClipCopy ClipOp = iota
	ClipCut  ClipOp = iota
)

// TreeNode is one visible row in the tree.
type TreeNode struct {
	Entry    fs.Entry
	Depth    int
	Expanded bool
}

// ─── Model ────────────────────────────────────────────────────────────────────

type Model struct {
	cfg      *config.Config
	cdFile   string
	openFile string

	// Tree state
	rootDir string
	nodes   []TreeNode
	cursor  int
	offset  int

	// Terminal size
	width  int
	height int

	// Modes
	mode          Mode
	inputAction   InputAction
	confirmAction ConfirmAction
	listMode      ListMode
	detailLevel   DetailLevel
	showHidden    bool

	// Text input (add / rename)
	textInput      textinput.Model
	pendingPath    string
	pendingDestDir string // destination for paste confirm
	pendingName    string
	confirmMsg     string

	// Error
	errorMsg string

	// Multi-digit navigation buffer
	digitBuffer string

	// Clipboard (filesystem yank/cut/paste)
	clipboardPath string
	clipboardOp   ClipOp

	// Transient status message (e.g. "path copied")
	statusMsg string

	// Search state
	searchTools     search.Tools
	searchRecursive bool
	searchTextMode  bool
	searchQuery     string
	searchResults   []search.Result
	searchRunning   bool
	prevNodes       []TreeNode
	prevRootDir     string
	searchLiveNodes []TreeNode

	keys resolvedKeys
}

// ─── Keybinds ─────────────────────────────────────────────────────────────────

type resolvedKeys struct {
	up           string
	down         string
	left         string
	right        string
	confirm      string
	parent       string
	pageUp       string
	pageDown     string
	jumpTop      string
	jumpBottom   string
	options      string
	add          string
	delete       string
	toggleList   string
	rename       string
	edit         string
	yank         string
	cut          string
	paste        string
	copyPath     string
	quit         string
	details      string
	toggleHidden string
	searchKey    string
}

func resolveKeys(k config.Keybinds) resolvedKeys {
	return resolvedKeys{
		up:           k.Up,
		down:         k.Down,
		left:         k.Left,
		right:        k.Right,
		confirm:      k.Confirm,
		parent:       k.Parent,
		pageUp:       k.PageUp,
		pageDown:     k.PageDown,
		jumpTop:      k.JumpTop,
		jumpBottom:   k.JumpBottom,
		options:      k.Options,
		add:          k.Add,
		delete:       k.Delete,
		toggleList:   k.ToggleList,
		rename:       k.Rename,
		edit:         k.Edit,
		yank:         k.Yank,
		cut:          k.Cut,
		paste:        k.Paste,
		copyPath:     k.CopyPath,
		quit:         k.Quit,
		details:      k.Details,
		toggleHidden: k.ToggleHidden,
		searchKey:    k.Search,
	}
}

// ─── Constructor ─────────────────────────────────────────────────────────────

func New(cfg *config.Config, startDir string, cdFile string, openFile string) (*Model, error) {
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			startDir = "/"
		}
	}

	ti := textinput.New()
	ti.CharLimit = 256

	listMode := ListDirsOnly
	if cfg.Display.DefaultListMode == "dirs_and_files" {
		listMode = ListDirsAndFiles
	}

	m := &Model{
		cfg:         cfg,
		cdFile:      cdFile,
		openFile:    openFile,
		rootDir:     startDir,
		listMode:    listMode,
		showHidden:  cfg.Display.ShowHidden,
		textInput:   ti,
		keys:        resolveKeys(cfg.Keybinds),
		searchTools: search.DetectTools(),
	}

	if err := m.initTree(startDir); err != nil {
		return nil, err
	}
	return m, nil
}

// ─── Tree helpers ─────────────────────────────────────────────────────────────

func (m *Model) initTree(dir string) error {
	entries, err := fs.ScanDir(dir, m.showHidden, m.listMode == ListDirsAndFiles)
	if err != nil {
		return err
	}
	m.rootDir = dir
	m.nodes = make([]TreeNode, len(entries))
	for i, e := range entries {
		m.nodes[i] = TreeNode{Entry: e, Depth: 0}
	}
	m.cursor = 0
	m.offset = 0
	return nil
}

func (m *Model) expandNode(idx int) error {
	if idx < 0 || idx >= len(m.nodes) {
		return nil
	}
	node := &m.nodes[idx]
	if !node.Entry.IsDir() {
		return nil
	}
	if node.Expanded {
		m.collapseNode(idx)
		return nil
	}
	entries, err := fs.ScanDir(node.Entry.Path, m.showHidden, m.listMode == ListDirsAndFiles)
	if err != nil {
		return err
	}
	childDepth := node.Depth + 1
	children := make([]TreeNode, len(entries))
	for i, e := range entries {
		children[i] = TreeNode{Entry: e, Depth: childDepth}
	}
	after := make([]TreeNode, 0, len(m.nodes)+len(children))
	after = append(after, m.nodes[:idx+1]...)
	after = append(after, children...)
	after = append(after, m.nodes[idx+1:]...)
	m.nodes = after
	m.nodes[idx].Expanded = true
	return nil
}

func (m *Model) collapseNode(idx int) {
	if idx < 0 || idx >= len(m.nodes) {
		return
	}
	if !m.nodes[idx].Expanded {
		return
	}
	targetDepth := m.nodes[idx].Depth
	end := idx + 1
	for end < len(m.nodes) && m.nodes[end].Depth > targetDepth {
		end++
	}
	m.nodes = append(m.nodes[:idx+1], m.nodes[end:]...)
	m.nodes[idx].Expanded = false
}

func (m *Model) refreshExpandedNode(idx int) error {
	if idx < 0 || idx >= len(m.nodes) {
		return nil
	}
	if !m.nodes[idx].Expanded {
		return m.expandNode(idx)
	}
	m.collapseNode(idx)
	return m.expandNode(idx)
}

func (m *Model) findNodeByPath(p string) int {
	for i, n := range m.nodes {
		if n.Entry.Path == p {
			return i
		}
	}
	return -1
}

func (m *Model) parentNodeIdx(idx int) int {
	if idx < 0 || idx >= len(m.nodes) {
		return -1
	}
	childDepth := m.nodes[idx].Depth
	if childDepth == 0 {
		return -1
	}
	for i := idx - 1; i >= 0; i-- {
		if m.nodes[i].Depth == childDepth-1 {
			return i
		}
	}
	return -1
}

func (m *Model) currentOperationDir() string {
	e := m.selectedEntry()
	if e == nil {
		return m.rootDir
	}
	if e.IsDir() {
		return e.Path
	}
	return filepath.Dir(e.Path)
}

func (m *Model) rebuildTree() error {
	expanded := map[string]bool{}
	for _, n := range m.nodes {
		if n.Expanded {
			expanded[n.Entry.Path] = true
		}
	}
	if err := m.initTree(m.rootDir); err != nil {
		return err
	}
	for i := 0; i < len(m.nodes); i++ {
		if m.nodes[i].Entry.IsDir() && expanded[m.nodes[i].Entry.Path] {
			_ = m.expandNode(i)
		}
	}
	return nil
}

// ─── Standard helpers ─────────────────────────────────────────────────────────

func (m Model) Init() tea.Cmd { return nil }

func (m *Model) selectedEntry() *fs.Entry {
	if len(m.nodes) == 0 || m.cursor >= len(m.nodes) {
		return nil
	}
	return &m.nodes[m.cursor].Entry
}

func (m *Model) selectedNode() *TreeNode {
	if len(m.nodes) == 0 || m.cursor >= len(m.nodes) {
		return nil
	}
	return &m.nodes[m.cursor]
}

func (m *Model) visibleRows() int {
	reserved := 6
	if m.cfg != nil {
		reserved += m.cfg.Display.ParentDepth + 1
	}
	if m.mode == ModeConfirm || m.mode == ModeInput || m.mode == ModeError {
		reserved += 6
	}
	if m.mode == ModeSearch {
		reserved += 3
	}
	if m.clipboardPath != "" {
		reserved += 1
	}
	rows := m.height - reserved
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m *Model) clampCursor() {
	if m.cursor < 0 {
		m.cursor = 0
	}
	n := len(m.nodes)
	if m.mode == ModeSearch {
		n = len(m.searchLiveNodes)
	}
	if n > 0 && m.cursor >= n {
		m.cursor = n - 1
	}
}

func (m *Model) adjustOffset() {
	rows := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+rows {
		m.offset = m.cursor - rows + 1
	}
}

func (m *Model) exitWithDir(dir string) tea.Cmd {
	if m.cdFile != "" {
		_ = os.WriteFile(m.cdFile, []byte(dir), 0600)
	}
	return tea.Quit
}

// exitWithFile writes the selected file path to openFile (for editor
// integrations) and the file's parent directory to cdFile (for shell cd).
func (m *Model) exitWithFile(path string) tea.Cmd {
	if m.openFile != "" {
		_ = os.WriteFile(m.openFile, []byte(path), 0600)
	}
	if m.cdFile != "" {
		_ = os.WriteFile(m.cdFile, []byte(filepath.Dir(path)), 0600)
	}
	return tea.Quit
}

func (m *Model) exitWithoutCD() tea.Cmd { return tea.Quit }

func matchKey(pressed, binding string) bool {
	return pressed == binding
}

// calcPageJump returns a logarithmically scaled page jump size.
func calcPageJump(n int) int {
	if n <= 1 {
		return 1
	}
	j := int(math.Round(math.Log2(float64(n))))
	if j < 1 {
		j = 1
	}
	return j
}

// ─── Update ───────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case errorMsg:
		m.errorMsg = string(msg)
		m.mode = ModeError
		return m, nil

	case reloadMsg:
		if msg.reloadPath != "" {
			idx := m.findNodeByPath(msg.reloadPath)
			if idx >= 0 {
				_ = m.refreshExpandedNode(idx)
			} else {
				_ = m.initTree(m.rootDir)
			}
		}
		return m, nil

	case searchResultMsg:
		m.searchRunning = false
		m.searchResults = msg.results
		entries := search.ResultsToEntries(msg.results)
		m.nodes = make([]TreeNode, len(entries))
		for i, e := range entries {
			m.nodes[i] = TreeNode{Entry: e, Depth: 0}
		}
		m.cursor = 0
		m.offset = 0
		m.mode = ModeSearchResult
		return m, nil

	case digitTimeoutMsg:
		if m.digitBuffer != "" {
			m.resolveDigitBuffer()
			m.digitBuffer = ""
		}
		return m, nil

	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil

	case tea.KeyMsg:
		key := msg.String()

		// ── Error: any key dismisses ──────────────────────────────────────
		if m.mode == ModeError {
			m.mode = ModeNormal
			m.errorMsg = ""
			return m, nil
		}

		// ── Confirm (y/n) ─────────────────────────────────────────────────
		if m.mode == ModeConfirm {
			switch key {
			case "y", "Y":
				return m.executeConfirmedAction()
			default:
				m.mode = ModeNormal
				m.confirmMsg = ""
			}
			return m, nil
		}

		// ── Text input (add / rename) ─────────────────────────────────────
		if m.mode == ModeInput {
			switch key {
			case "enter":
				return m.submitInput()
			case "esc":
				m.mode = ModeNormal
				m.textInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}

		// ── Search input (live filter) ────────────────────────────────────
		if m.mode == ModeSearch {
			switch key {
			case "enter":
				return m.executeSearch()
			case "esc":
				m.nodes = m.prevNodes
				m.rootDir = m.prevRootDir
				m.cursor = 0
				m.offset = 0
				m.searchLiveNodes = nil
				m.mode = ModeNormal
				m.textInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				m.applyLiveFilter()
				return m, cmd
			}
		}

		// ── Search result mode ────────────────────────────────────────────
		if m.mode == ModeSearchResult {
			switch {
			case key == "esc" || matchKey(key, m.keys.quit):
				m.nodes = m.prevNodes
				m.rootDir = m.prevRootDir
				m.cursor = 0
				m.offset = 0
				m.mode = ModeNormal
				return m, nil
			case matchKey(key, m.keys.up):
				m.cursor--
				m.clampCursor()
				m.adjustOffset()
				return m, nil
			case matchKey(key, m.keys.down):
				m.cursor++
				m.clampCursor()
				m.adjustOffset()
				return m, nil
			case matchKey(key, m.keys.confirm):
				e := m.selectedEntry()
				if e == nil {
					return m, nil
				}
				if e.IsDir() {
					if err := m.initTree(e.Path); err != nil {
						m.errorMsg = err.Error()
						m.mode = ModeError
					} else {
						m.mode = ModeNormal
						m.exitWithDir(e.Path)
					}
				} else {
					if m.openFile != "" {
						return m, m.exitWithFile(e.Path)
					}
					return m, m.exitWithDir(filepath.Dir(e.Path))
				}
				return m, nil
			case key == "/" || matchKey(key, m.keys.searchKey):
				return m.openSearchInput()
			}
			return m, nil
		}

		// ── Normal mode ───────────────────────────────────────────────────

		// Search
		if key == "/" || matchKey(key, m.keys.searchKey) {
			return m.openSearchInput()
		}

		// Quit
		if matchKey(key, m.keys.quit) || key == "esc" {
			return m, m.exitWithoutCD()
		}

		// Confirm / Enter: cd into dir, or open file in editor/default app
		if matchKey(key, m.keys.confirm) {
			e := m.selectedEntry()
			if e != nil && e.IsDir() {
				return m, m.exitWithDir(e.Path)
			} else if e != nil {
				if m.openFile != "" {
					return m, m.exitWithFile(e.Path)
				}
				return m, openDefaultCmd(e.Path, m.cfg.Apps.Opener)
			}
			return m, m.exitWithDir(m.rootDir)
		}

		// Navigate up / down
		if matchKey(key, m.keys.up) {
			m.cursor--
			m.clampCursor()
			m.adjustOffset()
			return m, nil
		}
		if matchKey(key, m.keys.down) {
			m.cursor++
			m.clampCursor()
			m.adjustOffset()
			return m, nil
		}

		// Page up / down
		if matchKey(key, m.keys.pageUp) {
			m.cursor -= calcPageJump(len(m.nodes))
			m.clampCursor()
			m.adjustOffset()
			return m, nil
		}
		if matchKey(key, m.keys.pageDown) {
			m.cursor += calcPageJump(len(m.nodes))
			m.clampCursor()
			m.adjustOffset()
			return m, nil
		}

		// Jump to top / bottom
		if matchKey(key, m.keys.jumpTop) {
			m.cursor = 0
			m.offset = 0
			return m, nil
		}
		if matchKey(key, m.keys.jumpBottom) {
			m.cursor = len(m.nodes) - 1
			m.clampCursor()
			m.adjustOffset()
			return m, nil
		}

		// Expand / navigate right
		if matchKey(key, m.keys.right) {
			node := m.selectedNode()
			if node != nil && node.Entry.IsDir() {
				wasExpanded := node.Expanded
				parentDepth := node.Depth
				if err := m.expandNode(m.cursor); err != nil {
					m.errorMsg = err.Error()
					m.mode = ModeError
				} else if !wasExpanded {
					if m.cursor+1 < len(m.nodes) && m.nodes[m.cursor+1].Depth > parentDepth {
						m.cursor++
						m.adjustOffset()
					}
				}
			}
			return m, nil
		}

		// Collapse / navigate left
		if matchKey(key, m.keys.left) || matchKey(key, m.keys.parent) {
			m.navigateLeft()
			return m, nil
		}

		// Digit keys — type-ahead multi-digit navigation
		if len(key) == 1 && key >= "0" && key <= "9" {
			// bare "0" with empty buffer = go to parent immediately
			if key == "0" && m.digitBuffer == "" {
				m.navigateLeft()
				return m, nil
			}
			m.digitBuffer += key
			return m, tea.Tick(500*time.Millisecond, func(_ time.Time) tea.Msg {
				return digitTimeoutMsg{}
			})
		}

		// Toggle list mode
		if matchKey(key, m.keys.toggleList) {
			if m.listMode == ListDirsOnly {
				m.listMode = ListDirsAndFiles
			} else {
				m.listMode = ListDirsOnly
			}
			_ = m.rebuildTree()
			return m, nil
		}

		// Toggle hidden
		if matchKey(key, m.keys.toggleHidden) {
			m.showHidden = !m.showHidden
			_ = m.rebuildTree()
			return m, nil
		}

		// Toggle detail level
		if matchKey(key, m.keys.details) {
			m.detailLevel = (m.detailLevel + 1) % 4
			return m, nil
		}

		// Add file/folder
		if matchKey(key, m.keys.add) {
			m.inputAction = InputAdd
			m.textInput.Reset()
			m.textInput.Placeholder = "name  (end with / to create a directory)"
			m.textInput.SetValue("")
			m.textInput.Focus()
			m.mode = ModeInput
			return m, textinput.Blink
		}

		// Delete
		if matchKey(key, m.keys.delete) {
			e := m.selectedEntry()
			if e != nil {
				m.confirmAction = ConfirmDelete
				m.pendingPath = e.Path
				entryType := "file"
				if e.IsDir() {
					entryType = "directory"
				}
				m.confirmMsg = fmt.Sprintf("Delete %s %q?\n  This cannot be undone.", entryType, e.Path)
				m.mode = ModeConfirm
			}
			return m, nil
		}

		// Rename
		if matchKey(key, m.keys.rename) {
			e := m.selectedEntry()
			if e != nil {
				m.inputAction = InputRename
				m.pendingPath = e.Path
				m.textInput.Reset()
				m.textInput.Placeholder = e.Name
				m.textInput.SetValue(e.Name)
				m.textInput.Focus()
				m.mode = ModeInput
				return m, textinput.Blink
			}
			return m, nil
		}

		// Yank (filesystem copy)
		if matchKey(key, m.keys.yank) {
			e := m.selectedEntry()
			if e != nil {
				if m.clipboardPath == e.Path && m.clipboardOp == ClipCopy {
					// pressing y again on same item clears
					m.clipboardPath = ""
					m.clipboardOp = ClipNone
				} else {
					m.clipboardPath = e.Path
					m.clipboardOp = ClipCopy
				}
			}
			return m, nil
		}

		// Cut
		if matchKey(key, m.keys.cut) {
			e := m.selectedEntry()
			if e != nil {
				if m.clipboardPath == e.Path && m.clipboardOp == ClipCut {
					m.clipboardPath = ""
					m.clipboardOp = ClipNone
				} else {
					m.clipboardPath = e.Path
					m.clipboardOp = ClipCut
				}
			}
			return m, nil
		}

		// Paste
		if matchKey(key, m.keys.paste) && m.clipboardPath != "" {
			destDir := m.currentOperationDir()
			m.pendingDestDir = destDir
			m.pendingPath = m.clipboardPath
			if m.clipboardOp == ClipCut {
				m.confirmAction = ConfirmPasteMove
				m.confirmMsg = fmt.Sprintf("Move %q\n  → %q?", filepath.Base(m.clipboardPath), destDir)
			} else {
				m.confirmAction = ConfirmPasteCopy
				m.confirmMsg = fmt.Sprintf("Copy %q\n  → %q?", filepath.Base(m.clipboardPath), destDir)
			}
			m.mode = ModeConfirm
			return m, nil
		}

		// Copy path to clipboard
		if matchKey(key, m.keys.copyPath) {
			e := m.selectedEntry()
			if e != nil {
				_ = clipboard.WriteAll(e.Path)
				m.statusMsg = "path copied to clipboard"
				return m, tea.Tick(1500*time.Millisecond, func(_ time.Time) tea.Msg {
					return clearStatusMsg{}
				})
			}
			return m, nil
		}

		// Edit in $EDITOR (or open in Neovim when running as a plugin)
		if matchKey(key, m.keys.edit) {
			e := m.selectedEntry()
			if e != nil {
				if m.openFile != "" {
					return m, m.exitWithFile(e.Path)
				}
				return m, m.openEditor(e.Path)
			}
			return m, nil
		}

		// Options: open config in editor (or open in Neovim when running as a plugin)
		if matchKey(key, m.keys.options) {
			if m.openFile != "" {
				return m, m.exitWithFile(config.ConfigPath())
			}
			return m, m.openEditor(config.ConfigPath())
		}
	}

	return m, nil
}

// ─── Navigation helpers ───────────────────────────────────────────────────────

func (m *Model) navigateLeft() {
	node := m.selectedNode()

	if node == nil {
		m.goToParentDir()
		return
	}

	// On an expanded dir: collapse it in place
	if node.Entry.IsDir() && node.Expanded {
		m.collapseNode(m.cursor)
		return
	}

	// At root depth on a collapsed dir/file: go up to parent dir
	if node.Depth == 0 {
		m.goToParentDir()
		return
	}

	// Inside tree on a collapsed node: collapse parent and jump to it
	parentIdx := m.parentNodeIdx(m.cursor)
	if parentIdx >= 0 {
		m.collapseNode(parentIdx)
		m.cursor = parentIdx
		m.adjustOffset()
	}
}

func (m *Model) goToParentDir() {
	parent := fs.ParentDir(m.rootDir)
	if parent == m.rootDir {
		return
	}
	prevRoot := m.rootDir
	_ = m.initTree(parent)
	for i, n := range m.nodes {
		if n.Entry.Path == prevRoot {
			m.cursor = i
			m.adjustOffset()
			break
		}
	}
}

// resolveDigitBuffer navigates to the Nth sibling at focused depth (1-based).
func (m *Model) resolveDigitBuffer() {
	n, err := strconv.Atoi(m.digitBuffer)
	if err != nil || n < 1 {
		return
	}
	focusedDepth := 0
	if m.cursor < len(m.nodes) {
		focusedDepth = m.nodes[m.cursor].Depth
	}
	target := nthSiblingAtDepth(m.nodes, focusedDepth, m.offset, n-1)
	if target < 0 {
		return
	}
	m.cursor = target
	m.adjustOffset()
	node := m.selectedNode()
	if node != nil && node.Entry.IsDir() {
		wasExpanded := node.Expanded
		parentDepth := node.Depth
		_ = m.expandNode(m.cursor)
		if !wasExpanded && m.cursor+1 < len(m.nodes) && m.nodes[m.cursor+1].Depth > parentDepth {
			m.cursor++
			m.adjustOffset()
		}
	}
}

// nthSiblingAtDepth returns the index of the nth (0-based) node at targetDepth
// scanning forward from fromOffset.
func nthSiblingAtDepth(nodes []TreeNode, targetDepth, fromOffset, n int) int {
	count := 0
	for i := fromOffset; i < len(nodes); i++ {
		if nodes[i].Depth == targetDepth {
			if count == n {
				return i
			}
			count++
		}
	}
	return -1
}

// ─── Input / confirm ─────────────────────────────────────────────────────────

func (m Model) submitInput() (tea.Model, tea.Cmd) {
	val := strings.TrimSpace(m.textInput.Value())
	m.textInput.Blur()
	m.mode = ModeNormal

	switch m.inputAction {
	case InputAdd:
		if val == "" {
			return m, nil
		}
		opDir := m.currentOperationDir()
		if err := fs.CreateEntry(opDir, val); err != nil {
			m.errorMsg = fmt.Sprintf("Error creating %q: %v", val, err)
			m.mode = ModeError
			return m, nil
		}
		dirIdx := m.findNodeByPath(opDir)
		if dirIdx >= 0 && m.nodes[dirIdx].Expanded {
			_ = m.refreshExpandedNode(dirIdx)
		} else if dirIdx >= 0 {
			_ = m.expandNode(dirIdx)
		} else {
			_ = m.rebuildTree()
		}

	case InputRename:
		if val == "" || val == filepath.Base(m.pendingPath) {
			return m, nil
		}
		m.confirmAction = ConfirmRename
		m.pendingName = val
		m.confirmMsg = fmt.Sprintf("Rename %q → %q?", filepath.Base(m.pendingPath), val)
		m.mode = ModeConfirm
	}

	return m, nil
}

func (m Model) executeConfirmedAction() (tea.Model, tea.Cmd) {
	m.mode = ModeNormal
	m.confirmMsg = ""

	switch m.confirmAction {
	case ConfirmDelete:
		parentDir := filepath.Dir(m.pendingPath)
		if err := fs.DeleteEntry(m.pendingPath); err != nil {
			m.errorMsg = fmt.Sprintf("Error deleting: %v", err)
			m.mode = ModeError
			return m, nil
		}
		for i, n := range m.nodes {
			if n.Entry.Path == m.pendingPath {
				m.nodes = append(m.nodes[:i], m.nodes[i+1:]...)
				if m.cursor >= len(m.nodes) && m.cursor > 0 {
					m.cursor--
				}
				break
			}
		}
		parentIdx := m.findNodeByPath(parentDir)
		if parentIdx >= 0 && m.nodes[parentIdx].Expanded {
			_ = m.refreshExpandedNode(parentIdx)
		}

	case ConfirmRename:
		parentDir := filepath.Dir(m.pendingPath)
		if err := fs.RenameEntry(m.pendingPath, m.pendingName); err != nil {
			m.errorMsg = fmt.Sprintf("Error renaming: %v", err)
			m.mode = ModeError
			return m, nil
		}
		parentIdx := m.findNodeByPath(parentDir)
		if parentIdx >= 0 {
			_ = m.refreshExpandedNode(parentIdx)
		} else {
			_ = m.rebuildTree()
		}

	case ConfirmPasteCopy:
		if err := fs.CopyEntry(m.pendingPath, m.pendingDestDir); err != nil {
			m.errorMsg = fmt.Sprintf("Error copying: %v", err)
			m.mode = ModeError
			return m, nil
		}
		// Refresh destination
		destIdx := m.findNodeByPath(m.pendingDestDir)
		if destIdx >= 0 && m.nodes[destIdx].Expanded {
			_ = m.refreshExpandedNode(destIdx)
		} else {
			_ = m.rebuildTree()
		}
		m.statusMsg = fmt.Sprintf("Copied %q", filepath.Base(m.pendingPath))
		return m, tea.Tick(1500*time.Millisecond, func(_ time.Time) tea.Msg { return clearStatusMsg{} })

	case ConfirmPasteMove:
		if err := fs.CopyEntry(m.pendingPath, m.pendingDestDir); err != nil {
			m.errorMsg = fmt.Sprintf("Error moving: %v", err)
			m.mode = ModeError
			return m, nil
		}
		if err := fs.DeleteEntry(m.pendingPath); err != nil {
			m.errorMsg = fmt.Sprintf("Copied but could not delete source: %v", err)
			m.mode = ModeError
			return m, nil
		}
		// Clear clipboard since item was moved
		m.clipboardPath = ""
		m.clipboardOp = ClipNone
		_ = m.rebuildTree()
		m.statusMsg = fmt.Sprintf("Moved %q", filepath.Base(m.pendingPath))
		return m, tea.Tick(1500*time.Millisecond, func(_ time.Time) tea.Msg { return clearStatusMsg{} })
	}

	return m, nil
}

// ─── Editor / opener ─────────────────────────────────────────────────────────

func (m Model) openEditor(path string) tea.Cmd {
	editor := m.cfg.Apps.Editor
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		for _, e := range []string{"nano", "vi", "vim", "nvim", "code", "notepad.exe"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}
	if editor == "" {
		return func() tea.Msg {
			return errorMsg("No editor found. Set $EDITOR or apps.editor in config.\nConfig: " + config.ConfigPath())
		}
	}
	c := exec.Command(editor, path)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return errorMsg(fmt.Sprintf("Editor exited with error: %v", err))
		}
		return reloadMsg{}
	})
}

// openDefaultCmd opens path in the default application using tea.ExecProcess,
// which suspends the TUI for the duration of the subprocess. This handles both
// GUI apps (imperceptible pause) and terminal apps (e.g. less, man) correctly,
// since the alt-screen is released while the subprocess runs and restored after.
func openDefaultCmd(path string, opener string) tea.Cmd {
	if opener == "" {
		switch runtime.GOOS {
		case "darwin":
			opener = "open"
		default:
			opener = "xdg-open"
		}
	}
	c := exec.Command(opener, path)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return errorMsg(fmt.Sprintf("Could not open %q: %v", path, err))
		}
		return reloadMsg{}
	})
}

// ─── Search ───────────────────────────────────────────────────────────────────

func (m Model) openSearchInput() (tea.Model, tea.Cmd) {
	m.prevNodes = make([]TreeNode, len(m.nodes))
	copy(m.prevNodes, m.nodes)
	m.prevRootDir = m.rootDir
	m.searchRecursive = false
	m.searchTextMode = false
	m.searchLiveNodes = nil
	m.textInput.Reset()
	m.textInput.Placeholder = "query  (-r recursive  -t text-in-files  -rt both)"
	m.textInput.SetValue("")
	m.textInput.Focus()
	m.mode = ModeSearch
	return m, textinput.Blink
}

func parseSearchFlags(raw string) (query string, recursive bool, textMode bool) {
	parts := strings.Fields(raw)
	var keep []string
	for _, p := range parts {
		switch p {
		case "-r":
			recursive = true
		case "-t":
			textMode = true
		case "-rt", "-tr":
			recursive = true
			textMode = true
		default:
			keep = append(keep, p)
		}
	}
	query = strings.Join(keep, " ")
	return
}

func (m *Model) applyLiveFilter() {
	raw := m.textInput.Value()
	query, _, _ := parseSearchFlags(raw)
	query = strings.TrimSpace(query)
	if query == "" {
		m.searchLiveNodes = nil
		m.cursor = 0
		m.offset = 0
		return
	}
	q := strings.ToLower(query)
	max := m.cfg.Display.SearchMaxResults
	if max < 1 {
		max = 20
	}
	var filtered []TreeNode
	for _, n := range m.prevNodes {
		if strings.Contains(strings.ToLower(n.Entry.Name), q) {
			filtered = append(filtered, n)
			if len(filtered) >= max {
				break
			}
		}
	}
	m.searchLiveNodes = filtered
	m.cursor = 0
	m.offset = 0
}

func (m Model) executeSearch() (tea.Model, tea.Cmd) {
	raw := m.textInput.Value()
	m.textInput.Blur()
	query, recursive, textMode := parseSearchFlags(raw)
	query = strings.TrimSpace(query)
	if query == "" {
		m.nodes = m.prevNodes
		m.rootDir = m.prevRootDir
		m.searchLiveNodes = nil
		m.mode = ModeNormal
		return m, nil
	}
	m.searchQuery = query
	m.searchRecursive = recursive
	m.searchTextMode = textMode
	m.searchRunning = true
	m.searchLiveNodes = nil
	req := search.Request{
		Dir:       m.rootDir,
		Query:     query,
		Recursive: recursive,
		TextMode:  textMode,
		Hidden:    m.showHidden,
	}
	tools := m.searchTools
	return m, func() tea.Msg {
		var results []search.Result
		_ = search.Run(tools, req, func(r search.Result) {
			results = append(results, r)
		})
		return searchResultMsg{results: results}
	}
}

// ─── Tea message types ────────────────────────────────────────────────────────

type errorMsg string
type reloadMsg struct{ reloadPath string }
type searchResultMsg struct{ results []search.Result }
type digitTimeoutMsg struct{}
type clearStatusMsg struct{}
