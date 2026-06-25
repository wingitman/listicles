package app

import (
	"compress/gzip"
	"fmt"
	"image"
	"image/color"
	stdDraw "image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	_ "github.com/askeladdk/aseprite"
	_ "github.com/gen2brain/heic"
	"github.com/oov/psd"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	_ "golang.org/x/image/bmp"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
	"github.com/wingitman/listicles/internal/fs"
	"github.com/wingitman/listicles/internal/state"
	"github.com/wingitman/listicles/internal/ui"
)

// ─── Top-level view ───────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Full-screen modes bypass the split layout entirely.
	switch m.mode {
	case ModeUpdates:
		b.WriteString(m.renderUpdatesScreen())
		b.WriteString("\n")
		b.WriteString(m.renderStatusBar())
		return b.String()
	case ModePlugins:
		b.WriteString(m.renderPluginsScreen())
		b.WriteString("\n")
		b.WriteString(m.renderStatusBar())
		return b.String()
	}

	// Build the list column content (crumbs/mode header + node list + overlay).
	var listBuf strings.Builder
	switch m.mode {
	case ModeRecents:
		listBuf.WriteString(m.renderRecentsHeader())
		listBuf.WriteString("\n")
	case ModeBookmarks:
		listBuf.WriteString(m.renderBookmarksHeader())
		listBuf.WriteString("\n")
	case ModeSearch:
		if m.cfg.Display.ParentDepth > 0 {
			crumbs := m.renderParentCrumbs()
			if crumbs != "" {
				listBuf.WriteString(crumbs)
			}
		}
		listBuf.WriteString(m.renderSearchBar())
		listBuf.WriteString("\n")
	default:
		crumbs := m.renderParentCrumbs()
		if crumbs != "" {
			listBuf.WriteString(crumbs)
		}
	}

	listBuf.WriteString(m.renderNodes())
	listBuf.WriteString("\n")
	listBuf.WriteString(m.renderOverlay())

	// ── Split layout when the preview panel is open ───────────────────────────
	previewWidth := m.previewPanelWidth()
	if previewWidth > 0 {
		// Height of the "inner" area between the header line and the footer.
		innerHeight := m.height - 2 // header row + the blank line we just wrote
		innerHeight--               // status bar
		if m.clipboardPath != "" {
			innerHeight--
		}
		if innerHeight < 1 {
			innerHeight = 1
		}

		listWidth := m.listColumnWidth()
		listContent := strings.TrimRight(listBuf.String(), "\n")
		listStr := lipgloss.NewStyle().
			Width(listWidth).
			Height(innerHeight).
			Render(listContent)

		previewStr := m.renderPreviewPanel(previewWidth, innerHeight)

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, listStr, previewStr))
	} else {
		b.WriteString(listBuf.String())
	}

	// Clipboard indicator line (above status bar).
	if m.clipboardPath != "" {
		b.WriteString(m.renderClipboardBar())
		b.WriteString("\n")
	}

	b.WriteString(m.renderStatusBar())

	return b.String()
}

// ─── Header ───────────────────────────────────────────────────────────────────

func (m Model) renderHeader() string {
	pathStr := m.rootDir
	if fs.IsDriveListRoot(pathStr) {
		pathStr = "This PC"
	}
	maxPathLen := m.width - 24
	if maxPathLen < 10 {
		maxPathLen = 10
	}
	if len(pathStr) > maxPathLen {
		pathStr = "…" + pathStr[len(pathStr)-maxPathLen:]
	}

	badges := []string{}
	if m.listMode == ListDirsAndFiles {
		badges = append(badges, ui.StyleMuted.Render("[files]"))
	}
	if m.showHidden {
		badges = append(badges, ui.StyleMuted.Render("[hidden]"))
	}
	if m.mode == ModeSearch {
		badges = append(badges, ui.StyleInputPrompt.Render("[search]"))
		if m.searchRunning {
			badges = append(badges, ui.StyleMuted.Render("[…]"))
		}
	}
	if m.mode == ModeRecents {
		badges = append(badges, ui.StyleInputPrompt.Render("[recents]"))
	}
	if m.mode == ModeBookmarks {
		badges = append(badges, ui.StyleInputPrompt.Render("[bookmarks]"))
	}
	if m.mode == ModeUpdates || m.mode == ModeUpdatePrompt {
		badges = append(badges, ui.StyleInputPrompt.Render("[updates]"))
	}
	if m.mode == ModePlugins {
		badges = append(badges, ui.StyleInputPrompt.Render("[plugins]"))
	}
	if m.digitBuffer != "" {
		badges = append(badges, ui.StyleNumber.Render("→ "+m.digitBuffer))
	}

	badgeStr := ""
	if len(badges) > 0 {
		badgeStr = "  " + strings.Join(badges, " ")
	}

	delby := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Render("delby")
	soft := lipgloss.NewStyle().Foreground(lipgloss.Color("#5865F2")).Bold(true).Render("soft")
	brand := " " + delby + soft + " "
	left := ui.StylePath.Render(pathStr) + badgeStr
	leftWidth := lipgloss.Width(left)
	brandWidth := lipgloss.Width(brand)
	pad := m.width - leftWidth - brandWidth
	if pad < 1 {
		pad = 1
	}
	headerLine := left + strings.Repeat(" ", pad) + brand

	rule := ui.StyleMuted.Render(strings.Repeat("─", clamp(m.width, 1, 80)))
	return headerLine + "\n" + rule
}

// ─── Parent crumbs ────────────────────────────────────────────────────────────

