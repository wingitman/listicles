package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Keybinds holds all configurable key mappings.
// Key values use BubbleTea key names: "up", "down", "left", "right", "enter",
// "pgup", "pgdown", "home", "end", "ctrl+u", or single characters like "q".
type Keybinds struct {
	Up               string `toml:"up"`
	Down             string `toml:"down"`
	Left             string `toml:"left"`
	Right            string `toml:"right"`
	Confirm          string `toml:"confirm"`
	Parent           string `toml:"parent"`
	PageUp           string `toml:"page_up"`
	PageDown         string `toml:"page_down"`
	JumpTop          string `toml:"jump_top"`
	JumpBottom       string `toml:"jump_bottom"`
	Options          string `toml:"options"`
	Add              string `toml:"add"`
	Delete           string `toml:"delete"`
	ToggleList       string `toml:"toggle_list"`
	Rename           string `toml:"rename"`
	Edit             string `toml:"edit"`
	Yank             string `toml:"yank"`
	Cut              string `toml:"cut"`
	Paste            string `toml:"paste"`
	CopyPath         string `toml:"copy_path"`
	Quit             string `toml:"quit"`
	Details          string `toml:"details"`
	ToggleHidden     string `toml:"toggle_hidden"`
	Search           string `toml:"search"`
	SwitchTabs       string `toml:"switch_tabs"`
	SwitchTabsGlobal string `toml:"switch_tabs_global"`
	Ignore           string `toml:"ignore"`
	FullSearch       string `toml:"full_search"`
	CDDir            string `toml:"cd_dir"`
	OpenExplorer     string `toml:"open_explorer"`
	Bookmark         string `toml:"bookmark"`
	ShowHints        string `toml:"show_hints"`
}

// Display holds display preferences.
type Display struct {
	ShowHidden       bool   `toml:"show_hidden"`
	DefaultListMode  string `toml:"default_list_mode"` // "dirs" | "dirs_and_files"
	SearchMaxResults int    `toml:"search_max_results"`
	ParentDepth      int    `toml:"parent_depth"`
}

// Apps holds default application overrides.
type Apps struct {
	Editor string `toml:"editor"`
	Opener string `toml:"opener"`
}

// Config is the root config struct.
type Config struct {
	Keybinds Keybinds `toml:"keybinds"`
	Display  Display  `toml:"display"`
	Apps     Apps     `toml:"apps"`
}

// keybindEntries is the single authoritative list of every keybind TOML key
// name paired with its comment. Order here controls the order written to the
// config file. When adding a new keybind, add it here — migration and
// default-filling are derived automatically from this list and from Default().
var keybindEntries = []struct{ key, comment string }{
	{"up", "move cursor up"},
	{"down", "move cursor down"},
	{"left", "collapse / go to parent"},
	{"right", "expand directory"},
	{"confirm", "expand dir / open file"},
	{"parent", "go to parent directory"},
	{"page_up", "page up"},
	{"page_down", "page down"},
	{"jump_top", "jump to first item"},
	{"jump_bottom", "jump to last item"},
	{"quit", "quit"},
	{"options", "open config file in $EDITOR"},
	{"add", "add file/directory"},
	{"delete", "delete (with confirmation)"},
	{"toggle_list", "toggle dirs-only / dirs+files"},
	{"rename", "rename entry"},
	{"edit", "open in $EDITOR"},
	{"yank", "copy (filesystem yank)"},
	{"cut", "cut"},
	{"paste", "paste into current dir"},
	{"copy_path", "copy absolute path to clipboard"},
	{"details", "cycle detail level"},
	{"toggle_hidden", "toggle hidden files"},
	{"search", "open live search bar"},
	{"switch_tabs", "switch tabs (Recents / Bookmarks)"},
	{"switch_tabs_global", "toggle global scope in Recents/Bookmarks"},
	{"ignore", "add to .gitignore (git repos only)"},
	{"full_search", "re-run full fd/rg search"},
	{"cd_dir", "cd into selected dir and exit"},
	{"open_explorer", "open in system file explorer"},
	{"bookmark", "bookmark current selection"},
	{"show_hints", "cycle hint display mode"},
}

// displayEntries is the authoritative list of every display TOML key, used for
// migration checks. Values are written from the Config struct directly.
var displayEntries = []string{
	"show_hidden",
	"default_list_mode",
	"search_max_results",
	"parent_depth",
}

