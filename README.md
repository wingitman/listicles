# listicles

An interactive terminal file explorer. Type `listicles` (or `l`) to open it, navigate with the keyboard, and press Enter to expand directories or edit files.

Built with [BubbleTea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss).

> Made by [delbysoft](https://github.com/wingitman)

---

## Requirements

- A terminal with colour support
- `bash`, `zsh`, `fish`, or `powershell (pwsh)`

**Optional plugins:** [`fd`](https://github.com/sharkdp/fd) for faster name search, [`rg`](https://github.com/BurntSushi/ripgrep) for content search, and [`zoxide`](https://github.com/ajeetdsouza/zoxide) for known-directory search.

---

## Install

### Windows

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

Requires `make`.

```bash
git clone https://github.com/wingitman/listicles.git
cd listicles
make install
```

This builds the binary, copies it to `~/.local/bin/listicles`, and patches your shell rc file (`~/.bashrc`, `~/.zshrc`, `~/.config/fish/config.fish`, or `~/.config/powershell/Microsoft.PowerShell_profile.ps1`).
Then reload your shell and type `l` or `listicles`.

---

## Neovim
Use [listicles.nvim](https://github.com/wingitman/listicles.nvim) in Neovim!

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
rm -rf ~/.config/delbysoft  # removes config (optional)
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
| `Enter` | Expand directory, or edit file |

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
| `E` | Open in system file explorer |

### Display & search

| Key | Action |
|-----|--------|
| `f` | Toggle directories-only / directories+files |
| `.` | Toggle hidden files |
| `i` | Cycle detail: none → count → size → full path |
| `/` | Search (live filter as you type) |
| `Enter` (in search) | Run full search via `fd`/`rg`, or navigate live `zoxide` results |
| `Esc` (in search) | Cancel |
| `U` | Show updates, recent changes, and install history commits |
| `P` | Show plugins and toggle optional integrations |
| `T` | Open the theme picker |
| `o` | Open config in `$EDITOR` |
| `q` / `Esc` | Quit |

**Search flags** (add to your query): `-r` recursive, `-t` search file contents (or `-rt` for both), `-z` live-search zoxide's known directories. When `-z` is present, `-r` and `-t` are ignored.
Example: `main -rt` finds files containing the word `main`, recursively.
Example: `.conf -r` find files/directories containing '.conf', recursively. 
Example: `system32` find file/directories containing 'system32' in this directory
Example: `proj -z` searches zoxide's directory database as you type.

---

## Configuration

The config files are created automatically on first launch or install. Press `o` inside listicles to open `listicles.toml` in your editor.

| OS | Path |
|---|---|
| Windows | `%APPDATA%\delbysoft\listicles.toml` (e.g. `C:\Users\you\AppData\Roaming\delbysoft\listicles.toml`) |
| macOS | `~/Library/Application Support/delbysoft/listicles.toml` |
| Linux | `~/.config/delbysoft/listicles.toml` |

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
show_updates  = "U"
plugins       = "P"
theme         = "T"       # open the theme picker

[display]
show_hidden        = false
default_list_mode  = "dirs_and_files"   # "dirs" or "dirs_and_files"
search_max_results = 20
parent_depth       = 1        # 0 = off, 1 = show immediate parent, 2+ = more ancestors

[apps]
editor = ""   # leave empty to use $EDITOR env var
opener = ""   # leave empty to use xdg-open (Linux) / open (macOS)

[updates]
disable_checks = false  # true disables startup update checks
current_commit = ""     # installed app commit, maintained by listicles
repo_path = ""          # source checkout used for updates
terminal = ""           # optional terminal command for detached updates

[plugins]
fd = true       # use fd for full name search when installed
rg = true       # use rg for full content search when installed
zoxide = true   # use zoxide for -z directory search when installed

[themes]
theme_name = "terminal"   # terminal (inherit terminal theme), or a named theme from the reference file
theme_file = "~/.config/delbysoft/themes.toml"   # shared theme file for Delbysoft apps

# Optional overrides applied after the selected theme:
# selected_background = "#ffffff"
# selected_foreground = "#000000"
```

### Shared themes

The shared theme file is created automatically at the platform-specific Delbysoft
configuration path (`~/.config/delbysoft/themes.toml` on Linux). It can be shared
by other Delbysoft terminal applications:

```toml
[themes.ocean]
foreground = "#D7E3FF"
background = "#101522"
primary = "#7C9EF0"
accent = "#F0A47C"
muted = "#66708F"
file = "#B0B0CC"
border = "#35415F"
selected_background = "#3568B8"
selected_foreground = "#FFFFFF"
header_background = "#17213A"
selector = "#FFFFFF"

[themes.high_contrast]
foreground = "#FFFFFF"
background = "#000000"
primary = "#00FFFF"
accent = "#FFFF00"
muted = "#C0C0C0"
file = "#FFFFFF"
border = "#FFFFFF"
selected_background = "#FFFF00"
selected_foreground = "#000000"
header_background = "#000000"
selector = "#FFFF00"
```

Set `theme_name` to one of these names to use it. Set it to `terminal` to inherit
the terminal's default colors. Per-app overrides take precedence over the shared
theme. If the shared file or selected theme is unavailable, listicles uses its
built-in fallback theme.

The generated file also includes `redteam`, `blueteam`, `vim`, `neovim`,
`monotone`, `cyberpunk`, and `sands`. Each theme supports the complete shared
palette: text, accents, errors, success, borders, selection, headers, crumbs,
clipboard, branding, selector, and image colors.

Installers and updates create the shared file when needed and validate the
required theme settings. Factory resets restore the `listicles.toml` selection
without overwriting the shared theme collection.

## Updates

On launch, listicles checks the configured source checkout with `git fetch --prune --all`. If commits exist between the installed commit and the current branch's upstream, it prompts before updating.

Updates run in a separate terminal and listicles exits before the installer starts. The update keeps your current branch/fork behavior: it uses your checkout's current branch and upstream, not a hardcoded release branch. Press `U` to review recent commits, expand commit descriptions, install the latest upstream commit, or install an older commit from history.

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
`l` is a shell function (not an alias or script) that passes a temp file path to the binary. When you select a directory and press `c`, listicles writes the path to that file. The function reads it and calls `cd`. This is the only way to change the parent shell's directory — a subprocess can't do it.

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

## Support
<a href='https://ko-fi.com/W7W21WP5L7' target='_blank'><img height='36' style='border:0px;height:36px;' src='https://storage.ko-fi.com/cdn/kofi4.png?v=6' border='0' alt='Buy Me a Coffee at ko-fi.com' /></a>

## License

MIT — see [LICENSE](LICENSE).

Copyright (c) 2026 [delbysoft](https://github.com/wingitman)