func (m Model) renderParentCrumbs() string {
	maxDepth := m.cfg.Display.ParentDepth
	if maxDepth <= 0 {
		return ""
	}

	var chain []string
	node := m.selectedNode()

	if node == nil || node.Depth == 0 {
		if fs.IsDriveListRoot(m.rootDir) {
			return ui.StyleRootDir.Render("  This PC") + "\n"
		}
		cur := m.rootDir
		for i := 0; i < maxDepth; i++ {
			parent := fs.ParentDir(cur)
			if parent == cur {
				break
			}
			chain = append(chain, parent)
			cur = parent
		}
	} else {
		idx := m.cursor
		for len(chain) < maxDepth {
			parentIdx := m.parentNodeIdx(idx)
			if parentIdx < 0 {
				chain = append(chain, m.rootDir)
				break
			}
			chain = append(chain, m.nodes[parentIdx].Entry.Path)
			idx = parentIdx
		}
	}

	if len(chain) == 0 {
		return ""
	}

	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	if len(chain) > maxDepth {
		chain = chain[len(chain)-maxDepth:]
	}

	var b strings.Builder
	for i, a := range chain {
		indent := strings.Repeat("  ", i)
		name := filepath.Base(a)
		if a == "/" {
			name = "/"
		}
		b.WriteString(ui.StyleParentCrumb.Render(indent+"  "+name+"/") + "\n")
	}

	var rootLabelPath string
	if node == nil || node.Depth == 0 {
		rootLabelPath = m.rootDir
	} else {
		parentIdx := m.parentNodeIdx(m.cursor)
		if parentIdx >= 0 {
			rootLabelPath = m.nodes[parentIdx].Entry.Path
		} else {
			rootLabelPath = m.rootDir
		}
	}

	lastCrumb := ""
	if len(chain) > 0 {
		lastCrumb = chain[len(chain)-1]
	}
	if rootLabelPath != lastCrumb {
		rootIndent := strings.Repeat("  ", len(chain))
		rootName := filepath.Base(rootLabelPath)
		if rootLabelPath == "/" {
			rootName = "/"
		}
		b.WriteString(ui.StyleRootDir.Render(rootIndent+"  "+rootName+"/") + "\n")
	}

	return b.String()
}

// ─── Search rendering ─────────────────────────────────────────────────────────

func (m Model) renderSearchBar() string {
	raw := m.textInput.Value()
	_, recursive, textMode, zoxideMode := parseSearchFlags(raw)

	flags := ""
	if recursive {
		flags += ui.StyleSuccess.Render(" -r")
	}
	if textMode {
		flags += ui.StyleSuccess.Render(" -t")
	}
	if zoxideMode {
		flags += ui.StyleSuccess.Render(" -z")
	}

	toolBadge := ""
	if zoxideMode && m.searchTools.HasZoxide {
		toolBadge = ui.StyleMuted.Render(" [zoxide]")
	} else if textMode && m.searchTools.HasRg {
		toolBadge = ui.StyleMuted.Render(" [rg]")
	} else if !zoxideMode && !textMode && m.searchTools.HasFd {
		toolBadge = ui.StyleMuted.Render(" [fd]")
	}

	// Count only top-level result nodes (not expanded snippet children).
	topLevelCount := 0
	for _, n := range m.searchLiveNodes {
		if n.Depth == 0 {
			topLevelCount++
		}
	}

	countStr := ""
	if m.searchRunning {
		countStr = ui.StyleMuted.Render("  searching…")
	} else if topLevelCount > 0 {
		max := m.cfg.Display.SearchMaxResults
		if topLevelCount >= max {
			countStr = ui.StyleMuted.Render(fmt.Sprintf("  %d+ matches", max))
		} else {
			countStr = ui.StyleMuted.Render(fmt.Sprintf("  %d match(es)", topLevelCount))
		}
	} else if !m.searchRunning && strings.TrimSpace(raw) != "" && m.searchQuery != "" {
		// Only show "no matches" after a full search has run, not while typing.
		countStr = ui.StyleMuted.Render("  no matches")
	}

	label := ui.StyleInputPrompt.Render("/") + flags + toolBadge + "  "
	bar := label + m.textInput.View() + countStr

	var hint string
	if m.searchInputActive {
		// Typing state: show how to run search.
		hint = ui.StyleMuted.Render("  [Enter]Search  [-r]Recursive  [-t]Content  [-z]Zoxide dirs  [Esc]Cancel")
	} else if len(m.searchLiveNodes) > 0 {
		// Navigation state: show how to navigate and confirm.
		hint = ui.StyleMuted.Render(fmt.Sprintf(
			"  [Enter]Open  [%s]Cd  [%s/%s]Navigate  [Esc]Edit Query  [%s]Exit",
			m.keys.cdDir, m.keys.up, m.keys.down, m.keys.quit,
		))
	} else {
		// Navigation state but no results: prompt to edit query.
		hint = ui.StyleMuted.Render(fmt.Sprintf(
			"  [Esc]Edit Query  [%s]Exit",
			m.keys.quit,
		))
	}
	return bar + "\n" + hint
}

// renderSearchResultHeader is kept for backward compatibility with tests.
// In normal usage the search result info is shown inline in the search bar.
func (m Model) renderSearchResultHeader() string {
	if m.searchRunning {
		return ui.StyleMuted.Render("  Searching…")
	}
	nodes := m.searchLiveNodes
	count := len(nodes)
	if count == 0 {
		return ui.StyleError.Render("  No results for ") +
			ui.StyleInputPrompt.Render(fmt.Sprintf("%q", m.searchQuery))
	}
	modeStr := "names"
	if m.searchTextMode {
		modeStr = "content"
	} else if m.searchZoxideMode {
		modeStr = "zoxide dirs"
	}
	scopeStr := "current dir"
	if m.searchRecursive {
		scopeStr = "recursive"
	}
	toolStr := ""
	if m.searchZoxideMode && m.searchTools.HasZoxide {
		toolStr = " via zoxide"
	} else if m.searchTextMode && m.searchTools.HasRg {
		toolStr = " via rg"
	} else if !m.searchZoxideMode && !m.searchTextMode && m.searchTools.HasFd {
		toolStr = " via fd"
	}
	return ui.StyleSuccess.Render(fmt.Sprintf("  %d result(s)", count)) +
		ui.StyleMuted.Render(fmt.Sprintf(" — %s search%s in %s for ", modeStr, toolStr, scopeStr)) +
		ui.StyleInputPrompt.Render(fmt.Sprintf("%q", m.searchQuery)) +
		ui.StyleMuted.Render("  ·  Esc to clear")
}

// ─── Recents / Bookmarks headers ─────────────────────────────────────────────

func (m Model) renderRecentsHeader() string {
	scopeLabel := filepath.Base(m.gitRootOrCwd())
	if m.stateScope {
		scopeLabel = "all projects"
	}

	count := 0
	for _, n := range m.nodes {
		if !n.IsGroupHeader {
			count++
		}
	}

	title := ui.StyleSuccess.Render(fmt.Sprintf("  Recents — %s", scopeLabel))
	if count == 0 {
		title = ui.StyleMuted.Render(fmt.Sprintf("  Recents — %s  (none)", scopeLabel))
	}
	hint := ui.StyleMuted.Render(fmt.Sprintf(
		"  [%s]Global  [%s]Bookmarks  [%s]Remove  [Esc]Back",
		m.keys.switchTabsGlobal, m.keys.switchTabs, m.keys.delete,
	))
	return title + "\n" + hint
}