// appEntries is the authoritative list of every [apps] TOML key.
var appEntries = []string{
	"editor",
	"opener",
}

// Default returns a Config with all default values.
func Default() *Config {
	return &Config{
		Keybinds: Keybinds{
			Up:               "up",
			Down:             "down",
			Left:             "left",
			Right:            "right",
			Confirm:          "enter",
			Parent:           "0",
			PageUp:           "pgup",
			PageDown:         "pgdown",
			JumpTop:          "home",
			JumpBottom:       "end",
			Options:          "o",
			Add:              "a",
			Delete:           "d",
			ToggleList:       "f",
			Rename:           "r",
			Edit:             "e",
			Yank:             "y",
			Cut:              "x",
			Paste:            "p",
			CopyPath:         "Y",
			Quit:             "q",
			Details:          "i",
			ToggleHidden:     ".",
			Search:           "/",
			SwitchTabs:       "\t",
			SwitchTabsGlobal: "g",
			Ignore:           "I",
			FullSearch:       "ctrl+f",
			CDDir:            "c",
			OpenExplorer:     "E",
			Bookmark:         "b",
			ShowHints:        "?",
		},
		Display: Display{
			ShowHidden:       false,
			DefaultListMode:  "dirs_and_files",
			SearchMaxResults: 20,
			ParentDepth:      1,
		},
		Apps: Apps{
			Editor: "",
			Opener: "",
		},
	}
}

// ConfigDir returns the platform-appropriate path to the listicles config directory.
//
// os.UserConfigDir() returns the right base per OS:
//   - Windows: %APPDATA%  (e.g. C:\Users\wing\AppData\Roaming)
//   - macOS:   ~/Library/Application Support
//   - Linux:   ~/.config  (XDG_CONFIG_HOME if set, else ~/.config)
func ConfigDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return ""
		}
		return filepath.Join(home, ".config", "listicles")
	}
	return filepath.Join(base, "listicles")
}

// ConfigPath returns the full path to the config file.
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "listicles.toml")
}

// resolvePath returns the config file path to use. It checks the primary
// (OS-native) path first, then falls back to the legacy path
// (~/.config/delbysoft/listicles.toml) for users who already have a config
// there from an earlier install. Returns the primary path when neither exists
// so that WriteDefault writes to the correct location.
func resolvePath() string {
	primary := ConfigPath()
	if _, err := os.Stat(primary); err == nil {
		return primary
	}

	// Legacy path: ~/.config/delbysoft/listicles.toml
	if home, err := os.UserHomeDir(); err == nil {
		legacy := filepath.Join(home, ".config", "delbysoft", "listicles.toml")
		if legacy != primary {
			if _, err := os.Stat(legacy); err == nil {
				return legacy
			}
		}
	}

	return primary
}

// Load reads the config file, creating it with defaults if it doesn't exist.
func Load() (*Config, error) {
	cfg := Default()
	path := resolvePath()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(ConfigDir(), 0755); err != nil {
			return cfg, nil
		}
		if err := WriteDefault(path); err != nil {
			return cfg, nil
		}
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return Default(), err
	}

	// Fill any keybind fields absent in the file with their defaults.
	// Primary safety net: even if migration fails, the app runs correctly.
	applyKeybindDefaults(cfg)

	// Clamp display values.
	if cfg.Display.SearchMaxResults < 1 {
		cfg.Display.SearchMaxResults = 1
	}
	if cfg.Display.ParentDepth < 0 {
		cfg.Display.ParentDepth = 0
	}
	if cfg.Display.DefaultListMode == "" {
		cfg.Display.DefaultListMode = "dirs_and_files"
	}

	// Migration: rewrite file if any known key is missing, preserving user values.
	if needsMigration(path) {
		_ = writeMigrated(path, cfg) // non-fatal
	}

	return cfg, nil
}

