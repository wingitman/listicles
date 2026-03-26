# listicles

An interactive terminal file explorer. Type `listicles` (or `l`) to open it, navigate with the keyboard, and press Enter to `cd` into the selected directory.

Built with [BubbleTea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss).

> Made by [delbysoft](https://github.com/wingitman)

---

## Requirements

- Go 1.21+ (to build)
- A terminal with colour support
- `bash`, `zsh`, `fish`, or `powershell (pwsh)`

**Optional (faster search):** [`fd`](https://github.com/sharkdp/fd) and [`rg`](https://github.com/BurntSushi/ripgrep)

---

## Install

### Windows

No `make` or Unix tools required — only [Go](https://go.dev/dl/).

```powershell
git clone https://github.com/wingitman/listicles.git
cd listicles
.\install.ps1
```

This builds the binary, installs it to `%LOCALAPPDATA%\Programs\listicles\`, adds that directory to your user PATH (registry, no admin required), and patches your PowerShell `$PROFILE` with the `l` function.

Open a new PowerShell terminal and type `l`.

> **Execution policy:** if you get a script-blocked error, run this once:
> ```powershell
> Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
> ```

### macOS / Linux

Requires `make` and Go.

```bash
git clone https://github.com/wingitman/listicles.git
cd listicles
make install
```

This builds the binary, copies it to `~/.local/bin/listicles`, and patches your shell rc file (`~/.bashrc`, `~/.zshrc`, `~/.config/fish/config.fish`, or `~/.config/powershell/Microsoft.PowerShell_profile.ps1`).
Then reload your shell and type `l`.

---

## Uninstall

### Windows

```powershell
.\uninstall.ps1
```

Removes the binary and install directory, and removes it from your user PATH. Then open your `$PROFILE` and delete the `# listicles shell integration` block if desired.

```powershell
notepad $PROFILE
```

### macOS / Linux

```bash
make uninstall          # removes the binary
rm ~/.config/listicles  # removes config (optional)
```

Also open your shell rc file and remove the two lines added by `make install`:

```
# listicles shell integration
source /path/to/listicles/shell/listicles.bash
```

---

## Keybinds

All keybinds are configurable. These are the defaults.

### Navigation

| Key | Action |
|-----|--------|
| `↑` / `↓` | Move up / down |
| `←` / `→` | Collapse / expand directory |
| `←` / `0` | Go to parent directory |
| `Home` / `End` | Jump to first / last item |
| `PgUp` / `PgDn` | Page up / down |
| `1`–`9` (multi-digit supported) | Jump to Nth item at current depth |
| `Enter` | `cd` to directory, or open file |

Multi-digit jump: type digits within 500ms. The number being typed shows in the header (`→ 19`).

### File operations

| Key | Action | Cofirmation |
|-----|--------|-------------|
| `a` | Add file or directory (end with `/` for directory) | |
| `d` | Delete | _Yes_ |
| `r` | Rename | _Yes_ |
| `y` | Yank (copy) file/directory |
| `Y` | Copy absolute path to clipboard |
| `x` | Cut |
| `p` | Paste into current directory | _Yes_|
| `e` | Open in `$EDITOR` |

### Display & search

| Key | Action |
|-----|--------|
| `f` | Toggle directories-only / directories+files |
| `.` | Toggle hidden files |
| `i` | Cycle detail: none → count → size → full path |
| `/` | Search (live filter as you type) |
| `Enter` (in search) | Run full search via `fd`/`rg` |
| `Esc` (in search) | Cancel |
| `o` | Open config in `$EDITOR` |
| `q` / `Esc` | Quit |

**Search flags** (add to your query): `-r` recursive, `-t` search file contents (or `-rt` for both).
Example: `main -rt` finds files containing the word `main`, recursively.
Example: `.conf -r` find files/directories containing '.conf', recursively. 
Example: `system32` find file/directories containing 'system32' in this directory

---

## Configuration

The config file is created automatically on first launch. Press `o` inside listicles to open it in your editor.

| OS | Path |
|---|---|
| Windows | `%APPDATA%\listicles\listicles.toml` (e.g. `C:\Users\you\AppData\Roaming\listicles\listicles.toml`) |
| macOS | `~/Library/Application Support/listicles/listicles.toml` |
| Linux | `~/.config/listicles/listicles.toml` |

### Default config

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

---

## Shell integration
`l` is a shell function (not an alias or script) that passes a temp file path to the binary. When you select a directory and press Enter, listicles writes the path to that file. The function reads it and calls `cd`. This is the only way to change the parent shell's directory — a subprocess can't do it.

Same pattern as `ranger`, `nnn`, and `zoxide`.

**Manual setup** (if `make install` didn't cover your shell):

```bash
# bash / zsh
l() {
    local tmp=$(mktemp)
    listicles --cd-file "$tmp" "$@"
    local dir=$(cat "$tmp" 2>/dev/null)
    rm -f "$tmp"
    [ -n "$dir" ] && builtin cd "$dir"
}
```

```fish
# fish
function l
    set tmp (mktemp)
    listicles --cd-file $tmp $argv
    set dir (cat $tmp 2>/dev/null)
    rm -f $tmp
    test -n "$dir" -a "$dir" != (pwd); and builtin cd $dir
end
```

```powershell
# PowerShell (pwsh)
function l {
    $tmp = [System.IO.Path]::GetTempFileName()
    listicles --cd-file $tmp @args
    $dir = Get-Content $tmp -ErrorAction SilentlyContinue
    Remove-Item $tmp -Force -ErrorAction SilentlyContinue
    if ($dir -and $dir -ne $PWD.Path) { Set-Location $dir }
}
```

**Adding support for another shell:** create a function that runs `listicles --cd-file <tmpfile>`, then reads the file and calls `cd` — that's the whole pattern. Contributions welcome.

---

## Building from source

**macOS / Linux**

```bash
make build    # → bin/listicles
make install  # build + install + patch shell rc
make clean
make uninstall
```

**Windows (PowerShell)**

```powershell
go build -ldflags='-s -w' -o bin\listicles.exe .   # build only
.\install.ps1                                        # build + install + patch profile
.\uninstall.ps1                                      # uninstall
Remove-Item -Recurse bin\                            # clean
```

Cross-compile:

```bash
GOOS=darwin  GOARCH=arm64 go build -o bin/listicles-macos-arm64 .
GOOS=linux   GOARCH=amd64 go build -o bin/listicles-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o bin/listicles-windows.exe .
```

---

## License

MIT — see [LICENSE](LICENSE).

Copyright (c) 2026 [delbysoft](https://github.com/wingitman)