func (m Model) renderBookmarksHeader() string {
	scopeLabel := filepath.Base(m.gitRootOrCwd())
	if m.stateScope {
		scopeLabel = "all projects"
	}

	count := 0
	for _, n := range m.nodes {
		if !n.IsGroupHeader {
			count++
		}
	}

	title := ui.StyleSuccess.Render(fmt.Sprintf("  Bookmarks — %s", scopeLabel))
	if count == 0 {
		title = ui.StyleMuted.Render(fmt.Sprintf("  Bookmarks — %s  (none)", scopeLabel))
	}
	hint := ui.StyleMuted.Render(fmt.Sprintf(
		"  [%s]Global  [%s]Close  [%s]Add  [%s]Remove  [%s]Rename  [Esc]Back",
		m.keys.switchTabsGlobal, m.keys.switchTabs, m.keys.bookmark, m.keys.delete, m.keys.rename,
	))
	return title + "\n" + hint
}

// ─── Node list ────────────────────────────────────────────────────────────────

func (m Model) renderNodes() string {
	nodes := m.nodes
	if m.mode == ModeSearch && m.searchLiveNodes != nil {
		nodes = m.searchLiveNodes
	}

	if len(nodes) == 0 {
		if m.mode == ModeSearch || m.mode == ModeRecents || m.mode == ModeBookmarks {
			return ""
		}
		return ui.StyleMuted.Render("  (empty directory)") + "\n"
	}

	focusedDepth := 0
	if m.cursor < len(nodes) {
		focusedDepth = nodes[m.cursor].Depth
	}

	var b strings.Builder
	rows := m.visibleRows()
	end := m.offset + rows
	if end > len(nodes) {
		end = len(nodes)
	}

	siblingCount := 0
	for i := m.offset; i < end; i++ {
		b.WriteString(m.renderNode(i, nodes[i], focusedDepth, siblingCount))
		b.WriteString("\n")
		if nodes[i].Depth == focusedDepth && !nodes[i].IsGroupHeader {
			siblingCount++
		}
	}

	if m.offset > 0 {
		b.WriteString(ui.StyleMuted.Render(fmt.Sprintf("  ↑ %d more above", m.offset)) + "\n")
	}
	below := len(nodes) - end
	if below > 0 {
		b.WriteString(ui.StyleMuted.Render(fmt.Sprintf("  ↓ %d more below", below)) + "\n")
	}

	return b.String()
}

func (m Model) renderNode(idx int, node TreeNode, focusedDepth int, siblingIdx int) string {
	e := node.Entry

	// ── Group headers (non-selectable separators in global bookmark/recents view)
	if node.IsGroupHeader {
		colW := m.listColumnWidth()
		name := e.Name
		if len(name) > colW-4 {
			name = "…" + name[len(name)-(colW-5):]
		}
		line := ui.StyleParentCrumb.Render("  " + name)
		if colW > lipgloss.Width(line)+2 {
			line += ui.StyleMuted.Render(strings.Repeat("─", colW-lipgloss.Width(line)-2))
		}
		return line
	}

	// ── Text-search match child (line snippet)
	if node.IsTextMatch {
		colW := m.listColumnWidth()
		lineNum := ui.StyleNumber.Render(fmt.Sprintf(" :%d  ", node.MatchLineNum))
		snippet := ui.StyleMuted.Render(node.MatchSnippet)
		line := lineNum + snippet
		if idx == m.cursor {
			lineWidth := lipgloss.Width(line)
			if lineWidth < colW {
				line = line + strings.Repeat(" ", colW-lineWidth)
			}
			return ui.StyleSelected.Render(line)
		}
		return line
	}

	// ── Recents / Bookmarks mode: show path + time-ago suffix
	if m.mode == ModeRecents || m.mode == ModeBookmarks {
		return m.renderTabNode(idx, node)
	}

	// ── Standard tree node ──────────────────────────────────────────────────

	// Number label: only for nodes at the focused depth
	numLabel := " · "
	digits := len(strconv.Itoa(siblingIdx + 1))
	strIndent := strings.Repeat(" ", 3-digits)
	if node.Depth == focusedDepth && siblingIdx < 99 {
		numLabel = fmt.Sprintf(" %d%v", siblingIdx+1, strIndent)
	}

	// Indent
	crumbDepth := 0
	if m.cfg.Display.ParentDepth > 0 && m.mode != ModeSearch {
		crumbDepth = m.cfg.Display.ParentDepth + 1
	}
	totalIndent := crumbDepth + node.Depth
	indent := strings.Repeat("  ", totalIndent)

	// Icon
	icon := "  "
	if e.IsDir() || len(node.PendingChildren) > 0 {
		if node.Expanded {
			icon = " ▼ "
		} else {
			icon = " ▶ "
		}
	}

	// Display name
	displayName := e.Name
	if m.mode == ModeSearch && !node.IsTextMatch {
		if rel, err := filepath.Rel(m.prevRootDir, e.Path); err == nil {
			base := filepath.Base(e.Path)
			if e.Name == base {
				// Plain name match: show relative path.
				displayName = rel
			} else if strings.HasPrefix(e.Name, base) {
				// Text-search parent: "basename (N matches)" — preserve the suffix.
				suffix := strings.TrimPrefix(e.Name, base)
				displayName = rel + suffix
			}
			// else: leave displayName = e.Name as-is.
		}
	}

	// Clipboard highlight
	isClipboard := m.clipboardPath == e.Path
	clipSuffix := ""
	if isClipboard {
		if m.clipboardOp == ClipCopy {
			clipSuffix = ui.StyleClipboard.Render(" [copy]")
		} else if m.clipboardOp == ClipCut {
			clipSuffix = ui.StyleClipboard.Render(" [cut]")
		}
	}

	// Style name — gitignored entries use muted style (same as hidden files).
	var nameStr string
	if e.Ignored {
		if e.IsDir() {
			nameStr = ui.StyleMuted.Render(displayName+"/") + clipSuffix
		} else {
			nameStr = ui.StyleMuted.Render(displayName) + clipSuffix
		}
	} else if e.IsDir() {
		nameStr = ui.StyleDirName.Render(displayName+"/") + clipSuffix
	} else {
		nameStr = ui.StyleFileName.Render(displayName) + clipSuffix
	}

	// Detail suffix
	detail := m.renderDetail(e)

	// Compose line
	numStyled := ui.StyleNumber.Render(numLabel)
	line := indent + numStyled + icon + nameStr

	if detail != "" {
		bare := indent + numLabel + icon + displayName
		if e.IsDir() {
			bare += "/"
		}
		padLen := 48 - len(bare)
		padding := "  "
		if padLen > 0 {
			padding = strings.Repeat(" ", padLen)
		}
		line = line + padding + ui.StyleDetail.Render(detail)
	}

	colW := m.listColumnWidth()
	maxW := colW - 1
	if maxW < 10 {
		maxW = 10
	}
	if lipgloss.Width(line) > maxW {
		line = indent + numStyled + icon + nameStr
	}

	// Highlight selected row
	if idx == m.cursor {
		lineWidth := lipgloss.Width(line)
		if lineWidth < colW {
			line = line + strings.Repeat(" ", colW-lineWidth)
		}
		return ui.StyleSelected.Render(line)
	}

	return line
}

