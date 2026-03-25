package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/wingitman/listicles/internal/fs"
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

	if m.mode != ModeSearchResult {
		crumbs := m.renderParentCrumbs()
		if crumbs != "" {
			b.WriteString(crumbs)
		}
	}

	if m.mode == ModeSearch {
		b.WriteString(m.renderSearchBar())
		b.WriteString("\n")
	}

	if m.mode == ModeSearchResult {
		b.WriteString(m.renderSearchResultHeader())
		b.WriteString("\n")
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
	if m.mode == ModeSearchResult {
		badges = append(badges, ui.StyleInputPrompt.Render("[search]"))
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

	countStr := ""
	if len(m.searchLiveNodes) > 0 {
		max := m.cfg.Display.SearchMaxResults
		if len(m.searchLiveNodes) >= max {
			countStr = ui.StyleMuted.Render(fmt.Sprintf("  %d+ matches", max))
		} else {
			countStr = ui.StyleMuted.Render(fmt.Sprintf("  %d match(es)", len(m.searchLiveNodes)))
		}
	} else if strings.TrimSpace(raw) != "" {
		bareQuery := strings.TrimSpace(raw)
		for _, f := range []string{"-rt", "-tr", "-r", "-t"} {
			bareQuery = strings.ReplaceAll(bareQuery, f, "")
		}
		if strings.TrimSpace(bareQuery) != "" {
			countStr = ui.StyleMuted.Render("  no matches")
		}
	}

	label := ui.StyleInputPrompt.Render("/") + flags + toolBadge + "  "
	bar := label + m.textInput.View() + countStr
	hint := ui.StyleMuted.Render("  Enter for full search  ·  -r recursive  ·  -t content  ·  -rt both  ·  Esc cancel")
	return bar + "\n" + hint
}

func (m Model) renderSearchResultHeader() string {
	if m.searchRunning {
		return ui.StyleMuted.Render("  Searching…")
	}
	count := len(m.nodes)
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

// ─── Node list ────────────────────────────────────────────────────────────────

func (m Model) renderNodes() string {
	nodes := m.nodes
	if m.mode == ModeSearch && m.searchLiveNodes != nil {
		nodes = m.searchLiveNodes
	}

	if len(nodes) == 0 {
		if m.mode == ModeSearch {
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
		if nodes[i].Depth == focusedDepth {
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

	// Number label: only for nodes at the focused depth
	numLabel := " · "
	if node.Depth == focusedDepth && siblingIdx < 99 {
		numLabel = fmt.Sprintf(" %d ", siblingIdx+1)
	}

	// Indent
	crumbDepth := 0
	if m.cfg.Display.ParentDepth > 0 && m.mode != ModeSearchResult {
		crumbDepth = m.cfg.Display.ParentDepth + 1
	}
	totalIndent := crumbDepth + node.Depth
	indent := strings.Repeat("  ", totalIndent)

	// Icon
	icon := "  "
	if e.IsDir() {
		if node.Expanded {
			icon = " ▼ "
		} else {
			icon = " ▶ "
		}
	}

	// Display name
	displayName := e.Name
	if m.mode == ModeSearchResult {
		if rel, err := filepath.Rel(m.prevRootDir, e.Path); err == nil {
			displayName = rel
			if e.Name != filepath.Base(e.Path) {
				displayName = rel + strings.TrimPrefix(e.Name, filepath.Base(e.Path))
			}
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

	// Style name
	var nameStr string
	if e.IsDir() {
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

	if m.mode == ModeSearchResult {
		bar := strings.Join([]string{
			k.up + "/" + k.down + " navigate",
			k.confirm + " cd/open",
			k.searchKey + " new search",
			"esc clear",
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
		k.confirm + " cd/open",
		k.searchKey + " search",
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
