# listicles — Technical Reference

This document describes the internal architecture, data model, key flows, and test infrastructure of listicles. It is intended for developers reading or modifying the source code.

---

## Table of Contents

1. [Overview](#1-overview)
2. [Architecture](#2-architecture)
3. [Directory Structure](#3-directory-structure)
4. [Package Reference](#4-package-reference)
   - [internal/config](#41-internalconfig)
   - [internal/fs](#42-internalfs)
   - [internal/search](#43-internalsearch)
   - [internal/ui](#44-internalui)
   - [internal/app — model](#45-internalapp--model)
   - [internal/app — view](#46-internalapp--view)
5. [Data Model Deep Dive — The Flat Tree](#5-data-model-deep-dive--the-flat-tree)
6. [Key Flows](#6-key-flows)
7. [Shell Integration](#7-shell-integration)
8. [Configuration Reference](#8-configuration-reference)
9. [CLI Flags](#9-cli-flags)
10. [Building](#10-building)
11. [Running the Tests](#11-running-the-tests)
12. [Test Architecture](#12-test-architecture)
13. [Adding New Features — Conventions](#13-adding-new-features--conventions)

---

## 1. Overview

listicles is an interactive terminal file explorer written in Go. The user types `l` in their shell, navigates a tree of directories using keyboard shortcuts, and pressing Enter actually changes the shell's working directory to wherever they ended up — a trick implemented via a temp file and a shell wrapper function.

**Language:** Go 1.26.1  
**TUI framework:** [BubbleTea](https://github.com/charmbracelet/bubbletea) (Elm architecture)  
**Styling:** [Lipgloss](https://github.com/charmbracelet/lipgloss)  
**Config:** [BurntSushi/toml](https://github.com/BurntSushi/toml)  
**Clipboard:** [atotto/clipboard](https://github.com/atotto/clipboard)  
**Binary:** ~4 MB, statically linked, no runtime dependencies  

---

## 2. Architecture

listicles follows the [Elm architecture](https://guide.elm-lang.org/architecture/) as implemented by BubbleTea:

```
┌─────────────────────────────────────────────────────────┐
│                     tea.Program                         │
│                                                         │
│   ┌──────────┐    msg     ┌──────────┐                  │
│   │  Update  │ ◄────────  │  Events  │ (keyboard, size) │
│   └────┬─────┘            └──────────┘                  │
│        │ new Model + Cmd                                │
│        ▼                                                │
│   ┌──────────┐    Cmd     ┌──────────┐                  │
│   │   View   │            │  Effects │ (goroutines,     │
│   └────┬─────┘            └──────────┘  tea.Tick, etc.) │
│        │ string                                         │
│        ▼                                                │
│   ┌──────────┐                                          │
│   │ Terminal │ (alt-screen, mouse enabled)              │
│   └──────────┘                                          │
└─────────────────────────────────────────────────────────┘
```

**Model** (`internal/app/model.go`) holds all application state: the flat tree of visible nodes, cursor position, current mode, clipboard, search state, keybind map, terminal dimensions, and more.

**Update** (`model.go: Update()`) is a pure function that receives a `tea.Msg` and returns a new `Model` plus an optional `tea.Cmd`. All state mutations happen here. Side effects (filesystem operations, subprocesses, clipboard writes) are either executed synchronously inside Update or dispatched as `tea.Cmd` goroutines.

**View** (`internal/app/view.go`) is a pure function from `Model` to `string`. It renders the entire screen on every frame. Lipgloss handles ANSI colour codes and string width calculations.

**Alt-screen mode:** the program runs in an alternate terminal buffer (`tea.WithAltScreen()`). When it exits, the original shell buffer is restored. Mouse cell motion is enabled (`tea.WithMouseCellMotion()`) but not currently used for navigation — it is included for future compatibility.

**The cd mechanism:** BubbleTea programs cannot change the parent shell's working directory directly (they run in a child process). Instead, the binary accepts `--cd-file <path>` and writes the chosen directory to that file on exit. The shell wrapper function reads the file and calls `builtin cd`. See [Shell Integration](#7-shell-integration).

---

## 3. Directory Structure

```
listicles/
│
├── main.go                 Entry point — parses flags, loads config,
│                           constructs app.Model, runs tea.Program
│
├── integration_test.go     End-to-end integration tests (build tag: integration)
│
├── go.mod                  Module: github.com/wingitman/listicles, Go 1.26.1
├── go.sum
├── Makefile                build / install / test targets
├── README.md               User-facing installation and usage guide
├── TECHNICAL.md            This document
├── LICENSE                 MIT, copyright 2026 delbysoft
├── .gitignore              Ignores bin/
│
├── bin/                    Compiled binary (gitignored)
│   └── listicles
│
├── shell/                  Shell wrapper functions
│   ├── listicles.bash      For bash (~/.bashrc)
│   ├── listicles.zsh       For zsh (~/.zshrc)
│   ├── listicles.fish      For fish (~/.config/fish/config.fish)
│   └── listicles.ps1       For PowerShell pwsh (~/.config/powershell/Microsoft.PowerShell_profile.ps1)
│
└── internal/               All application code (not importable externally)
    │
    ├── app/
    │   ├── model.go        BubbleTea Model: all state + Update dispatch
    │   ├── view.go         BubbleTea View: all rendering logic
    │   ├── model_test.go   Unit tests for model logic (package app)
    │   └── view_test.go    Unit tests for rendering (package app)
    │
    ├── config/
    │   ├── config.go       TOML config structs, Load(), WriteDefault()
    │   └── config_test.go
    │
    ├── fs/
    │   ├── fs.go           Filesystem operations: scan, create, delete,
    │   │                   rename, copy, stats, human size
    │   └── fs_test.go
    │
    ├── search/
    │   ├── search.go       Search via fd/rg with find/grep fallback
    │   └── search_test.go
    │
    └── ui/
        └── styles.go       Lipgloss style variables (no functions)
```

**Dependency graph** (internal packages only):

```
main
  ├── internal/config
  └── internal/app
        ├── internal/config
        ├── internal/fs
        ├── internal/search
        │     └── internal/fs
        └── internal/ui
```

`internal/config`, `internal/fs`, and `internal/ui` have no internal dependencies. `internal/search` depends only on `internal/fs` (for the `Entry` type). `internal/app` is the only package that depends on everything.

---

## 4. Package Reference

### 4.1 `internal/config`

**File:** `internal/config/config.go`

Responsible for loading and writing the user's configuration file.

#### Config file location

```
~/.config/delbysoft/listicles.toml
```

`ConfigDir()` returns `~/.config/delbysoft`.  
`ConfigPath()` returns `~/.config/delbysoft/listicles.toml`.  
Both use `os.UserHomeDir()`.

#### `Load()` behaviour

```
Load()
  → os.Stat(ConfigPath())
  → File does not exist:
      os.MkdirAll(ConfigDir(), 0755)
      WriteDefault(ConfigPath())   ← writes commented TOML
      return Default(), nil
  → File exists:
      toml.DecodeFile(path, cfg)   ← unknown fields silently ignored
      clamp: SearchMaxResults = max(1, cfg.Display.SearchMaxResults)
      clamp: ParentDepth      = max(0, cfg.Display.ParentDepth)
      return cfg, nil
```

The TOML decoder (`BurntSushi/toml`) silently ignores unknown fields. This means stale config entries from older versions (e.g. the removed `vim_mode` key and `[vim_mode]` section) do not cause errors.

On decode error, `Load()` returns `Default()` with the error — the app always has a valid config.

#### Types

**`Keybinds`** — 24 string fields. Each maps to a BubbleTea key name:
- Named keys: `"up"`, `"down"`, `"left"`, `"right"`, `"enter"`, `"pgup"`, `"pgdown"`, `"home"`, `"end"`, `"esc"`
- Single characters: `"q"`, `"a"`, `"/"`, `"Y"`, etc.
- Modifier combos: `"ctrl+u"`, `"ctrl+d"` (for vim-style remapping)

**`Display`** — four fields:
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `ShowHidden` | bool | false | Show dotfiles |
| `DefaultListMode` | string | `"dirs"` | `"dirs"` or `"dirs_and_files"` |
| `SearchMaxResults` | int | 20 | Max live filter results (min 1) |
| `ParentDepth` | int | 1 | Greyed ancestor lines above tree (min 0) |

**`Apps`** — two string fields:
| Field | Default | Description |
|-------|---------|-------------|
| `Editor` | `""` | Falls back to `$EDITOR`, then `$VISUAL`, then `nano`/`vi`/`vim` |
| `Opener` | `""` | Falls back to `xdg-open` (Linux) or `open` (macOS) |

---

### 4.2 `internal/fs`

**File:** `internal/fs/fs.go`

All filesystem I/O is isolated here. No other package performs direct filesystem operations except `internal/app` (which calls `internal/fs` functions) and `internal/search` (which runs external subprocesses).

#### `Entry` and `EntryType`

```go
type EntryType int   // EntryDir = 0, EntryFile = 1

type Entry struct {
    Name     string
    Path     string    // absolute path
    Type     EntryType
    Size     int64     // bytes (files only; dirs always 0)
    NumFiles int       // unused in current display (populated by DirStats)
    NumDirs  int       // unused in current display
}
```

#### `ScanDir` — sorting rules

Entries are sorted in two passes:
1. **Type first:** all `EntryDir` (0) entries before all `EntryFile` (1) entries
2. **Within each group:** case-insensitive alphabetical (`strings.ToLower`)

Hidden entries (names starting with `.`) are excluded when `showHidden=false`. Files are excluded entirely when `showFiles=false` (the default `dirs`-only mode).

#### `HumanSize` — scale thresholds

```
< 1024          → "N B"
< 1024²         → "N.N KB"
< 1024³         → "N.N MB"
< 1024⁴         → "N.N GB"
...             → TB, PB, EB
```

One decimal place. Uses binary (1024-based) prefixes.

#### `CopyEntry` — recursion strategy

`CopyEntry(src, dstDir)` copies `src` to `dstDir/basename(src)`.

For files: `os.Open` → `os.OpenFile(O_CREATE|O_WRONLY|O_TRUNC)` → `io.Copy`. Permissions are preserved via the source file's `os.FileMode`.

For directories: `filepath.WalkDir(src, ...)` visits every node in the source tree. Directories are recreated with `os.MkdirAll`. Files are copied with `copyFile`. The `dst` path for each node is computed as `filepath.Join(dst, rel)` where `rel` is the relative path from `src`.

---

### 4.3 `internal/search`

**File:** `internal/search/search.go`

Handles the two search modes (name search and text-in-file search) and manages tool detection.

#### Tool detection

`DetectTools()` calls `exec.LookPath("fd")` and `exec.LookPath("rg")` once at startup and caches the results in a `Tools` struct. This avoids repeated PATH lookups during interactive search.

#### Tool selection matrix

| Mode | `HasFd` | `HasRg` | Command used |
|------|---------|---------|--------------|
| Name search | true | — | `fd --glob *query* [--hidden] [--max-depth 1] dir` |
| Name search | false | — | `find dir [-maxdepth 1] [-not -path */.*] -iname *query*` |
| Text search | — | true | `rg --line-number --no-heading --color never [--hidden] [--max-depth 1] query dir` |
| Text search | — | false | `grep -n query <files>` (non-recursive) or `grep -rn query dir` |

The `--max-depth 1` flag is omitted when `Recursive=true`.

#### `streamLines`

The core I/O helper: starts `cmd` with `cmd.StdoutPipe()`, reads stdout line-by-line with `bufio.Scanner`, calls the callback for each line, then calls `cmd.Wait()`. Non-zero exit codes are ignored — search tools return exit code 1 when no matches are found, which is not an error.

#### `ResultsToEntries` — dedup logic

Results are deduplicated by a key:
- Name-search results: key = `result.Path`
- Text-search results with `LineNum > 0`: key = `result.Path + ":" + lineNum`

This means multiple matches in the same file produce one entry per matching line, while name-search results for the same path are collapsed to one.

The `Name` field of text-match entries is set to `"basename:lineNum"` so the line number is visible in the result list.

---

### 4.4 `internal/ui`

**File:** `internal/ui/styles.go`

A collection of package-level Lipgloss style variables. No functions. All styles are referenced by name throughout `view.go`.

#### Colour palette

| Variable | Hex | Usage |
|----------|-----|-------|
| `colorPrimary` | `#7C9EF0` | Directory names, path header, header badges |
| `colorAccent` | `#F0A47C` | Row numbers, input prompts, confirm box borders |
| `colorMuted` | `#666688` | Divider rule, scroll indicators, status bar, hints |
| `colorError` | `#F07C7C` | Error overlay title, "No results" text |
| `colorSuccess` | `#7CF09C` | Flag active indicators (`-r`, `-t`), result count |
| `colorFile` | `#B0B0CC` | File name text |
| `colorBorder` | `#444466` | Generic bordered box |
| `colorSelected` | `#2A2A4A` | Selected row background (deep navy) |
| `colorHeaderBg` | `#1A1A2E` | `StyleHeader` background (currently unused in view) |

#### Key styles and their usage in `view.go`

| Style | Where used |
|-------|-----------|
| `StylePath` | Header path string |
| `StyleSelected` | Highlighted row (cursor position) — full-width background |
| `StyleDirName` | Directory name with `/` suffix |
| `StyleFileName` | File name |
| `StyleNumber` | `" 1 "` through `" 9 "` labels; digit buffer `→ N` in header |
| `StyleMuted` | Divider rule, `↑ N more above`, scroll hints, status bar text |
| `StyleError` | Error overlay title, "No results" prefix |
| `StyleSuccess` | `-r`/`-t` flag badges, `N result(s)` in search result header |
| `StyleConfirmBox` | All modal overlays (error, confirm, input, clipboard prompt) |
| `StyleInputPrompt` | Input label text, search `/` prefix |
| `StyleDetail` | Italic detail suffix (count/size/path) |
| `StyleParentCrumb` | Greyed ancestor directory lines |
| `StyleRootDir` | Root directory label line (slightly brighter than crumbs) |
| `StyleClipboard` | Clipboard bar and `[copy]`/`[cut]` suffixes on entries |

---

### 4.5 `internal/app` — model

**File:** `internal/app/model.go`

The heart of the application. Contains all state, the BubbleTea `Update` method, and every action handler.

#### Mode state machine

The `Mode` integer field controls which input handlers are active:

```
              ┌─────────────────────────────────────────────┐
              │                  ModeNormal                  │
              │                   (default)                  │
              └──┬──────────────────────────────────────────┘
                 │
        ┌────────┼───────────┬────────────┬────────────┐
        ▼        ▼           ▼            ▼            ▼
  ModeConfirm ModeInput  ModeSearch  ModeError  ModeSearchResult
  (y/n prompt) (text box) (live filter) (overlay) (result list)
        │        │           │                        │
        │ y/n    │ enter/esc  │ enter/esc              │ esc/q
        └────────┴───────────┴────────────────────────┘
              all → ModeNormal (or ModeSearchResult after search)
```

**ModeNormal** — all navigation keys, action keys active.

**ModeConfirm** — `y`/`Y` executes `executeConfirmedAction()`; any other key cancels and returns to `ModeNormal`. Confirm actions: `ConfirmDelete`, `ConfirmRename`, `ConfirmPasteCopy`, `ConfirmPasteMove`.

**ModeInput** — the `textinput.Model` receives all keys except `enter` (submit) and `esc` (cancel). Input actions: `InputAdd`, `InputRename`. On `enter`, `submitInput()` is called; for `InputRename` this transitions to `ModeConfirm` rather than executing directly.

**ModeSearch** — same as `ModeInput` but every keystroke also calls `applyLiveFilter()` to update `searchLiveNodes` in real time. `enter` calls `executeSearch()` which launches a subprocess goroutine and returns immediately; results arrive via `searchResultMsg`. `esc` restores the saved `prevNodes`.

**ModeSearchResult** — a simplified input handler: only up/down navigation, `enter` to cd/open, `/` to start a new search, and `esc`/`q` to clear results and restore the tree.

**ModeError** — any key clears the error and returns to `ModeNormal`.

#### `resolvedKeys`

On construction, all `config.Keybinds` strings are copied into a `resolvedKeys` struct. This avoids map lookups on every keypress and makes the key names available as named fields in all handlers. `matchKey(pressed, binding)` is a simple `pressed == binding` string comparison.

#### Internal message types

| Type | Sent by | Handled in Update |
|------|---------|-------------------|
| `errorMsg` | Various error paths | Sets `m.errorMsg`, enters `ModeError` |
| `reloadMsg` | `openEditor` via `tea.ExecProcess` callback | Calls `refreshExpandedNode` or `initTree` |
| `searchResultMsg` | `executeSearch` goroutine | Populates `m.nodes` from results, enters `ModeSearchResult` |
| `digitTimeoutMsg` | `tea.Tick(400ms)` | Calls `resolveDigitBuffer`, clears `digitBuffer` |
| `clearStatusMsg` | `tea.Tick(1400ms)` | Clears `m.statusMsg` |

#### Multi-digit navigation (`digitBuffer`)

When a digit key (`0`–`9`) is pressed in `ModeNormal`:

- `"0"` with an empty buffer immediately triggers `navigateLeft()` (go to parent). This preserves `0` as the parent key without conflicting with multi-digit numbers starting with `0x`.
- Any other digit appends to `m.digitBuffer` and fires `tea.Tick(400ms)`.
- If another digit arrives before the tick, it appends and resets the timer.
- When the tick fires, `resolveDigitBuffer()` is called:
  1. Parse `m.digitBuffer` as a 1-based integer `n`
  2. Call `nthSiblingAtDepth(m.nodes, focusedDepth, m.offset, n-1)` to find the target index
  3. Move cursor there; expand if it's a directory (with cursor advance to first child)
  4. Clear `m.digitBuffer`

The current buffer is shown live in the header as `→ N` via `StyleNumber`.

#### Page jump formula

```go
func calcPageJump(n int) int {
    j := int(math.Round(math.Log2(float64(n))))
    if j < 1 { j = 1 }
    return j
}
```

| Visible items | Jump size |
|--------------|-----------|
| 1–2 | 1 |
| 3–5 | 2 |
| 6–11 | 3 |
| 12–22 | 4 |
| 23–45 | 5 |
| 46–90 | 6 |

This gives natural-feeling jumps that scale with directory size without requiring manual configuration.

#### Clipboard state machine

```
clipboardOp = ClipNone

Press y on entry X:
  if clipboardPath == X && clipboardOp == ClipCopy  → clear (ClipNone)
  else                                               → clipboardPath = X, ClipCopy

Press x on entry X:
  if clipboardPath == X && clipboardOp == ClipCut   → clear (ClipNone)
  else                                               → clipboardPath = X, ClipCut

Press p (with non-empty clipboard):
  → set pendingPath = clipboardPath
  → set pendingDestDir = currentOperationDir()
  → set confirmAction = ConfirmPasteCopy or ConfirmPasteMove
  → enter ModeConfirm

After ConfirmPasteMove executes successfully:
  → clipboardPath = "", clipboardOp = ClipNone  (auto-clear after move)
```

Copy does not auto-clear the clipboard — users can paste multiple copies to different destinations.

#### `visibleRows` calculation

```
visibleRows = height
            - 2   (header line + rule)
            - (ParentDepth + 1)  (crumb lines + root label)
            - 1   (status bar)
            - 3   (margin)
            - 5   (if overlay is shown: ModeConfirm/ModeInput/ModeError)
            - 3   (if ModeSearch: search bar + hint line)
            - 1   (if clipboard bar is shown)
```

---

### 4.6 `internal/app` — view

**File:** `internal/app/view.go`

Implements `View() string` — called by BubbleTea on every state change to produce the full terminal output.

#### Render pipeline (in order)

```
View()
  1. renderHeader()          path + badges + divider rule
  2. renderParentCrumbs()    greyed ancestors (if ParentDepth > 0, not in SearchResult mode)
  3. renderSearchBar()       (if ModeSearch)
  4. renderSearchResultHeader() (if ModeSearchResult)
  5. renderNodes()           the main entry list
  6. renderOverlay()         modal (ModeError / ModeConfirm / ModeInput)
  7. renderClipboardBar()    (if clipboardPath is set)
  8. renderStatusBar()       key hint bar (or statusMsg if set)
```

#### Number label assignment in `renderNodes`

The number labels (`1`–`N`) are assigned only to nodes at the **focused depth** — the depth of the node currently under the cursor. This prevents root-level siblings from showing numbers while the cursor is inside an expanded subdirectory.

The algorithm in `renderNodes`:

```go
focusedDepth := nodes[m.cursor].Depth

siblingCount := 0
for i := m.offset; i < end; i++ {
    renderNode(i, nodes[i], focusedDepth, siblingCount)
    if nodes[i].Depth == focusedDepth {
        siblingCount++
    }
}
```

`renderNode` uses `siblingIdx` directly as the label index. If `node.Depth != focusedDepth`, the label is always ` · ` regardless of `siblingIdx`.

#### Indent calculation in `renderNode`

```go
crumbDepth := 0
if ParentDepth > 0 && not ModeSearchResult {
    crumbDepth = ParentDepth + 1   // ancestor lines + root label line
}
totalIndent := crumbDepth + node.Depth
indent := strings.Repeat("  ", totalIndent)
```

Each indent level is two spaces. The `crumbDepth` offset ensures tree nodes visually align below the root directory label line.

#### Parent crumb algorithm in `renderParentCrumbs`

The crumbs track the **cursor's current location**, not the fixed tree root. When the cursor is inside the tree (depth > 0), the function walks backward through `m.nodes` using `parentNodeIdx` to collect ancestor paths:

```
cursor at depth 3:
  parentNodeIdx(cursor)   → depth-2 ancestor path
  parentNodeIdx(that)     → depth-1 ancestor path
  parentNodeIdx(that)     → depth-0 root path
  stop (parentNodeIdx returns -1)

Take last min(ParentDepth, len(chain)) entries from chain.
Render as greyed italic lines with increasing indent.
Render the cursor's direct parent dir as the root label (StyleRootDir).
```

When the cursor is at depth 0 (or the directory is empty), the function walks up the real filesystem from `m.rootDir` instead.

---

## 5. Data Model Deep Dive — The Flat Tree

The entire visible directory tree is represented as a single flat slice `[]TreeNode`. This is the central data structure of the application.

### `TreeNode`

```go
type TreeNode struct {
    Entry    fs.Entry   // the filesystem entry
    Depth    int        // nesting level: 0 = root children
    Expanded bool       // true if this dir's children are currently inline
}
```

### Why a flat slice?

A flat slice makes rendering trivial: iterate from `m.offset` to `m.offset + visibleRows`, render each `TreeNode`. The `cursor` and `offset` are simple integers. Scroll logic is a single arithmetic clamp.

The trade-off is that expand/collapse operations require slice mutation (inserting or removing a run of elements). For typical directory sizes this is negligible.

### expandNode — inline insertion

```
Before: [parent(0), sibling(0), ...]
         cursor = 0

expandNode(0):
  load children via fs.ScanDir
  build []TreeNode at depth 1
  m.nodes = nodes[:1] + children + nodes[1:]
  m.nodes[0].Expanded = true

After:  [parent(0,expanded), child1(1), child2(1), sibling(0), ...]
```

The `m.nodes` backing array is replaced entirely on every expand. This means any pointer into `m.nodes` captured before an `expandNode` call (such as `node := m.selectedNode()`) is stale afterward. The `wasExpanded` and `parentDepth` variables in the right-key handler capture the needed values before the call to avoid this.

### collapseNode — range removal

```
collapseNode(idx):
  targetDepth = m.nodes[idx].Depth
  end = first index after idx where Depth <= targetDepth
  m.nodes = m.nodes[:idx+1] + m.nodes[end:]
  m.nodes[idx].Expanded = false
```

This removes all descendants regardless of how deeply nested they are — a subtree with multiple levels of expansion collapses entirely in one operation.

### rebuildTree — preserving expansion state

Used after filesystem mutations (add/delete/rename) that affect the root-level scan:

```
rebuildTree():
  expanded = set of paths where node.Expanded == true
  initTree(rootDir)            ← fresh flat list at depth 0
  for i := range nodes:
    if nodes[i].IsDir() && expanded[nodes[i].Entry.Path]:
      expandNode(i)            ← re-inserts children inline
```

Because `expandNode` inserts children after position `i`, the loop index `i` naturally advances past them on the next iteration (they are at `i+1`, `i+2`, etc., all with `Expanded=false`).

### parentNodeIdx — walking backward

```
parentNodeIdx(idx):
  childDepth = m.nodes[idx].Depth
  if childDepth == 0: return -1
  scan backward from idx-1:
    first node with Depth == childDepth-1 → return its index
```

This works correctly because of the insert-after invariant: a parent's children are always immediately contiguous after the parent node in the slice. The backward scan always finds the correct parent before encountering any unrelated nodes at the same parent depth.

---

## 6. Key Flows

### 6.1 Startup

```
main()
  → config.Load()
      → Config file missing: WriteDefault → create ~/.config/delbysoft/listicles.toml
      → Config file exists: toml.DecodeFile → clamp minimums
  → app.New(cfg, startDir, cdFile)
      → search.DetectTools()    ← probes PATH for fd and rg once
      → initTree(startDir)
          → fs.ScanDir(startDir, showHidden=false, showFiles=false)
          → populate m.nodes as []TreeNode at Depth=0, Expanded=false
      → return *Model
  → tea.NewProgram(model, WithAltScreen, WithMouseCellMotion)
  → p.Run()
      → sends tea.WindowSizeMsg (triggers first render with real dimensions)
      → renders: header + crumbs + node list + status bar
```

### 6.2 Pressing `→` on a directory

```
tea.KeyMsg{Type: tea.KeyRight}
  → Update(): ModeNormal branch
  → matchKey("right", m.keys.right) == true
  → node = m.selectedNode()   ← captures Expanded + Depth BEFORE expandNode
  → wasExpanded = node.Expanded   (false)
  → parentDepth = node.Depth      (e.g. 0)
  → expandNode(m.cursor)
      → fs.ScanDir(node.Entry.Path, ...)
      → insert children into m.nodes after cursor
      → m.nodes[cursor].Expanded = true
  → !wasExpanded == true:
      → m.cursor+1 < len(m.nodes)  (true if dir has children)
      → m.nodes[m.cursor+1].Depth > parentDepth  (depth 1 > 0: true)
      → m.cursor++               ← cursor moves to first child
      → m.adjustOffset()
  → return updated model
```

If the directory has no children, `expandNode` inserts nothing — `m.nodes[m.cursor+1]` is the next sibling (same or lesser depth), so the depth check fails and the cursor does not advance.

If the directory is already expanded, `expandNode` calls `collapseNode` instead, `wasExpanded` is `true`, and the cursor-advance branch is skipped.

### 6.3 Pressing `←` (navigateLeft)

Three cases based on the selected node:

**Case 1: Cursor on an expanded directory**
```
navigateLeft():
  node.Entry.IsDir() && node.Expanded → true
  collapseNode(m.cursor)     ← removes all children from slice
  return                     ← cursor stays on the dir
```

**Case 2: Cursor on a collapsed node at depth > 0**
```
navigateLeft():
  node.Depth > 0
  parentIdx = parentNodeIdx(m.cursor)   ← walks backward to depth-1 ancestor
  collapseNode(parentIdx)               ← removes all of parent's children
  m.cursor = parentIdx                  ← jump up to parent
  m.adjustOffset()
```

**Case 3: Cursor at depth 0 (or empty directory)**
```
navigateLeft():
  node == nil || node.Depth == 0
  goToParentDir():
    parent = fs.ParentDir(m.rootDir)
    if parent == m.rootDir: return  ← at filesystem root, no-op
    prevRoot = m.rootDir
    initTree(parent)                ← complete tree re-init at parent dir
    find node whose path == prevRoot
    m.cursor = that index           ← cursor lands on the dir we just left
    m.adjustOffset()
```

### 6.4 Multi-digit navigation

```
User presses "1":
  digitBuffer = "1"
  tea.Tick(400ms) → digitTimeoutMsg

User presses "9" within 400ms:
  digitBuffer = "19"
  tea.Tick(400ms) → digitTimeoutMsg  (previous tick still fires but buffer was already updated)

400ms passes without another digit:
  digitTimeoutMsg received:
  resolveDigitBuffer():
    n = 19  (1-based)
    focusedDepth = m.nodes[m.cursor].Depth
    target = nthSiblingAtDepth(nodes, focusedDepth, m.offset, 18)  (0-based)
    if target >= 0:
      m.cursor = target
      m.adjustOffset()
      if node.IsDir(): expandNode(target)  (with cursor-advance to first child)
  digitBuffer = ""
```

### 6.5 Live search filter

```
ModeSearch: user types a character
  → Update() ModeSearch branch
  → m.textInput.Update(msg)   ← textinput component processes keystroke
  → m.applyLiveFilter():
      raw = textInput.Value()
      parseSearchFlags(raw) → strip -r/-t/-rt/-tr → bare query
      if bare query == "":
        searchLiveNodes = nil
        return
      q = strings.ToLower(bareQuery)
      max = cfg.Display.SearchMaxResults
      for each node in prevNodes:
        if strings.Contains(strings.ToLower(node.Entry.Name), q):
          append to filtered
          if len(filtered) >= max: break
      searchLiveNodes = filtered
      m.cursor = 0; m.offset = 0
  → renderNodes() uses searchLiveNodes instead of nodes
```

`prevNodes` is the snapshot of the tree taken when `/` was pressed. The live filter never modifies `m.nodes` — it only populates `searchLiveNodes` as a separate display slice.

### 6.6 Full subprocess search

```
User presses Enter in ModeSearch:
  → executeSearch():
      parseSearchFlags(textInput.Value()) → (query, recursive, textMode)
      if query == "": restore prevNodes, ModeNormal, return
      build search.Request{Dir, Query, Recursive, TextMode, Hidden}
      searchRunning = true
      searchLiveNodes = nil
      return tea.Cmd:
        goroutine:
          var results []search.Result
          search.Run(tools, req, func(r) { results = append(...) })
            → builds subprocess (fd/find/rg/grep)
            → streamLines: reads stdout line by line → emit Result{}
          return searchResultMsg{results}

  → searchResultMsg received:
      searchRunning = false
      search.ResultsToEntries(results) → []fs.Entry (deduplicated)
      m.nodes = []TreeNode{...}  (depth-0, not expanded)
      m.cursor = 0; m.offset = 0
      m.mode = ModeSearchResult
```

### 6.7 Yank → navigate → paste → confirm

```
1. Press y on "file.txt":
   clipboardPath = "/path/to/file.txt"
   clipboardOp   = ClipCopy

2. Navigate to destination directory "dst/"

3. Press p:
   pendingPath    = "/path/to/file.txt"
   pendingDestDir = "/path/to/dst"
   confirmAction  = ConfirmPasteCopy
   confirmMsg     = "Copy "file.txt" → "/path/to/dst"?"
   mode           = ModeConfirm

4. Press y:
   executeConfirmedAction():
     fs.CopyEntry("/path/to/file.txt", "/path/to/dst")
       → dst = "/path/to/dst/file.txt"
       → copyFile(src, dst, mode)
     refreshExpandedNode(findNodeByPath("/path/to/dst"))
     statusMsg = "Copied "file.txt""
     tea.Tick(1400ms) → clearStatusMsg
```

### 6.8 Enter on a directory → shell `cd`

```
tea.KeyMsg{Type: tea.KeyEnter}
  → Update(): ModeNormal
  → matchKey("enter", m.keys.confirm) == true
  → e = m.selectedEntry()   (type: EntryDir)
  → m.exitWithDir(e.Path):
      os.WriteFile(m.cdFile, []byte(e.Path), 0600)
      return tea.Quit
  → tea.Program.Run() returns
  → main() returns (exit code 0)
  → shell wrapper:
      dir=$(cat "$tmp")    ← reads the written path
      rm -f "$tmp"
      builtin cd "$dir"    ← changes shell's cwd
```

If the user quits with `q` or `Esc`, `exitWithoutCD()` is called — it returns `tea.Quit` without writing anything to `cdFile`. The shell wrapper reads an empty or missing file and skips the `cd`.

---

## 7. Shell Integration

The fundamental problem: a Go binary is a child process of the shell. When it exits, any `os.Chdir()` calls it made affect only its own process — not the parent shell. There is no cross-process mechanism to change the parent's working directory.

**The solution:** the binary writes its chosen path to a temp file. The shell wrapper reads that file and runs `builtin cd` (the shell's own `cd` builtin, not a subprocess).

### Wrapper anatomy (bash/zsh)

```bash
l() {
    local tmp
    tmp=$(mktemp)                          # 1. create temp file
    listicles --cd-file "$tmp" "$@"        # 2. run the binary
                                           #    (blocks until binary exits)
    local dir
    dir=$(cat "$tmp" 2>/dev/null)          # 3. read the chosen path
    rm -f "$tmp"                           # 4. clean up
    if [ -n "$dir" ] && [ "$dir" != "$PWD" ]; then
        builtin cd "$dir" || return 1      # 5. change shell's cwd
    fi
}
```

`builtin cd` is required (not just `cd`) because some shells alias `cd` to a custom function (e.g. with `zoxide` or `autojump` integration) and the wrapper needs to call the raw shell builtin to avoid infinite recursion.

`"$@"` passes any arguments through to the binary (e.g. `l --dir /some/path`).

### Fish variant

Fish uses its own syntax (`set`, `test`, `and`) but the logic is identical. `builtin cd` is supported in fish as well.

### PowerShell (pwsh) variant

```powershell
function l {
    $tmp = [System.IO.Path]::GetTempFileName()   # 1. create temp file
    listicles --cd-file $tmp @args                # 2. run the binary
    $dir = Get-Content $tmp -ErrorAction SilentlyContinue  # 3. read chosen path
    Remove-Item $tmp -Force -ErrorAction SilentlyContinue  # 4. clean up
    if ($dir -and $dir -ne $PWD.Path) {
        Set-Location $dir                         # 5. change shell's cwd
    }
}
```

Key differences from bash/zsh:
- `[System.IO.Path]::GetTempFileName()` creates the temp file (unlike `mktemp`, this also writes the file on disk — listicles overwrites it on exit).
- `@args` is PowerShell's equivalent of `"$@"`.
- `Get-Content` / `Remove-Item` / `Set-Location` replace the POSIX builtins.
- `$PWD.Path` gives the string value of the current directory (plain `$PWD` is a `PathInfo` object).
- `Set-Location` is the PowerShell `cd` — no `builtin` prefix is needed since there is no shadowing concern.

`make install` detects `~/.config/powershell/Microsoft.PowerShell_profile.ps1` (the default cross-platform pwsh profile path) and dot-sources the wrapper with `. /path/to/listicles/shell/listicles.ps1`. To reload: `. $PROFILE`.

### Adding support for a new shell

The underlying pattern is the same for any shell that can create a temp file, invoke a subprocess, and `cd` inside a function. The constraint is always the same: **`cd` must run inside a shell function, not a child process**, because only the current shell process can change its own working directory.

Generic pseudocode:

```
function l(args):
    tmp = create_temp_file()
    run: listicles --cd-file <tmp> <args>   # blocks until exit
    dir = read_file(tmp)
    delete_file(tmp)
    if dir is non-empty and dir != current_dir:
        cd dir
```

Concrete steps to add a new shell:

1. Create `shell/listicles.<ext>` implementing the pattern above in the target shell's syntax.
2. Add a detection block to the `install-shell` Makefile target: check for the shell's rc file, grep for `"listicles shell integration"`, and append the source line if absent.
3. Update the reload hint at the bottom of `install-shell`.
4. Document the new shell in `README.md` (requirements, install, uninstall, manual setup sections) and in this document.

### `make install-shell` idempotency

`install-shell` checks for the string `"listicles shell integration"` in each rc file before appending. Running `make install` multiple times will not duplicate the lines. The source line points to the absolute path of the shell file in the project directory (`$(CURDIR)/shell/listicles.bash`), so moving the project directory breaks the integration — re-run `make install` from the new location to fix it.

---

## 8. Configuration Reference

**File:** `~/.config/delbysoft/listicles.toml`  
**Created automatically** on first run with all defaults and comments.  
**Format:** TOML — unknown fields are silently ignored (backward compatible).

### Complete default config

```toml
# listicles configuration file
# Key values: use names like "up", "down", "left", "right", "enter",
# "pgup", "pgdown", "home", "end", or single characters like "q", "j", "k".
# To use hjkl navigation: set up="k" down="j" left="h" right="l"

[keybinds]
up            = "up"
down          = "down"
left          = "left"
right         = "right"
confirm       = "enter"
parent        = "0"
page_up       = "pgup"
page_down     = "pgdown"
jump_top      = "home"
jump_bottom   = "end"
options       = "o"
add           = "a"
delete        = "d"
toggle_list   = "f"
rename        = "r"
edit          = "e"
yank          = "y"
cut           = "x"
paste         = "p"
copy_path     = "Y"
quit          = "q"
details       = "i"
toggle_hidden = "."
search        = "/"
swithc_tab    = " "

[display]
show_hidden        = false
default_list_mode  = "dirs"
search_max_results = 20
parent_depth       = 1

[apps]
editor = ""
opener = ""
```

### Field reference

#### `[keybinds]`

All values are BubbleTea key name strings. Valid values:

| Category | Values |
|----------|--------|
| Arrow keys | `"up"`, `"down"`, `"left"`, `"right"` |
| Named keys | `"enter"`, `"esc"`, `"space"`, `"tab"`, `"backspace"` |
| Page/jump keys | `"pgup"`, `"pgdown"`, `"home"`, `"end"` |
| Function keys | `"f1"` through `"f20"` |
| Modifier combos | `"ctrl+a"` through `"ctrl+z"`, `"alt+a"`, etc. |
| Single characters | Any printable ASCII character: `"q"`, `"/"`, `"."`, `"Y"`, etc. |

#### `[display]`

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `show_hidden` | bool | `false` | Show dotfiles. Can be toggled live with `toggle_hidden` |
| `default_list_mode` | string | `"dirs"` | `"dirs"` shows only directories; `"dirs_and_files"` shows both |
| `search_max_results` | int | `20` | Maximum entries shown in the live search filter. Minimum 1 (enforced by `Load()`). Does not cap subprocess search results |
| `parent_depth` | int | `1` | Number of greyed ancestor directory lines shown above the tree. `0` disables. Minimum 0 (negative values clamped to 0) |

#### `[apps]`

| Key | Default | Fallback chain |
|-----|---------|----------------|
| `editor` | `""` | `$EDITOR` env var → `$VISUAL` env var → `nano` → `vi` → `vim` |
| `opener` | `""` | `xdg-open` (Linux) → `open` (macOS) |

### Vim-style keybind config

To use `hjkl` navigation and common vim-style bindings:

```toml
[keybinds]
up          = "k"
down        = "j"
left        = "h"
right       = "l"
confirm     = "enter"
parent      = "0"
page_up     = "ctrl+u"
page_down   = "ctrl+d"
jump_top    = "home"
jump_bottom = "G"
# ... rest unchanged
```

Note: with this config, `l` navigates right and `toggle_list` must be bound to a different key (default `"f"` still works fine since `f` is unused in vim navigation).

---

## 9. CLI Flags

The binary accepts two flags:

| Flag | Type | Description |
|------|------|-------------|
| `--cd-file <path>` | string | Path of a temp file. On exit, the binary writes the chosen directory path to this file. If empty, no file is written and the app has no side effects on exit. Used by shell wrappers. |
| `--dir <path>` | string | Starting directory. Defaults to `$PWD` (`os.Getwd()`). Falls back to `"/"` if `os.Getwd()` fails. |

Both flags are optional. Running `listicles` directly (without a shell wrapper) opens the navigator in the current directory; pressing Enter exits without changing anything since no `--cd-file` is provided.

---

## 10. Building

### Standard build

```bash
make build
# → go build -ldflags="-s -w" -o bin/listicles .
# → bin/listicles (~4 MB)
```

`-ldflags="-s -w"` strips the symbol table and DWARF debug info, reducing binary size by roughly 30%.

### Manual build

```bash
go build -o bin/listicles .
```

### Cross-compilation

Go's cross-compilation requires no additional tools:

```bash
# macOS (Apple Silicon)
GOOS=darwin  GOARCH=arm64 go build -o bin/listicles-macos-arm64 .

# macOS (Intel)
GOOS=darwin  GOARCH=amd64 go build -o bin/listicles-macos-amd64 .

# Linux (amd64)
GOOS=linux   GOARCH=amd64 go build -o bin/listicles-linux-amd64 .

# Windows
GOOS=windows GOARCH=amd64 go build -o bin/listicles.exe .
```

Note: clipboard support on Linux requires `xclip`, `xsel`, or `wl-clipboard` at runtime (provided by `atotto/clipboard`). On Windows and macOS the OS clipboard API is used directly.

### Binary characteristics

- **Statically linked** — no shared library dependencies
- **No runtime** — Go compiles to native machine code; no VM or interpreter
- **Size** — approximately 3.5–4 MB (stripped)
- **Startup time** — under 50ms on any modern machine

---

## 11. Running the Tests

### Prerequisites

- Go 1.21+ (the project uses Go 1.26.1)
- A working terminal with a PTY (for integration tests)
- No other dependencies are required — `go test` downloads test dependencies automatically

### Unit tests

```bash
make test
# equivalent to:
go test ./internal/... -timeout 30s
```

Unit tests are in `package app`, `package config`, `package fs`, and `package search`. They use only `t.TempDir()` for filesystem isolation and have no external dependencies. They run in approximately 30ms on a modern machine.

**Expected output:**

```
ok  	github.com/wingitman/listicles/internal/app     0.013s
ok  	github.com/wingitman/listicles/internal/config  0.003s
ok  	github.com/wingitman/listicles/internal/fs      0.004s
ok  	github.com/wingitman/listicles/internal/search  0.007s
?   	github.com/wingitman/listicles/internal/ui      [no test files]
```

### Integration tests

```bash
make test-integration
# equivalent to:
go test -tags integration -timeout 60s -v .
```

Integration tests use `charmbracelet/x/exp/teatest` to run the real BubbleTea program inside a PTY. They require a working pseudo-terminal and typically run in 2–3 seconds total.

The `//go:build integration` tag at the top of `integration_test.go` means these tests are **excluded** from `go test ./...` by default — they only run when the tag is explicitly passed.

**Expected output (abbreviated):**

```
=== RUN   TestIntegration_QuitWithQ
--- PASS: TestIntegration_QuitWithQ (0.05s)
=== RUN   TestIntegration_InitialRender_ShowsEntries
--- PASS: TestIntegration_InitialRender_ShowsEntries (0.05s)
...
--- PASS: TestIntegration_SearchFlags_RT (0.15s)
PASS
ok  	github.com/wingitman/listicles	1.874s
```

### Run everything

```bash
make test-all
# runs: make test && make test-integration
```

### Useful test invocations

```bash
# Run a single test by name
go test ./internal/app/... -run TestExpandNode_LoadsChildren -v

# Run all tests matching a pattern
go test ./internal/... -run TestNavigate -v

# Run with race detector
go test ./internal/... -race

# Bypass test cache (force re-run)
go test ./internal/... -count=1

# Generate coverage report
go test ./internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out        # opens browser
go tool cover -func=coverage.out        # prints per-function coverage

# Verbose output for all unit tests
go test ./internal/... -v

# Run integration tests with race detector
go test -tags integration -race -timeout 60s -v .
```

### Clipboard tests

`TestUpdate_CopyPath` (and any test involving `clipboard.WriteAll`) will automatically skip on headless machines where no clipboard daemon is available. The test detects this at runtime — no configuration is needed. On a desktop system with `xclip`, `xsel`, or `wl-clipboard` installed, the test runs normally.

### Test caching

Go caches test results. If you modify source files, the cache is invalidated automatically. To force a re-run without modifying code, use `-count=1`:

```bash
go test ./internal/... -count=1
```

---

## 12. Test Architecture

### Two-layer strategy

| Layer | Location | Build tag | Speed | Requires PTY |
|-------|----------|-----------|-------|--------------|
| Unit | `internal/*/..._test.go` | (none) | ~30ms | No |
| Integration | `integration_test.go` | `integration` | ~2s | Yes |

Unit tests are the primary regression safety net. They test pure functions and model mutations directly — no terminal, no subprocess, no BubbleTea program. Integration tests verify that the whole system assembles correctly and that key presses produce the expected terminal output.

### White-box unit tests (`package app`)

`model_test.go` and `view_test.go` both declare `package app` (not `package app_test`). This gives them direct access to all unexported fields (`m.cursor`, `m.nodes`, `m.mode`, `m.clipboardPath`, etc.) and unexported functions (`expandNode`, `collapseNode`, `parseSearchFlags`, `calcPageJump`, etc.).

The trade-off: these tests are tightly coupled to internal implementation details and will break if internal types are renamed. For a tool of this size and single-maintainer nature, this is the correct trade-off — the tests cover the most complex logic directly without requiring a public API.

### Filesystem isolation

Every test that touches the filesystem calls `t.TempDir()` to get a unique temporary directory that is automatically removed when the test ends. No test modifies files outside its `TempDir`. Tests are safe to run in parallel (`go test -parallel N`), though this is not currently enabled.

### `teatest` integration pattern

Each integration test follows the same structure:

```go
func TestIntegration_Something(t *testing.T) {
    dir := newTestDir(t, "entry1", "entry2")   // temp dir with subdirs
    tm := newTestModel(t, dir)                  // starts real tea.Program in PTY

    // Wait for the initial render to appear in the output stream
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("entry1"))
    }, teatest.WithDuration(3*time.Second))

    // Send a key event
    tm.Send(tea.KeyMsg{Type: tea.KeyRight})

    // Assert on the resulting output
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("expected text"))
    }, teatest.WithDuration(3*time.Second))

    // Clean up
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
    tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
```

`tm.Output()` returns an `io.Reader` that streams the accumulated terminal output. `teatest.WaitFor` polls this reader in a loop (default every 50ms) until the condition is met or the timeout expires. Because the output is streamed and accumulated, a single `WaitFor` that checks for multiple strings in the condition function is more reliable than multiple sequential `WaitFor` calls.

### `t.TempDir()` vs `os.TempDir()`

All test helpers use `t.TempDir()`. This ensures:
- The temp directory is unique per test (no cross-test interference)
- The directory is removed when the test ends, even on failure
- No manual cleanup is needed

---

## 13. Adding New Features — Conventions

### Adding a new keybind

1. **`internal/config/config.go`** — add the new field to `Keybinds`:
   ```go
   MyAction string `toml:"my_action"`
   ```
   Add the default value to `Default()`:
   ```go
   MyAction: "m",
   ```
   Add the line to `WriteDefault()`:
   ```toml
   my_action = "m"
   ```

2. **`internal/app/model.go`** — add to `resolvedKeys`:
   ```go
   myAction string
   ```
   Map it in `resolveKeys()`:
   ```go
   myAction: k.MyAction,
   ```
   Add the handler in `Update()` under the `ModeNormal` block:
   ```go
   if matchKey(key, m.keys.myAction) {
       // ... do something
       return m, nil
   }
   ```

3. **`internal/app/view.go`** — add the hint to `renderStatusBar()`:
   ```go
   k.myAction + " my action",
   ```

4. **Tests** — add a test to `model_test.go`:
   ```go
   func TestUpdate_MyAction(t *testing.T) { ... }
   ```

### Adding a new confirm action

1. Add the constant to the `ConfirmAction` iota in `model.go`
2. Set `m.confirmAction = ConfirmMyAction` and `m.mode = ModeConfirm` in the trigger handler
3. Add a `case ConfirmMyAction:` branch to `executeConfirmedAction()`
4. If you need to pass extra data through the confirm flow, add a field to `Model` alongside the existing `pendingPath` and `pendingDestDir`

### Adding a new tea message type

```go
// At the bottom of model.go:
type myMsg struct {
    data string
}

// In Update(), add a case before tea.KeyMsg:
case myMsg:
    // handle it
    return m, nil

// To send it:
return m, func() tea.Msg { return myMsg{data: "..."} }
// Or via tea.Tick for delayed messages:
return m, tea.Tick(500*time.Millisecond, func(_ time.Time) tea.Msg {
    return myMsg{data: "..."}
})
```

### Adding a new display mode (beyond `DetailLevel`)

Follow the pattern of `ListMode`:
1. Define a new `type MyMode int` with constants
2. Add a field to `Model`
3. Add a keybind for toggling it
4. Use it in `renderNode` or `renderNodes` to change rendering behaviour
5. Add a badge to `renderHeader` if the mode is visually active
6. Add a hint to `renderStatusBar`

### Modifying the tree structure

Any function that reads `m.nodes[i]` and stores a **pointer** to it must be aware that `expandNode` replaces the backing array. Always capture necessary values (depth, expanded state, path) as plain values before calling `expandNode`:

```go
// Correct:
wasExpanded := node.Expanded    // bool copy
parentDepth := node.Depth       // int copy
err := m.expandNode(m.cursor)   // may replace m.nodes backing array
// node pointer is now stale — use parentDepth and wasExpanded instead

// Wrong:
err := m.expandNode(m.cursor)
if !node.Expanded { ... }       // node pointer is stale
```