// applyKeybindDefaults fills every empty keybind field with its default value.
// TOML leaves fields absent from the file as zero-value (""), so this ensures
// the app always has a working binding regardless of what the file contains.
func applyKeybindDefaults(cfg *Config) {
	d := Default().Keybinds
	if cfg.Keybinds.Up == "" {
		cfg.Keybinds.Up = d.Up
	}
	if cfg.Keybinds.Down == "" {
		cfg.Keybinds.Down = d.Down
	}
	if cfg.Keybinds.Left == "" {
		cfg.Keybinds.Left = d.Left
	}
	if cfg.Keybinds.Right == "" {
		cfg.Keybinds.Right = d.Right
	}
	if cfg.Keybinds.Confirm == "" {
		cfg.Keybinds.Confirm = d.Confirm
	}
	if cfg.Keybinds.Parent == "" {
		cfg.Keybinds.Parent = d.Parent
	}
	if cfg.Keybinds.PageUp == "" {
		cfg.Keybinds.PageUp = d.PageUp
	}
	if cfg.Keybinds.PageDown == "" {
		cfg.Keybinds.PageDown = d.PageDown
	}
	if cfg.Keybinds.JumpTop == "" {
		cfg.Keybinds.JumpTop = d.JumpTop
	}
	if cfg.Keybinds.JumpBottom == "" {
		cfg.Keybinds.JumpBottom = d.JumpBottom
	}
	if cfg.Keybinds.Options == "" {
		cfg.Keybinds.Options = d.Options
	}
	if cfg.Keybinds.Add == "" {
		cfg.Keybinds.Add = d.Add
	}
	if cfg.Keybinds.Delete == "" {
		cfg.Keybinds.Delete = d.Delete
	}
	if cfg.Keybinds.ToggleList == "" {
		cfg.Keybinds.ToggleList = d.ToggleList
	}
	if cfg.Keybinds.Rename == "" {
		cfg.Keybinds.Rename = d.Rename
	}
	if cfg.Keybinds.Edit == "" {
		cfg.Keybinds.Edit = d.Edit
	}
	if cfg.Keybinds.Yank == "" {
		cfg.Keybinds.Yank = d.Yank
	}
	if cfg.Keybinds.Cut == "" {
		cfg.Keybinds.Cut = d.Cut
	}
	if cfg.Keybinds.Paste == "" {
		cfg.Keybinds.Paste = d.Paste
	}
	if cfg.Keybinds.CopyPath == "" {
		cfg.Keybinds.CopyPath = d.CopyPath
	}
	if cfg.Keybinds.Quit == "" {
		cfg.Keybinds.Quit = d.Quit
	}
	if cfg.Keybinds.Details == "" {
		cfg.Keybinds.Details = d.Details
	}
	if cfg.Keybinds.ToggleHidden == "" {
		cfg.Keybinds.ToggleHidden = d.ToggleHidden
	}
	if cfg.Keybinds.Search == "" {
		cfg.Keybinds.Search = d.Search
	}
	if cfg.Keybinds.SwitchTabs == "" {
		cfg.Keybinds.SwitchTabs = d.SwitchTabs
	}
	if cfg.Keybinds.SwitchTabsGlobal == "" {
		cfg.Keybinds.SwitchTabsGlobal = d.SwitchTabsGlobal
	}
	if cfg.Keybinds.Ignore == "" {
		cfg.Keybinds.Ignore = d.Ignore
	}
	if cfg.Keybinds.FullSearch == "" {
		cfg.Keybinds.FullSearch = d.FullSearch
	}
	if cfg.Keybinds.CDDir == "" {
		cfg.Keybinds.CDDir = d.CDDir
	}
	if cfg.Keybinds.OpenExplorer == "" {
		cfg.Keybinds.OpenExplorer = d.OpenExplorer
	}
	if cfg.Keybinds.Bookmark == "" {
		cfg.Keybinds.Bookmark = d.Bookmark
	}
	if cfg.Keybinds.ShowHints == "" {
		cfg.Keybinds.ShowHints = d.ShowHints
	}
}

// needsMigration returns true if the config file is missing any known keybind
// or display key. Derived from keybindEntries / displayEntries / appEntries so
// it stays in sync automatically when new keys are added to those lists.
func needsMigration(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	content := string(data)
	for _, e := range keybindEntries {
		if !fileContainsKey(content, e.key) {
			return true
		}
	}
	for _, key := range displayEntries {
		if !fileContainsKey(content, key) {
			return true
		}
	}
	for _, key := range appEntries {
		if !fileContainsKey(content, key) {
			return true
		}
	}
	return false
}