// renderTabNode renders a single row in Recents or Bookmarks mode.
// Format: icon  name  ·  rel/path/  ·  time-ago
func (m Model) renderTabNode(idx int, node TreeNode) string {
	e := node.Entry

	icon := "  "
	if e.IsDir() {
		icon = " ▶ "
	}

	name := e.Name
	nameStr := ui.StyleFileName.Render(name)
	if e.IsDir() {
		nameStr = ui.StyleDirName.Render(name + "/")
	}

	// Relative path from project root.
	root := m.gitRootOrCwd()
	rel := ""
	if r, err := filepath.Rel(root, filepath.Dir(e.Path)); err == nil && r != "." {
		rel = ui.StyleMuted.Render("  " + r + "/")
	}

	// Time-ago from recents (only in ModeRecents).
	timeStr := ""
	if m.mode == ModeRecents && m.appState != nil {
		for _, r := range m.appState.Recents {
			if r.Path == e.Path {
				timeStr = ui.StyleMuted.Render("  " + state.FormatTimeAgo(r.AccessedAt))
				break
			}
		}
	}

	line := icon + nameStr + rel + timeStr

	if idx == m.cursor {
		colW := m.listColumnWidth()
		lineWidth := lipgloss.Width(line)
		if lineWidth < colW {
			line = line + strings.Repeat(" ", colW-lineWidth)
		}
		return ui.StyleSelected.Render(line)
	}
	return line
}

func (m Model) renderDetail(e fs.Entry) string {
	switch m.detailLevel {
	case DetailNone:
		return ""
	case DetailCount:
		if e.IsDir() {
			nf, nd, _ := fs.DirStats(e.Path)
			return fmt.Sprintf("%d files, %d dirs", nf, nd)
		}
		return fs.HumanSize(e.Size)
	case DetailSize:
		if e.IsDir() {
			_, _, sz := fs.DirStats(e.Path)
			return fs.HumanSize(sz)
		}
		return fs.HumanSize(e.Size)
	case DetailFullPath:
		return e.Path
	case DetailModTime:
		return fs.FileModTime(e.Path)
	case DetailBirthTime:
		return fs.FileBirthTime(e.Path)
	case DetailPermissions:
		return fs.FilePermissions(e.Path)
	case DetailOwner:
		return fs.FileOwner(e.Path)
	case DetailMimeType:
		return fs.FileMimeType(e.Path, e.IsDir())
	}
	return ""
}

// ─── Overlays ─────────────────────────────────────────────────────────────────

func (m Model) renderOverlay() string {
	switch m.mode {
	case ModeError:
		box := ui.StyleConfirmBox.Render(
			ui.StyleError.Render("Error") + "\n\n" +
				m.errorMsg + "\n\n" +
				ui.StyleMuted.Render("Press any key to continue"),
		)
		return box + "\n"

	case ModeConfirm:
		box := ui.StyleConfirmBox.Render(
			ui.StyleError.Render("Confirm") + "\n\n" +
				m.confirmMsg + "\n\n" +
				ui.StyleMuted.Render("Press ") +
				ui.StyleSuccess.Render("y") +
				ui.StyleMuted.Render(" to confirm, any other key to cancel"),
		)
		return box + "\n"

	case ModeInput:
		label := ""
		switch m.inputAction {
		case InputAdd:
			label = "Add (end name with / to create a directory):"
		case InputRename:
			label = fmt.Sprintf("Rename %q:", filepath.Base(m.pendingPath))
		case InputPasteCopy:
			label = fmt.Sprintf("Copy %q as:", filepath.Base(m.pendingPath))
		case InputPasteMove:
			label = fmt.Sprintf("Move %q as:", filepath.Base(m.pendingPath))
		}
		box := ui.StyleConfirmBox.Render(
			ui.StyleInputPrompt.Render(label) + "\n\n" +
				m.textInput.View() + "\n\n" +
				ui.StyleMuted.Render("Enter to confirm · Esc to cancel"),
		)
		return box + "\n"

	case ModeUpdatePrompt:
		return m.renderUpdatePrompt() + "\n"
	}
	return ""
}

