package app

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

	switch m.mode {
	case ModeRecents:
		b.WriteString(m.renderRecentsHeader())
		b.WriteString("\n")
	case ModeBookmarks:
		b.WriteString(m.renderBookmarksHeader())
		b.WriteString("\n")
	case ModeSearch:
		if m.cfg.Display.ParentDepth > 0 {
			crumbs := m.renderParentCrumbs()
			if crumbs != "" {
				b.WriteString(crumbs)
			}
		}
		b.WriteString(m.renderSearchBar())
		b.WriteString("\n")
	default:
		crumbs := m.renderParentCrumbs()
		if crumbs != "" {
			b.WriteString(crumbs)
		}
	}

	b.WriteString(m.renderNodes())
	b.WriteString("\n")
	b.WriteString(m.renderOverlay())

	// Clipboard indicator line (above status bar)
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
	_, recursive, textMode := parseSearchFlags(raw)

	flags := ""
	if recursive {
		flags += ui.StyleSuccess.Render(" -r")
	}
	if textMode {
		flags += ui.StyleSuccess.Render(" -t")
	}

	toolBadge := ""
	if textMode && m.searchTools.HasRg {
		toolBadge = ui.StyleMuted.Render(" [rg]")
	} else if !textMode && m.searchTools.HasFd {
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
		hint = ui.StyleMuted.Render("  Enter search  ·  -r recursive  ·  -t content  ·  -rt both  ·  Esc cancel")
	} else if len(m.searchLiveNodes) > 0 {
		// Navigation state: show how to navigate and confirm.
		hint = ui.StyleMuted.Render(fmt.Sprintf(
			"  Enter open  ·  %s/%s navigate  ·  Esc edit query  ·  %s exit",
			m.keys.up, m.keys.down, m.keys.quit,
		))
	} else {
		// Navigation state but no results: prompt to edit query.
		hint = ui.StyleMuted.Render(fmt.Sprintf(
			"  Esc edit query  ·  %s exit",
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
	}
	scopeStr := "current dir"
	if m.searchRecursive {
		scopeStr = "recursive"
	}
	toolStr := ""
	if m.searchTextMode && m.searchTools.HasRg {
		toolStr = " via rg"
	} else if !m.searchTextMode && m.searchTools.HasFd {
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
		"  %s global  ·  %s bookmarks  ·  %s remove  ·  Esc back",
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
		"  %s global  ·  %s close  ·  %s add  ·  %s remove  ·  %s rename  ·  Esc back",
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
		name := e.Name
		if len(name) > m.width-4 {
			name = "…" + name[len(name)-(m.width-5):]
		}
		line := ui.StyleParentCrumb.Render("  " + name)
		if m.width > lipgloss.Width(line)+2 {
			line += ui.StyleMuted.Render(strings.Repeat("─", m.width-lipgloss.Width(line)-2))
		}
		return line
	}

	// ── Text-search match child (line snippet)
	if node.IsTextMatch {
		lineNum := ui.StyleNumber.Render(fmt.Sprintf(" :%d  ", node.MatchLineNum))
		snippet := ui.StyleMuted.Render(node.MatchSnippet)
		line := lineNum + snippet
		if idx == m.cursor {
			lineWidth := lipgloss.Width(line)
			if lineWidth < m.width {
				line = line + strings.Repeat(" ", m.width-lineWidth)
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

	maxW := m.width - 1
	if maxW < 10 {
		maxW = 10
	}
	if lipgloss.Width(line) > maxW {
		line = indent + numStyled + icon + nameStr
	}

	// Highlight selected row
	if idx == m.cursor {
		lineWidth := lipgloss.Width(line)
		if lineWidth < m.width {
			line = line + strings.Repeat(" ", m.width-lineWidth)
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
		lineWidth := lipgloss.Width(line)
		if lineWidth < m.width {
			line = line + strings.Repeat(" ", m.width-lineWidth)
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
		}
		box := ui.StyleConfirmBox.Render(
			ui.StyleInputPrompt.Render(label) + "\n\n" +
				m.textInput.View() + "\n\n" +
				ui.StyleMuted.Render("Enter to confirm · Esc to cancel"),
		)
		return box + "\n"
	}
	return ""
}

// ─── Clipboard bar ────────────────────────────────────────────────────────────

func (m Model) renderClipboardBar() string {
	op := "copy"
	if m.clipboardOp == ClipCut {
		op = "cut"
	}
	return ui.StyleClipboard.Render(fmt.Sprintf(
		"  [%s] %s  ·  %s paste  ·  press %s/%s again to clear",
		op,
		filepath.Base(m.clipboardPath),
		m.keys.paste,
		m.keys.yank,
		m.keys.cut,
	))
}

// ─── Status bar ───────────────────────────────────────────────────────────────

func (m Model) renderStatusBar() string {
	// Transient message overrides status bar
	if m.statusMsg != "" {
		return ui.StyleSuccess.Render("  " + m.statusMsg)
	}

	k := m.keys

	if m.mode == ModeRecents {
		bar := strings.Join([]string{
			k.up + "/" + k.down + " navigate",
			k.confirm + " open",
			k.delete + " remove",
			k.switchTabsGlobal + " global",
			k.switchTabs + " bookmarks",
			"esc back",
		}, "  ·  ")
		if len(bar) > m.width {
			bar = bar[:m.width-1]
		}
		return ui.StyleStatusBar.Render(bar)
	}

	if m.mode == ModeBookmarks {
		bar := strings.Join([]string{
			k.up + "/" + k.down + " navigate",
			k.confirm + " open",
			k.bookmark + " add",
			k.delete + " remove",
			k.rename + " rename",
			k.switchTabsGlobal + " global",
			k.switchTabs + " close",
			"esc back",
		}, "  ·  ")
		if len(bar) > m.width {
			bar = bar[:m.width-1]
		}
		return ui.StyleStatusBar.Render(bar)
	}

	entry := m.selectedEntry()
	entryDesc := ""
	if entry != nil {
		if entry.IsDir() {
			entryDesc = "dir"
		} else {
			entryDesc = "file"
		}
	}

	detailLabel := []string{"none", "count", "size", "path"}[m.detailLevel]
	listLabel := "dirs"
	if m.listMode == ListDirsAndFiles {
		listLabel = "all"
	}

	navKeys := k.up + "/" + k.down + "/" + k.left + "/" + k.right

	parts := []string{
		navKeys + " nav",
		"1-N jump",
		k.pageUp + "/" + k.pageDown + " page",
		k.jumpTop + "/" + k.jumpBottom + " top/bot",
		k.confirm + " expand",
		k.cdDir + " cd",
		k.openExplorer + " explorer",
		k.searchKey + " search",
		k.switchTabs + " recents",
		k.bookmark + " bookmark",
		k.quit + " quit",
		k.details + " detail:" + detailLabel,
		k.toggleList + " files:" + listLabel,
		k.add + " add",
		k.delete + " del",
		k.rename + " rename",
		k.edit + " edit",
		k.yank + " yank",
		k.cut + " cut",
		k.paste + " paste",
		k.copyPath + " copy path",
		k.options + " opts",
	}
	if m.gitRoot != "" {
		parts = append(parts, k.ignore+" ignore")
	}
	if entryDesc != "" {
		parts = append([]string{entryDesc}, parts...)
	}

	bar := strings.Join(parts, "  ·  ")
	if len(bar) > m.width {
		bar = bar[:m.width-1]
	}
	return ui.StyleStatusBar.Render(bar)
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