// fileContainsKey returns true when the TOML file content contains a line
// where the given key appears as an actual assignment (key = ...), not just
// as part of a comment or a value string. Prevents false positives from keys
// whose names appear inside other values or comments.
func fileContainsKey(content, key string) bool {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, key+"=") || strings.HasPrefix(trimmed, key+" ") {
			return true
		}
	}
	return false
}

// writeMigrated rewrites the config file with all current keys and comments,
// preserving the user's existing values (already loaded into cfg).
func writeMigrated(path string, cfg *Config) error {
	return os.WriteFile(path, []byte(buildTOML(cfg)), 0644)
}

// WriteDefault writes the default config file to path.
func WriteDefault(path string) error {
	return os.WriteFile(path, []byte(buildTOML(Default())), 0644)
}

// buildTOML generates the full config file content from a Config, using
// keybindEntries as the authoritative source so the output stays in sync
// automatically when new keybinds are added.
func buildTOML(cfg *Config) string {
	vals := keybindValues(&cfg.Keybinds)
	d := cfg.Display
	a := cfg.Apps

	// Find longest key name for column alignment.
	maxLen := 0
	for _, e := range keybindEntries {
		if len(e.key) > maxLen {
			maxLen = len(e.key)
		}
	}

	out := "# listicles configuration file\n" +
		"# Key values: use names like \"up\", \"down\", \"left\", \"right\", \"enter\",\n" +
		"# \"pgup\", \"pgdown\", \"home\", \"end\", or single characters like \"q\", \"j\", \"k\".\n" +
		"# To use hjkl navigation: set up=\"k\" down=\"j\" left=\"h\" right=\"l\"\n\n" +
		"[keybinds]\n"

	for _, e := range keybindEntries {
		val := vals[e.key]
		pad := strings.Repeat(" ", maxLen-len(e.key))
		out += e.key + pad + " = " + quote(val) + "  # " + e.comment + "\n"
	}

	out += "\n[display]\n" +
		"show_hidden       = " + boolStr(d.ShowHidden) + "\n" +
		"default_list_mode = " + quote(d.DefaultListMode) + "   # \"dirs\" or \"dirs_and_files\"\n\n" +
		"# Max results shown during live / subprocess search (min 1)\n" +
		"search_max_results = " + itoa(d.SearchMaxResults) + "\n\n" +
		"# Greyed-out ancestor directories shown above the tree.\n" +
		"# Set to 0 to disable. Default 1 shows the immediate parent.\n" +
		"parent_depth = " + itoa(d.ParentDepth) + "\n\n" +
		"[apps]\n" +
		"editor = " + quote(a.Editor) + "   # leave empty to use $EDITOR env var\n" +
		"opener = " + quote(a.Opener) + "   # leave empty to use xdg-open (Linux) / open (macOS)\n"

	return out
}

// keybindValues maps every TOML key name to the current value in k.
// This is the one place that couples struct fields to their TOML names for writing.
func keybindValues(k *Keybinds) map[string]string {
	return map[string]string{
		"up":                k.Up,
		"down":              k.Down,
		"left":              k.Left,
		"right":             k.Right,
		"confirm":           k.Confirm,
		"parent":            k.Parent,
		"page_up":           k.PageUp,
		"page_down":         k.PageDown,
		"jump_top":          k.JumpTop,
		"jump_bottom":       k.JumpBottom,
		"quit":              k.Quit,
		"options":           k.Options,
		"add":               k.Add,
		"delete":            k.Delete,
		"toggle_list":       k.ToggleList,
		"rename":            k.Rename,
		"edit":              k.Edit,
		"yank":              k.Yank,
		"cut":               k.Cut,
		"paste":             k.Paste,
		"copy_path":         k.CopyPath,
		"details":           k.Details,
		"toggle_hidden":     k.ToggleHidden,
		"search":            k.Search,
		"switch_tabs":       k.SwitchTabs,
		"switch_tabs_global": k.SwitchTabsGlobal,
		"ignore":            k.Ignore,
		"full_search":       k.FullSearch,
		"cd_dir":            k.CDDir,
		"open_explorer":     k.OpenExplorer,
		"bookmark":          k.Bookmark,
		"show_hints":        k.ShowHints,
	}
}

func quote(s string) string   { return `"` + s + `"` }
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		return "-" + string(buf[pos:])
	}
	return string(buf[pos:])
}