func (m Model) renderUpdatePrompt() string {
	commits := m.updateInfo.Available
	rows := len(commits)
	if rows > 5 {
		rows = 5
	}
	start := m.updateCursor - rows/2
	if start < 0 {
		start = 0
	}
	if start+rows > len(commits) {
		start = len(commits) - rows
		if start < 0 {
			start = 0
		}
	}
	var b strings.Builder
	b.WriteString(ui.StyleSuccess.Render("Update available"))
	b.WriteString("\n\n")
	b.WriteString("Current: " + shortCommit(m.updateInfo.CurrentCommit) + "\n")
	b.WriteString("Latest:  " + shortCommit(m.updateInfo.LatestCommit) + "\n")
	if m.updateInfo.Branch != "" {
		b.WriteString("Branch:  " + m.updateInfo.Branch + " -> " + m.updateInfo.Upstream + "\n")
	}
	b.WriteString("\nRecent changes:\n")
	for i := start; i < start+rows && i < len(commits); i++ {
		c := commits[i]
		prefix := "  "
		if i == m.updateCursor {
			prefix = "> "
		}
		line := fmt.Sprintf("%s%s %s", prefix, c.Short, c.Subject)
		if i == m.updateCursor {
			line = ui.StyleSelected.Render(line)
		}
		b.WriteString(line + "\n")
		if m.updateExpanded[c.Hash] && c.Body != "" {
			b.WriteString(ui.StyleMuted.Render(indentLines(c.Body, "    ")) + "\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(ui.StyleMuted.Render("y install in new terminal and exit · enter show/hide details · n/esc skip"))
	return ui.StyleConfirmBox.Render(b.String())
}

func (m Model) renderUpdatesScreen() string {
	var b strings.Builder
	b.WriteString(ui.StyleInputPrompt.Render("Updates"))
	b.WriteString("\n")
	if m.updateChecking {
		b.WriteString(ui.StyleMuted.Render("Checking for updates..."))
		return b.String()
	}
	if m.updateInfo.CheckError != "" {
		b.WriteString(ui.StyleError.Render("Check failed: ") + m.updateInfo.CheckError)
		return b.String()
	}
	if m.updateInfo.RepoPath == "" {
		b.WriteString(ui.StyleMuted.Render("No update information loaded."))
		return b.String()
	}
	b.WriteString(ui.StyleMuted.Render("Repo: ") + m.updateInfo.RepoPath + "\n")
	b.WriteString(ui.StyleMuted.Render("Branch: ") + m.updateInfo.Branch + " -> " + m.updateInfo.Upstream + "\n")
	b.WriteString(ui.StyleMuted.Render("Current: ") + shortCommit(m.updateInfo.CurrentCommit) + "\n")
	b.WriteString(ui.StyleMuted.Render("Latest: ") + shortCommit(m.updateInfo.LatestCommit) + "\n\n")

	commits := m.updateCommits()
	if len(commits) == 0 {
		b.WriteString(ui.StyleSuccess.Render("No newer commits found."))
		return b.String()
	}
	if len(m.updateInfo.Available) > 0 {
		b.WriteString(ui.StyleSuccess.Render(fmt.Sprintf("%d update(s) available", len(m.updateInfo.Available))) + "\n")
	} else {
		b.WriteString(ui.StyleMuted.Render("Recent history") + "\n")
	}

	rows := m.visibleRows()
	if rows > len(commits) {
		rows = len(commits)
	}
	start := m.updateCursor - rows/2
	if start < 0 {
		start = 0
	}
	if start+rows > len(commits) {
		start = len(commits) - rows
		if start < 0 {
			start = 0
		}
	}
	for i := start; i < start+rows && i < len(commits); i++ {
		c := commits[i]
		line := fmt.Sprintf("  %s  %s  %s", c.Short, c.Date, c.Subject)
		if i == m.updateCursor {
			line = ui.StyleSelected.Render(line)
		}
		b.WriteString(line + "\n")
		if m.updateExpanded[c.Hash] && c.Body != "" {
			b.WriteString(ui.StyleMuted.Render(indentLines(c.Body, "    ")) + "\n")
		}
	}
	return b.String()
}

func (m Model) renderPluginsScreen() string {
	var b strings.Builder
	b.WriteString(ui.StyleInputPrompt.Render("Plugins"))
	b.WriteString("\n")
	b.WriteString(ui.StyleMuted.Render("Optional command integrations used by search."))
	b.WriteString("\n\n")

	infos := m.pluginInfos()
	for i, info := range infos {
		status := ui.StyleSuccess.Render("active")
		if !info.Enabled {
			status = ui.StyleMuted.Render("disabled")
		} else if !info.Installed {
			status = ui.StyleError.Render("missing")
		}
		line := fmt.Sprintf("  %-8s %-10s %s", info.Name, status, ui.StyleMuted.Render(info.Description))
		if i == m.pluginCursor {
			lineWidth := lipgloss.Width(line)
			if lineWidth < m.width {
				line += strings.Repeat(" ", m.width-lineWidth)
			}
			line = ui.StyleSelected.Render(line)
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

// ─── Clipboard bar ────────────────────────────────────────────────────────────

func (m Model) renderClipboardBar() string {
	op := "copy"
	if m.clipboardOp == ClipCut {
		op = "cut"
	}
	return ui.StyleClipboard.Render(fmt.Sprintf(
		"  [%s] %s  [%s]Paste  [%s/%s]Clear",
		op,
		filepath.Base(m.clipboardPath),
		m.keys.paste,
		m.keys.yank,
		m.keys.cut,
	))
}

// ─── Preview panel ────────────────────────────────────────────────────────────

// renderPreviewPanel renders the right-hand preview column.
//
// For image content the panel is built manually, line-by-line, without passing
// the halfblock ANSI sequences through lipgloss's Render pipeline. Doing so
// causes spurious line-wrapping because the Unicode width library used by
// lipgloss measures the ▀ glyph (U+2580) as 2 columns rather than 1 in some
// configurations, making every halfblock row appear twice as wide as it is and
// causing lipgloss to wrap it — producing the alternating stripe artifact.
//
// For text/details content lipgloss is used normally.
func (m Model) renderPreviewPanel(width, height int) string {
	// Border (1) + left pad (1) + right pad (1) = 3 overhead columns.
	const overhead = 3
	innerW := width - overhead
	if innerW < 4 {
		innerW = 4
	}

	e := m.selectedEntry()
	showImage := e != nil && !e.IsDir() &&
		fs.IsImageFile(e.Path) && m.previewMode == PreviewModeImage

	if showImage {
		if panel := m.buildImagePanel(e.Path, innerW, height); panel != "" {
			return panel
		}
		// Image rendering produced no visible content (e.g. unsupported SVG
		// features). Show the details view with a note so the panel isn't blank.
		note := ui.StyleMuted.Render("  preview unavailable")
		content := ui.StylePreviewTitle.Render("details") + "\n" + note + "\n" + renderFileInfo(*e, innerW)
		return ui.StylePreview.Width(innerW).Height(height).Render(content)
	}

	// Text / details path — lipgloss handles layout.
	content := m.renderPreviewContent(innerW, height)
	return ui.StylePreview.Width(innerW).Height(height).Render(content)
}

// buildImagePanel constructs the preview panel for image content without
// involving lipgloss's text-layout engine for the ANSI halfblock rows.
// Each output line is exactly (innerW + 3) terminal columns wide:
//
//	│[space][innerW cols of content][space]\n
//
// Returns "" when the render produced no visible image (e.g. unsupported SVG
// features), so the caller can fall back to showing the details view.
func (m Model) buildImagePanel(path string, innerW, height int) string {
	imgRows := height - 1 // last row reserved for the mode-toggle hint
	if imgRows < 1 {
		imgRows = 1
	}

	imageStr := m.renderImagePreview(path, innerW, imgRows)

	// No ▀ glyphs = the decode/render failed or produced nothing visible.
	// Signal the caller to use the details fallback instead of a blank panel.
	if strings.Count(imageStr, "▀") == 0 {
		return ""
	}

	lines := strings.Split(strings.TrimRight(imageStr, "\n"), "\n")

	hint := ui.StyleMuted.Render("[" + m.keys.previewMode + "] details")
	border := ui.StyleMuted.Render("│")

	// darkFill returns n spaces with an explicit dark background so that
	// transparent terminals don't bleed the desktop through unused cells.
	darkFill := func(n int) string {
		if n <= 0 {
			return ""
		}
		return fmt.Sprintf("\x1b[48;2;26;26;46m%s\x1b[0m", strings.Repeat(" ", n))
	}

	var sb strings.Builder
	for row := 0; row < height; row++ {
		sb.WriteString(border)
		sb.WriteByte(' ') // left pad

		if row == height-1 {
			// Hint row.
			sb.WriteString(hint)
			if fill := innerW - lipgloss.Width(hint); fill > 0 {
				sb.WriteString(darkFill(fill))
			}
		} else if row < len(lines) {
			// Image row — write raw ANSI halfblocks; do NOT pass through lipgloss.
			sb.WriteString(lines[row])
			// Count ▀ runes to determine exact rendered column width.
			if fill := innerW - strings.Count(lines[row], "▀"); fill > 0 {
				sb.WriteString(darkFill(fill))
			}
		} else {
			// Blank row below the image.
			sb.WriteString(darkFill(innerW))
		}

		sb.WriteByte(' ') // right pad
		sb.WriteByte('\n')
	}
	return sb.String()
}

// renderPreviewContent returns the content string for non-image panel modes
// (details view, or image file in details mode). This is fed into lipgloss.
func (m Model) renderPreviewContent(width, height int) string {
	e := m.selectedEntry()
	if e == nil {
		return ui.StyleMuted.Render("nothing selected")
	}

	isImage := !e.IsDir() && fs.IsImageFile(e.Path)

	// Details view (or non-image fallback).
	title := ui.StylePreviewTitle.Render("details")
	if isImage {
		title += "  " + ui.StyleMuted.Render("[" + m.keys.previewMode + "] image")
	}
	return title + "\n" + renderFileInfo(*e, width)
}

// renderImagePreview returns a mosaic-rendered image string, using a per-size
// cache to avoid re-decoding on every frame. Cache writes are safe from a
// value-receiver because maps are reference types in Go.
func (m Model) renderImagePreview(path string, width, height int) string {
	if m.previewCache == nil {
		return ui.StyleMuted.Render("(preview unavailable)")
	}
	key := fmt.Sprintf("%s\x00%d\x00%d", path, width, height)
	if cached, ok := m.previewCache[key]; ok {
		return cached
	}
	result := decodeAndRenderImage(path, width, height)
	m.previewCache[key] = result
	return result
}

// ── Image decoder ─────────────────────────────────────────────────────────────

// decodeAndRenderImage decodes an image file and renders it as truecolor
// halfblock terminal art sized to fit panelCols × panelRows terminal cells.
//
// Supported formats:
//   - PNG, JPEG, GIF, WebP, BMP, TIFF  — stdlib / golang.org/x/image decoders
//   - HEIC / HEIF                       — gen2brain/heic (CGo-free via wasm2go)
//   - Aseprite (.ase / .aseprite)       — askeladdk/aseprite
//   - Photoshop (.psd / .psb)           — oov/psd (merged image)
//   - SVG / SVGZ                        — srwiley/oksvg + rasterx
func decodeAndRenderImage(path string, panelCols, panelRows int) string {
	if panelCols < 1 || panelRows < 1 {
		return ""
	}

	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".svg", ".svgz":
		img, err := rasterizeSVG(path, panelCols, panelRows)
		if err != nil {
			return ui.StyleMuted.Render("  (SVG: " + truncErr(err) + ")")
		}
		return renderHalfblocks(img, panelCols, panelRows)

	case ".psd", ".psb":
		img, err := decodePSD(path)
		if err != nil {
			return ui.StyleMuted.Render("  (PSD: " + truncErr(err) + ")")
		}
		return scaleAndRender(img, panelCols, panelRows)

	default:
		f, err := os.Open(path)
		if err != nil {
			return ui.StyleMuted.Render("  (cannot open image)")
		}
		defer f.Close()
		img, _, err := image.Decode(f)
		if err != nil {
			return ui.StyleMuted.Render("  (cannot decode: " + truncErr(err) + ")")
		}
		return scaleAndRender(img, panelCols, panelRows)
	}
}

// scaleAndRender scales img to fit within panelCols × panelRows and renders it.
func scaleAndRender(img image.Image, panelCols, panelRows int) string {
	b := img.Bounds()
	outCols, outRows := fitImageToPanel(b.Dx(), b.Dy(), panelCols, panelRows)
	pixW, pixH := outCols, outRows*2
	scaled := image.NewRGBA(image.Rect(0, 0, pixW, pixH))
	xdraw.ApproxBiLinear.Scale(scaled, scaled.Bounds(), img, img.Bounds(), stdDraw.Over, nil)
	return renderHalfblocks(scaled, outCols, outRows)
}

// decodePSD decodes a Photoshop PSD/PSB file and returns its merged image.
func decodePSD(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	doc, _, err := psd.Decode(f, &psd.DecodeOptions{SkipMergedImage: false})
	if err != nil {
		return nil, err
	}
	if doc.Picker == nil {
		return nil, fmt.Errorf("no merged image in PSD")
	}
	return doc.Picker, nil
}

// rasterizeSVG rasterizes an SVG or SVGZ file into an RGBA image scaled to
// fit within panelCols × panelRows terminal cells, preserving aspect ratio.
func rasterizeSVG(path string, panelCols, panelRows int) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// SVGZ is gzip-compressed SVG; decompress on the fly.
	var r interface {
		Read([]byte) (int, error)
	} = f
	if strings.ToLower(filepath.Ext(path)) == ".svgz" {
		gz, gerr := gzip.NewReader(f)
		if gerr != nil {
			return nil, gerr
		}
		defer gz.Close()
		r = gz
	}

	icon, err := oksvg.ReadIconStream(r)
	if err != nil {
		return nil, err
	}

	svgW, svgH := icon.ViewBox.W, icon.ViewBox.H
	if svgW <= 0 {
		svgW = float64(panelCols)
	}
	if svgH <= 0 {
		svgH = float64(panelRows * 2)
	}

	outCols, outRows := fitImageToPanel(int(svgW), int(svgH), panelCols, panelRows)
	pixW, pixH := outCols, outRows*2

	icon.SetTarget(0, 0, float64(pixW), float64(pixH))
	rgba := image.NewRGBA(image.Rect(0, 0, pixW, pixH))

	// Pre-fill with the app's dark background so transparent SVGs look right.
	bg := &image.Uniform{color.RGBA{0x1A, 0x1A, 0x2E, 0xFF}}
	stdDraw.Draw(rgba, rgba.Bounds(), bg, image.Point{}, stdDraw.Src)

	scanner := rasterx.NewScannerGV(pixW, pixH, rgba, rgba.Bounds())
	icon.Draw(rasterx.NewDasher(pixW, pixH, scanner), 1.0)

	// oksvg silently skips unsupported features (e.g. linearGradient fill
	// references, CSS class styles). If every sampled pixel still equals the
	// pre-fill background, nothing was rendered — treat that as a failure so
	// the caller can show the details view instead of a blank panel.
	if svgRenderedNothing(rgba, 0x1A, 0x1A, 0x2E) {
		return nil, fmt.Errorf("no visible content (SVG may use unsupported features such as gradients or CSS styles)")
	}

	return rgba, nil
}

// svgRenderedNothing samples the image on a coarse grid and returns true when
// every sampled pixel matches the background colour — indicating that oksvg
// produced no visible output.
func svgRenderedNothing(img *image.RGBA, bgR, bgG, bgB uint8) bool {
	b := img.Bounds()
	const step = 3
	for y := b.Min.Y; y < b.Max.Y; y += step {
		for x := b.Min.X; x < b.Max.X; x += step {
			r, g, bv, _ := img.At(x, y).RGBA()
			if uint8(r>>8) != bgR || uint8(g>>8) != bgG || uint8(bv>>8) != bgB {
				return false
			}
		}
	}
	return true
}

// fitImageToPanel calculates output dimensions in terminal cells that fit an
// image of imgW × imgH pixels within panelCols × panelRows cells, preserving
// aspect ratio. Each terminal row covers ~2× the visual height of a column.
func fitImageToPanel(imgW, imgH, panelCols, panelRows int) (cols, rows int) {
	if imgW < 1 {
		imgW = 1
	}
	if imgH < 1 {
		imgH = 1
	}
	if panelCols < 1 {
		panelCols = 1
	}
	if panelRows < 1 {
		panelRows = 1
	}
	panelVis := float64(panelCols) / (float64(panelRows) * 2.0)
	imgAspect := float64(imgW) / float64(imgH)

	if imgAspect >= panelVis {
		cols = panelCols
		rows = int(math.Round(float64(panelCols) / imgAspect / 2.0))
	} else {
		rows = panelRows
		cols = int(math.Round(float64(panelRows) * 2.0 * imgAspect))
	}
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	if cols > panelCols {
		cols = panelCols
	}
	if rows > panelRows {
		rows = panelRows
	}
	return
}

// renderHalfblocks renders a pre-scaled image as truecolor terminal art using
// upper-halfblock glyphs (▀). Foreground = top pixel, background = bottom pixel.
// The image should be exactly cols × (rows*2) pixels.
//
// Each glyph uses two separate SGR sequences (one for fg, one for bg) followed
// by an explicit reset. This per-glyph reset is the format charmbracelet/x/ansi
// produces and is what lipgloss's width counter expects; a single combined
// sequence across multiple glyphs can confuse the counter and cause spurious
// line-wrapping that produces the alternating-stripe artifact.
func renderHalfblocks(img image.Image, cols, rows int) string {
	var sb strings.Builder
	b := img.Bounds()

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			x := b.Min.X + col
			topY := b.Min.Y + row*2
			botY := b.Min.Y + row*2 + 1

			topR, topG, topB, topA := img.At(x, topY).RGBA()
			botR, botG, botB, botA := img.At(x, botY).RGBA()

			fgR, fgG, fgB := compositeOnDark(topR, topG, topB, topA)
			bgR, bgG, bgB := compositeOnDark(botR, botG, botB, botA)

			// Two separate sequences + per-glyph reset.
			fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀\x1b[0m",
				fgR, fgG, fgB, bgR, bgG, bgB)
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// compositeOnDark alpha-composites a premultiplied RGBA pixel (as returned by
// image.Color.RGBA) onto the app's dark background (#1A1A2E).
func compositeOnDark(r, g, b, a uint32) (uint8, uint8, uint8) {
	const bgR, bgG, bgB = 0x1A, 0x1A, 0x2E
	if a == 0xffff {
		return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)
	}
	if a == 0 {
		return bgR, bgG, bgB
	}
	alpha := float64(a) / 0xffff
	inv := 1.0 - alpha
	return uint8(float64(r>>8)*alpha + float64(bgR)*inv),
		uint8(float64(g>>8)*alpha + float64(bgG)*inv),
		uint8(float64(b>>8)*alpha + float64(bgB)*inv)
}

// truncErr returns a short error string suitable for inline display.
func truncErr(err error) string {
	s := err.Error()
	if len(s) > 40 {
		return s[:40] + "…"
	}
	return s
}

// renderFileInfo formats filesystem metadata for the given entry into a
// details panel of the given character width.
func renderFileInfo(e fs.Entry, width int) string {
	var b strings.Builder

	// Path — truncate if it exceeds the panel width.
	path := e.Path
	if len(path) > width {
		path = "…" + path[len(path)-width+1:]
	}
	b.WriteString(ui.StyleMuted.Render(path))
	b.WriteString("\n\n")

	row := func(label, value string) {
		if len(value) > width-11 && width > 12 {
			value = value[:width-11]
		}
		b.WriteString(ui.StylePreviewLabel.Render(fmt.Sprintf("  %-8s", label)))
		b.WriteString(" " + value + "\n")
	}

	if e.IsDir() {
		row("type", "directory")
		nf, nd, sz := fs.DirStats(e.Path)
		row("items", fmt.Sprintf("%d files, %d dirs", nf, nd))
		row("size", fs.HumanSize(sz))
	} else {
		row("type", fs.FileMimeType(e.Path, false))
		row("size", fs.HumanSize(e.Size))
	}

	row("modified", fs.FileModTime(e.Path))
	row("created", fs.FileBirthTime(e.Path))
	row("perms", fs.FilePermissions(e.Path))
	row("owner", fs.FileOwner(e.Path))

	return b.String()
}

// ─── Status bar ───────────────────────────────────────────────────────────────

func (m Model) renderStatusBar() string {
	k := m.keys

	truncate := func(s string) string {
		if len(s) > m.width {
			return s[:m.width-1]
		}
		return s
	}
	render := func(parts []string) string {
		const maxHints = 6
		if len(parts) > maxHints {
			parts = parts[:maxHints]
		}
		row := truncate(strings.Join(parts, "  "))
		return ui.StyleStatusBar.Render(renderHintKeys(row))
	}
	withMore := func(parts []string) []string {
		more := "[" + k.showHints + "/o" + "] More hints/config"
		if len(parts) >= 6 {
			parts = parts[:5]
		}
		return append(parts, more)
	}

	// Transient messages keep the hint cycler visible so [?] stays discoverable.
	if m.statusMsg != "" {
		return ui.StyleSuccess.Render(truncate("  " + m.statusMsg + "  [" + k.showHints + "/o" + "] More hints/config"))
	}

	if m.mode == ModeSearch {
		if m.searchInputActive {
			switch m.hintsMode {
			case HintsNavigation:
				return render(withMore([]string{"[-r]Recursive", "[-t]Contents", "[-z]Zoxide", "[Esc]Cancel"}))
			case HintsActions:
				return render(withMore([]string{"[Enter]Run full search", "[Esc]Cancel"}))
			default:
				return render(withMore([]string{"[Type]Filter", "[Enter]Run search", "[-r/-t/-z]Flags", "[Esc]Cancel"}))
			}
		}

		switch m.hintsMode {
		case HintsNavigation:
			return render(withMore([]string{"[" + k.left + "]Collapse", "[" + k.right + "]Expand", "[Esc]Edit query", "[" + k.fullSearch + "]Rerun"}))
		case HintsActions:
			return render(withMore([]string{"[" + k.confirm + "]Edit/Open", "[" + k.cdDir + "]Cd", "[Esc]Edit query", "[" + k.quit + "]Exit search"}))
		default:
			return render(withMore([]string{"[" + k.up + "/" + k.down + "]Nav", "[" + k.confirm + "]Edit/Open", "[" + k.cdDir + "]Cd", "[" + k.left + "/" + k.right + "]Collapse/Expand", "[Esc]Edit query"}))
		}
	}

	if m.mode == ModeRecents {
		switch m.hintsMode {
		case HintsNavigation:
			return render(withMore([]string{"[" + k.switchTabs + "]Bookmarks", "[" + k.switchTabsGlobal + "]Global", "[Esc]Back"}))
		case HintsActions:
			return render(withMore([]string{"[" + k.delete + "]Remove", "[" + k.quit + "]Back"}))
		default:
			return render(withMore([]string{"[" + k.up + "/" + k.down + "]Nav", "[" + k.confirm + "]Open", "[" + k.delete + "]Remove", "[Esc]Back"}))
		}
	}

	if m.mode == ModeBookmarks {
		switch m.hintsMode {
		case HintsNavigation:
			return render(withMore([]string{"[" + k.switchTabsGlobal + "]Global", "[" + k.switchTabs + "]Close", "[Esc]Back"}))
		case HintsActions:
			return render(withMore([]string{"[" + k.bookmark + "]Add", "[" + k.delete + "]Remove", "[" + k.rename + "]Rename"}))
		default:
			return render(withMore([]string{"[" + k.up + "/" + k.down + "]Nav", "[" + k.confirm + "]Open", "[" + k.delete + "]Remove", "[Esc]Back"}))
		}
	}

	if m.mode == ModeUpdates {
		return render(withMore([]string{"[↑/↓]Nav", "[Enter]Details", "[i]Install selected", "[y]Install latest", "[ctrl+f]Refresh", "[Esc]Back"}))
	}

	if m.mode == ModePlugins {
		return render(withMore([]string{"[" + k.up + "/" + k.down + "]Nav", "[" + k.confirm + "]Toggle", "[Esc]Back"}))
	}

	// ── Normal mode ───────────────────────────────────────────────────────────

	listLabel := "dirs"
	if m.listMode == ListDirsAndFiles {
		listLabel = "all"
	}

	switch m.hintsMode {
	case HintsNavigation:
		previewHint := "[" + k.previewToggle + "]Preview"
		if m.previewVisible {
			previewHint = "[" + k.previewToggle + "]Preview  [" + k.previewMode + "]Image/Details"
		}
		return render(withMore([]string{"[" + k.pageUp + "/" + k.pageDown + "]Page", "[" + k.jumpTop + "/" + k.jumpBottom + "]Top/Bot", "[" + k.searchKey + "]Search", "[" + k.plugins + "]Plugins", "[" + k.toggleList + "]Files:" + listLabel, previewHint, "[" + k.showUpdates + "]Updates"}))
	case HintsActions:
		hints := []string{"[" + k.add + "]Add", "[" + k.delete + "]Delete", "[" + k.rename + "]Rename", "[" + k.yank + "/" + k.cut + "]Yank/Cut", "[" + k.paste + "]Paste"}
		if m.clipboardPath == "" {
			hints[4] = "[" + k.copyPath + "]Copy path"
		}
		return render(withMore(hints))
	default: // HintsFull
		return render(withMore([]string{"[" + k.up + "/" + k.down + "/" + k.left + "/" + k.right + "]Nav", "[" + k.confirm + "]Expand/Edit", "[" + k.edit + "]Edit", "[" + k.openExplorer + "]Explorer", "[" + k.cdDir + "]Cd"}))
	}
}

func renderHintKeys(row string) string {
	var b strings.Builder
	for len(row) > 0 {
		start := strings.Index(row, "[")
		if start < 0 {
			b.WriteString(row)
			break
		}
		b.WriteString(row[:start])
		row = row[start:]
		end := strings.Index(row, "]")
		if end < 0 {
			b.WriteString(row)
			break
		}
		b.WriteString(ui.StyleHintKey.Render(row[:end+1]))
		row = row[end+1:]
	}
	return b.String()
}

func shortCommit(hash string) string {
	if len(hash) > 7 {
		return hash[:7]
	}
	if hash == "" {
		return "unknown"
	}
	return hash
}

func indentLines(s string, prefix string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i, line := range lines {
		lines[i] = prefix + strings.TrimSpace(line)
	}
	return strings.Join(lines, "\n")
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
