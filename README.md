# listicle

An interactive terminal file explorer. Type `l` to launch it from any directory, navigate with keyboard shortcuts, and press Enter to `cd` into the selected directory.

Built with [BubbleTea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss).

> Made by [delbysoft](https://github.com/wingitman)

---

## Features

- **Inline tree navigation** — expand and collapse subdirectories in place; siblings stay visible
- **Live search filter** — press `/` and type to filter the current directory instantly; press Enter to escalate to a full recursive or content search via `fd`/`rg` (falls back to `find`/`grep`)
- **Multi-digit jump** — type `19` within 500ms to jump directly to item 19 at the current depth
- **Page jump** — logarithmically scaled page up/down that adapts to the number of visible items
- **Yank / cut / paste** — filesystem copy and move with confirmation
- **Copy path to clipboard** — copies the absolute path of the selected item to the system clipboard
- **Add / rename / delete** — create files and directories, rename, and delete with confirmation
- **Open in default app** — press Enter on a file to open it with `xdg-open`/`open`
- **Edit in `$EDITOR`** — open any file directly in your configured editor
- **Detail toggle** — cycle through file count, size, and full path displays
- **Greyed parent crumbs** — configurable number of ancestor directories shown above the tree
- **Shell `cd` integration** — pressing Enter on a directory actually changes your shell's working directory
- **Fully remappable keybinds** — every key is configurable via `~/.config/listicle/listicle.toml`
- **Hidden file toggle** — show or hide dotfiles on demand

---

## Requirements

- Go 1.21+ (to build from source)
- A modern terminal with colour support (kitty, alacritty, ghostty, iTerm2, Windows Terminal, etc.)
- `bash`, `zsh`, or `fish` shell

**Optional but recommended (for faster search):**
- [`fd`](https://github.com/sharkdp/fd) — fast filename search
- [`rg`](https://github.com/BurntSushi/ripgrep) — fast content search

---

## Installation

### 1. Clone and build

```bash
git clone https://github.com/wingitman/listicles.git
cd listicles
make build
```

This produces `bin/listicle` (a single ~4 MB binary with no runtime dependencies).

### 2. Install the binary and shell integration

```bash
make install
```

This copies `bin/listicle` to `~/.local/bin/listicle` and appends the shell wrapper function to your `~/.bashrc`, `~/.zshrc`, and/or `~/.config/fish/config.fish` depending on which exist.

Then reload your shell:

```bash
# bash / zsh
source ~/.bashrc   # or ~/.zshrc

# fish
source ~/.config/fish/config.fish
```

### 3. Use it

```bash
l
```

That's it. Navigate, press Enter on a directory, and your shell's working directory changes.

---

## Uninstall

### 1. Remove the binary

```bash
make uninstall
```

This removes `~/.local/bin/listicle`.

### 2. Remove the shell integration

Open your shell rc file and delete the two lines added by `make install`:

```bash
# bash
nano ~/.bashrc   # or vim, etc.

# zsh
nano ~/.zshrc

# fish
nano ~/.config/fish/config.fish
```

Find and delete this block (the comment and the `source` line below it):

```
# listicle shell integration
source /path/to/listicles/shell/listicle.bash
```

Then reload your shell:

```bash
source ~/.bashrc   # or ~/.zshrc / source ~/.config/fish/config.fish
```

### 3. Remove the config (optional)

```bash
rm -rf ~/.config/listicle
```

### 4. Remove the repo (optional)

```bash
cd ..
rm -rf listicles/
```

---

## Shell integration explained

The `l` function is a thin shell wrapper. It passes a temp file path to the binary via `--cd-file`. When you press Enter on a directory (or quit with `q`/`Esc`), listicle writes the chosen path to that file. The shell wrapper reads it and runs `cd`.

This is the same pattern used by tools like `ranger`, `nnn`, and `zoxide`.

**Manual setup** (if `make install` didn't cover your shell):

```bash
# bash / zsh — add to ~/.bashrc or ~/.zshrc
l() {
    local tmp=$(mktemp)
    listicle --cd-file "$tmp" "$@"
    local dir=$(cat "$tmp" 2>/dev/null)
    rm -f "$tmp"
    [ -n "$dir" ] && builtin cd "$dir"
}

# fish — add to ~/.config/fish/config.fish
function l
    set tmp (mktemp)
    listicle --cd-file $tmp $argv
    set dir (cat $tmp 2>/dev/null)
    rm -f $tmp
    test -n "$dir" -a "$dir" != (pwd); and builtin cd $dir
end
```

---

## Keybinds

All keybinds are configurable. These are the defaults.

### Navigation

| Key | Action |
|-----|--------|
| `↑` / `↓` | Move selection up / down |
| `←` / `→` | Collapse / expand directory |
| `Enter` | `cd` to selected directory, or open file in default app |
| `0` | Go to parent directory |
| `Home` | Jump to first item |
| `End` | Jump to last item |
| `PgUp` / `PgDn` | Page up / down (logarithmically scaled) |
| `1`–`9` (or multi-digit, e.g. `1` then `9`) | Jump to Nth item at current depth and expand it |

**Multi-digit jump:** type digits within 500ms of each other. The number being typed is shown in the header (`→ 19`). The jump always operates at the depth of the currently selected item.

**Page jump scaling:** the jump size is `round(log₂(N))` where N is the number of visible items. This means small directories jump fewer rows and large directories jump more.

### File operations

| Key | Action |
|-----|--------|
| `a` | Add a file or directory (end name with `/` for directory) |
| `d` | Delete selected item (confirmation required) |
| `r` | Rename selected item (confirmation required) |
| `y` | Yank (mark for copy). Press `y` again on the same item to clear. |
| `x` | Cut (mark for move). Press `x` again to clear. |
| `p` | Paste yanked/cut item into current directory (confirmation required) |
| `Y` | Copy absolute path of selected item to system clipboard |
| `e` | Open selected file in `$EDITOR` |

### Display

| Key | Action |
|-----|--------|
| `f` | Toggle between directories-only and directories+files |
| `.` | Toggle hidden files (dotfiles) |
| `i` | Cycle detail display: none → file/dir count → size → full path |

### Search

| Key | Action |
|-----|--------|
| `/` | Open search bar (filters current directory live as you type) |
| `Enter` (in search) | Run full search via `fd` / `rg` (supports flags below) |
| `Esc` (in search) | Cancel and restore previous listing |

**Search flags** (type anywhere in the query):

| Flag | Meaning |
|------|---------|
| `-r` | Recursive — search all subdirectories |
| `-t` | Text mode — search file contents instead of names |
| `-rt` or `-tr` | Both recursive and text mode |

Example: typing `main -rt` will search recursively for files containing the text `main`.

If `fd` is installed it is used for name searches; if `rg` is installed it is used for content searches. Both fall back to `find` and `grep` otherwise.

### Other

| Key | Action |
|-----|--------|
| `o` | Open `~/.config/listicle/listicle.toml` in `$EDITOR` |
| `q` / `Esc` | Quit without changing directory |

---

## Configuration

On first launch, `~/.config/listicle/listicle.toml` is created with all defaults. Edit it with `o` from inside listicle, or directly.

### Default config

```toml
# listicle configuration file
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

[display]
show_hidden        = false
default_list_mode  = "dirs_and_files"   # "dirs" or "dirs_and_files"
search_max_results = 20
parent_depth       = 1        # 0 = off, 1 = show immediate parent, 2+ = more ancestors

[apps]
editor = ""   # leave empty to use $EDITOR env var
opener = ""   # leave empty to use xdg-open (Linux) / open (macOS)
```

### Vim-style config

To use `hjkl` navigation and vim-style page/top/bottom binds, replace the `[keybinds]` section with:

```toml
[keybinds]
up            = "k"
down          = "j"
left          = "h"
right         = "l"
confirm       = "enter"
parent        = "0"
page_up       = "ctrl+u"
page_down     = "ctrl+d"
jump_top      = "g"
jump_bottom   = "G"
options       = "o"
add           = "a"
delete        = "d"
toggle_list   = "L"
rename        = "r"
edit          = "e"
yank          = "y"
cut           = "x"
paste         = "p"
copy_path     = "Y"
quit          = "q"
details       = "J"
toggle_hidden = "H"
search        = "/"
```

### `parent_depth` examples

```toml
parent_depth = 0   # no crumbs — clean minimal view
parent_depth = 1   # show immediate parent (default)
parent_depth = 3   # show three levels of ancestors
```

With `parent_depth = 1` and the cursor inside `listicle/internal/`:

```
  Work/              ← greyed ancestor
    listicle/        ← root label
 1 ▶ app/
 2 ▶ config/
 3 ▶ fs/
 4 ▶ search/
 5 ▶ ui/
```

### `default_list_mode`

```toml
default_list_mode = "dirs"            # show directories only (default)
default_list_mode = "dirs_and_files"  # show files too on startup
```

---

## Building from source

```bash
make build    # → bin/listicle
make install  # build + install to ~/.local/bin + patch shell rc files
make clean    # remove bin/
make uninstall
```

**Cross-compile:**

```bash
GOOS=darwin  GOARCH=arm64 go build -o bin/listicle-macos-arm64 .
GOOS=linux   GOARCH=amd64 go build -o bin/listicle-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o bin/listicle-windows.exe .
```

---

## License

MIT — see [LICENSE](LICENSE).

Copyright (c) 2026 [delbysoft](https://github.com/wingitman)
