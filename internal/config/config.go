package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	ShowUpdates      string `toml:"show_updates"`
	Plugins          string `toml:"plugins"`
	PreviewToggle    string `toml:"preview_toggle"`
	PreviewMode      string `toml:"preview_mode"`
	Theme            string `toml:"theme"`
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

// Updates holds update-check and installer preferences.
type Updates struct {
	DisableChecks bool   `toml:"disable_checks"`
	CurrentCommit string `toml:"current_commit"`
	RepoPath      string `toml:"repo_path"`
	Terminal      string `toml:"terminal"`
}

// Plugins controls optional external command integrations.
type Plugins struct {
	Fd     bool `toml:"fd"`
	Rg     bool `toml:"rg"`
	Zoxide bool `toml:"zoxide"`
}

// Themes selects a shared theme and contains optional per-app overrides.
type Themes struct {
	ThemeName          string `toml:"theme_name"`
	ThemeFile          string `toml:"theme_file"`
	Foreground         string `toml:"foreground"`
	Background         string `toml:"background"`
	Primary            string `toml:"primary"`
	Accent             string `toml:"accent"`
	Muted              string `toml:"muted"`
	Error              string `toml:"error"`
	Success            string `toml:"success"`
	File               string `toml:"file"`
	Border             string `toml:"border"`
	SelectedBackground string `toml:"selected_background"`
	SelectedForeground string `toml:"selected_foreground"`
	HeaderBackground   string `toml:"header_background"`
	HintKey            string `toml:"hint_key"`
	ParentCrumb        string `toml:"parent_crumb"`
	RootDirectory      string `toml:"root_directory"`
	Clipboard          string `toml:"clipboard"`
	BrandPrimary       string `toml:"brand_primary"`
	BrandSecondary     string `toml:"brand_secondary"`
	Selector           string `toml:"selector"`
	ImageBackground    string `toml:"image_background"`
}

// Theme is the portable subset currently consumed by listicles.
type Theme struct {
	Themes
}

type themeFile struct {
	Themes map[string]Theme `toml:"themes"`
}

// ResolvedTheme describes the selection style after applying shared and local
// configuration. Terminal means that no explicit base colors should be used.
type ResolvedTheme struct {
	Colors   map[string]string
	Terminal bool
}

// Config is the root config struct.
type Config struct {
	Keybinds Keybinds `toml:"keybinds"`
	Display  Display  `toml:"display"`
	Apps     Apps     `toml:"apps"`
	Updates  Updates  `toml:"updates"`
	Plugins  Plugins  `toml:"plugins"`
	Themes   Themes   `toml:"themes"`
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
	{"confirm", "expand dir / edit file"},
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
	{"show_updates", "show update history and installers"},
	{"plugins", "show optional plugin integrations"},
	{"preview_toggle", "toggle file/image preview panel"},
	{"preview_mode", "swap image / details in preview panel"},
	{"theme", "open theme picker"},
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

// updateEntries is the authoritative list of every [updates] TOML key.
var updateEntries = []string{
	"disable_checks",
	"current_commit",
	"repo_path",
	"terminal",
}

// pluginEntries is the authoritative list of every [plugins] TOML key.
var pluginEntries = []string{
	"fd",
	"rg",
	"zoxide",
}

var themeEntries = []string{
	"theme_name",
	"theme_file",
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
			ShowUpdates:      "U",
			Plugins:          "P",
			PreviewToggle:    "v",
			PreviewMode:      "V",
			Theme:            "T",
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
		Updates: Updates{
			DisableChecks: false,
			CurrentCommit: "",
			RepoPath:      "",
			Terminal:      "",
		},
		Plugins: Plugins{
			Fd:     true,
			Rg:     true,
			Zoxide: true,
		},
		Themes: Themes{
			ThemeName: "terminal",
			ThemeFile: filepath.Join(ConfigDir(), "themes.toml"),
		},
	}
}

// ConfigDir returns the platform-appropriate path to the listicles config directory.
//
// os.UserConfigDir() returns the right base per OS:
//   - Windows: %APPDATA%  (e.g. C:\Users\wing\AppData\Roaming\delbysoft)
//   - macOS:   ~/Library/Application Support/delbysoft
//   - Linux:   ~/.config/delbysoft  (XDG_CONFIG_HOME if set, else ~/.config)
func ConfigDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return ""
		}
		return filepath.Join(home, ".config", "delbysoft")
	}
	return filepath.Join(base, "delbysoft")
}

// ConfigPath returns the full path to the config file.
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "listicles.toml")
}

// ThemeFilePath expands the configured theme file path. A leading ~/ is
// resolved against the current user's home directory.
func ThemeFilePath(cfg *Config) string {
	if cfg == nil || strings.TrimSpace(cfg.Themes.ThemeFile) == "" {
		return filepath.Join(ConfigDir(), "themes.toml")
	}
	path := strings.TrimSpace(cfg.Themes.ThemeFile)
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, strings.TrimLeft(path[1:], `/\`))
		}
	}
	return filepath.Clean(path)
}

// EnsureThemesFile creates the shared theme file if it is missing. Existing
// files are never overwritten, including during a factory reset.
func EnsureThemesFile(cfg *Config) error {
	path := ThemeFilePath(cfg)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		updated := appendMissingStarterThemes(string(data))
		if updated == string(data) {
			return nil
		}
		return os.WriteFile(path, []byte(updated), 0644)
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(path, []byte(defaultThemesTOML), 0644)
}

// ThemeNames returns terminal plus all named themes in the shared file.
func ThemeNames(cfg *Config) ([]string, error) {
	var file themeFile
	if _, err := toml.DecodeFile(ThemeFilePath(cfg), &file); err != nil {
		return []string{"terminal"}, err
	}
	names := []string{"terminal"}
	for name := range file.Themes {
		names = append(names, name)
	}
	sort.Strings(names[1:])
	return names, nil
}

// SetThemeName updates only the selected theme in listicles.toml.
func SetThemeName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("theme name cannot be empty")
	}
	if err := EnsureConfig(false); err != nil {
		return err
	}
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return err
	}
	content := setSectionKey(string(data), "themes", "theme_name", quote(name))
	return os.WriteFile(ConfigPath(), []byte(content), 0644)
}

func appendMissingStarterThemes(content string) string {
	for _, name := range starterThemeNames {
		header := "[themes." + name + "]"
		if strings.Contains(content, header) {
			continue
		}
		block := starterThemeBlock(name)
		if block == "" {
			continue
		}
		if strings.TrimSpace(content) != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += "\n" + block + "\n"
	}
	return content
}

func starterThemeBlock(name string) string {
	start := strings.Index(defaultThemesTOML, "[themes."+name+"]")
	if start < 0 {
		return ""
	}
	end := strings.Index(defaultThemesTOML[start:], "\n\n[themes.")
	if end < 0 {
		return strings.TrimSpace(defaultThemesTOML[start:])
	}
	return strings.TrimSpace(defaultThemesTOML[start : start+end])
}

// ValidateThemeFile checks the configured shared file without changing it.
// Runtime loading remains forgiving and uses the built-in theme on failure.
func ValidateThemeFile(cfg *Config) error {
	path := ThemeFilePath(cfg)
	var file themeFile
	if _, err := toml.DecodeFile(path, &file); err != nil {
		return fmt.Errorf("invalid themes file %q: %w", path, err)
	}
	if len(file.Themes) == 0 {
		return fmt.Errorf("themes file %q contains no [themes.<name>] entries", path)
	}
	return nil
}

// ResolveTheme loads the selected shared theme and applies local overrides.
// A zero-valued result means the caller should use its built-in fallback.
func ResolveTheme(cfg *Config) ResolvedTheme {
	if cfg == nil {
		return ResolvedTheme{}
	}
	result := ResolvedTheme{Colors: map[string]string{}, Terminal: cfg.Themes.ThemeName == "terminal"}
	if !result.Terminal {
		var file themeFile
		if _, err := toml.DecodeFile(ThemeFilePath(cfg), &file); err != nil {
			return ResolvedTheme{}
		}
		selected, ok := file.Themes[cfg.Themes.ThemeName]
		if !ok {
			return ResolvedTheme{}
		}
		result.Colors = themeColors(selected.Themes)
		if len(result.Colors) == 0 {
			return ResolvedTheme{}
		}
	}
	for key, value := range themeColors(cfg.Themes) {
		result.Colors[key] = value
	}
	return result
}

func themeColors(t Themes) map[string]string {
	values := map[string]string{
		"foreground":          t.Foreground,
		"background":          t.Background,
		"primary":             t.Primary,
		"accent":              t.Accent,
		"muted":               t.Muted,
		"error":               t.Error,
		"success":             t.Success,
		"file":                t.File,
		"border":              t.Border,
		"selected_background": t.SelectedBackground,
		"selected_foreground": t.SelectedForeground,
		"header_background":   t.HeaderBackground,
		"hint_key":            t.HintKey,
		"parent_crumb":        t.ParentCrumb,
		"root_directory":      t.RootDirectory,
		"clipboard":           t.Clipboard,
		"brand_primary":       t.BrandPrimary,
		"brand_secondary":     t.BrandSecondary,
		"selector":            t.Selector,
		"image_background":    t.ImageBackground,
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		if validThemeColor(value) {
			result[key] = value
		}
	}
	return result
}

func validThemeColor(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if len(value) != 4 && len(value) != 7 {
		return false
	}
	if value[0] != '#' {
		return false
	}
	for _, c := range value[1:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

var starterThemeNames = []string{
	"ocean", "high_contrast", "redteam", "blueteam", "vim", "neovim",
	"monotone", "cyberpunk", "sands",
}

const defaultThemesTOML = `# Shared themes for Delbysoft terminal applications.
# Add themes as [themes.name] tables. Colors use #RGB or #RRGGBB values.
# Supported colors: foreground, background, primary, accent, muted, error,
# success, file, border, selected_background, selected_foreground,
# header_background, hint_key, parent_crumb, root_directory, clipboard,
# brand_primary, brand_secondary, selector, image_background.

[themes.ocean]
foreground = "#D7E3FF"
background = "#101522"
primary = "#7C9EF0"
accent = "#F0A47C"
muted = "#66708F"
error = "#F07C7C"
success = "#7CF09C"
file = "#B0B0CC"
border = "#35415F"
selected_background = "#3568B8"
selected_foreground = "#FFFFFF"
header_background = "#17213A"
hint_key = "#FFE66D"
parent_crumb = "#58627F"
root_directory = "#7D88A8"
clipboard = "#F0E07C"
brand_primary = "#FFFFFF"
brand_secondary = "#6F86FF"
selector = "#FFFFFF"
image_background = "#101522"

[themes.high_contrast]
foreground = "#FFFFFF"
background = "#000000"
primary = "#00FFFF"
accent = "#FFFF00"
muted = "#C0C0C0"
error = "#FF5555"
success = "#00FF00"
file = "#FFFFFF"
border = "#FFFFFF"
selected_background = "#FFFF00"
selected_foreground = "#000000"
header_background = "#000000"
hint_key = "#FFFF00"
parent_crumb = "#C0C0C0"
root_directory = "#FFFFFF"
clipboard = "#FFFF00"
brand_primary = "#FFFFFF"
brand_secondary = "#00FFFF"
selector = "#FFFF00"
image_background = "#000000"

[themes.redteam]
foreground = "#FFE8E8"
background = "#210B0B"
primary = "#FF6B6B"
accent = "#FFB86B"
muted = "#A97878"
error = "#FF3333"
success = "#8BE28B"
file = "#F2CACA"
border = "#713333"
selected_background = "#9E2020"
selected_foreground = "#FFFFFF"
header_background = "#3A1010"
hint_key = "#FFD166"
parent_crumb = "#805555"
root_directory = "#C88A8A"
clipboard = "#FFD166"
brand_primary = "#FFFFFF"
brand_secondary = "#FF4D4D"
selector = "#FFFFFF"
image_background = "#210B0B"

[themes.blueteam]
foreground = "#E7F1FF"
background = "#081525"
primary = "#69A7FF"
accent = "#72E0D1"
muted = "#6D86A5"
error = "#FF7B8B"
success = "#7DDEB3"
file = "#C4D8F2"
border = "#294C75"
selected_background = "#1557A5"
selected_foreground = "#FFFFFF"
header_background = "#0D223D"
hint_key = "#F4D35E"
parent_crumb = "#4D6888"
root_directory = "#8BA9CB"
clipboard = "#F4D35E"
brand_primary = "#FFFFFF"
brand_secondary = "#69A7FF"
selector = "#FFFFFF"
image_background = "#081525"

[themes.vim]
foreground = "#D7D7AF"
background = "#1C1C1C"
primary = "#87AF87"
accent = "#D7AF5F"
muted = "#808080"
error = "#AF5F5F"
success = "#87AF87"
file = "#D7D7AF"
border = "#5F5F5F"
selected_background = "#5F5F00"
selected_foreground = "#FFFFAF"
header_background = "#262626"
hint_key = "#FFFF87"
parent_crumb = "#5F875F"
root_directory = "#AFAF87"
clipboard = "#D7AF5F"
brand_primary = "#FFFFFF"
brand_secondary = "#87AF87"
selector = "#FFFFAF"
image_background = "#1C1C1C"

[themes.neovim]
foreground = "#C8D3F5"
background = "#1B1D2B"
primary = "#82AAFF"
accent = "#FFC777"
muted = "#828BB8"
error = "#FF757F"
success = "#C3E88D"
file = "#C8D3F5"
border = "#444A73"
selected_background = "#394B70"
selected_foreground = "#FFFFFF"
header_background = "#222436"
hint_key = "#FFCB6B"
parent_crumb = "#545C8C"
root_directory = "#A9B8E8"
clipboard = "#C3E88D"
brand_primary = "#FFFFFF"
brand_secondary = "#82AAFF"
selector = "#FFFFFF"
image_background = "#1B1D2B"

[themes.monotone]
foreground = "#D0D0D0"
background = "#202020"
primary = "#E0E0E0"
accent = "#FFFFFF"
muted = "#808080"
error = "#B0B0B0"
success = "#D8D8D8"
file = "#C0C0C0"
border = "#606060"
selected_background = "#D0D0D0"
selected_foreground = "#101010"
header_background = "#303030"
hint_key = "#FFFFFF"
parent_crumb = "#707070"
root_directory = "#A0A0A0"
clipboard = "#FFFFFF"
brand_primary = "#FFFFFF"
brand_secondary = "#A0A0A0"
selector = "#FFFFFF"
image_background = "#202020"

[themes.cyberpunk]
foreground = "#F4E8FF"
background = "#170D24"
primary = "#00E5FF"
accent = "#FFEA00"
muted = "#9B75B5"
error = "#FF3864"
success = "#39FF14"
file = "#E6CFFF"
border = "#7A2F9B"
selected_background = "#D100A8"
selected_foreground = "#FFFFFF"
header_background = "#28113C"
hint_key = "#FFEA00"
parent_crumb = "#754D91"
root_directory = "#C68AF0"
clipboard = "#FFEA00"
brand_primary = "#FFFFFF"
brand_secondary = "#00E5FF"
selector = "#FFFFFF"
image_background = "#170D24"

[themes.sands]
foreground = "#F3E7CE"
background = "#282016"
primary = "#E4B96A"
accent = "#F2D06B"
muted = "#9F8B6D"
error = "#D9795B"
success = "#A8B875"
file = "#E8D6B5"
border = "#6D583C"
selected_background = "#A66A2C"
selected_foreground = "#FFF4D6"
header_background = "#382A1B"
hint_key = "#F2D06B"
parent_crumb = "#806B4E"
root_directory = "#CBAE7A"
clipboard = "#F2D06B"
brand_primary = "#FFF4D6"
brand_secondary = "#E4B96A"
selector = "#FFF4D6"
image_background = "#282016"
`

// Load reads the config file, creating it with defaults if it doesn't exist.
func Load() (*Config, error) {
	cfg := Default()
	path := ConfigPath()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := EnsureConfig(false); err != nil {
			return cfg, nil
		}
		_ = EnsureThemesFile(cfg)
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		_ = EnsureThemesFile(cfg)
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
	if cfg.Themes.ThemeName == "" {
		cfg.Themes.ThemeName = "terminal"
	}
	if cfg.Themes.ThemeFile == "" {
		cfg.Themes.ThemeFile = filepath.Join(ConfigDir(), "themes.toml")
	}

	// Migration: add only missing keys, preserving user values and existing text.
	if needsMigration(path) {
		_ = EnsureConfig(false) // non-fatal
	}
	_ = EnsureThemesFile(cfg) // themes are optional at runtime; built-in fallback remains available

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
	if cfg.Keybinds.ShowUpdates == "" {
		cfg.Keybinds.ShowUpdates = d.ShowUpdates
	}
	if cfg.Keybinds.Plugins == "" {
		cfg.Keybinds.Plugins = d.Plugins
	}
	if cfg.Keybinds.PreviewToggle == "" {
		cfg.Keybinds.PreviewToggle = d.PreviewToggle
	}
	if cfg.Keybinds.PreviewMode == "" {
		cfg.Keybinds.PreviewMode = d.PreviewMode
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
	for _, key := range updateEntries {
		if !fileContainsKey(content, key) {
			return true
		}
	}
	for _, key := range pluginEntries {
		if !fileContainsKey(content, key) {
			return true
		}
	}
	for _, key := range themeEntries {
		if !fileContainsKeyInSection(content, "themes", key) {
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

func fileContainsKeyInSection(content, section, key string) bool {
	inSection := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			inSection = trimmed == "["+section+"]"
			continue
		}
		if inSection && !strings.HasPrefix(trimmed, "#") &&
			(strings.HasPrefix(trimmed, key+"=") || strings.HasPrefix(trimmed, key+" ")) {
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

// EnsureConfig creates the config if missing and adds only missing known keys.
// When resetDefault is true, it intentionally rewrites the full file to defaults.
func EnsureConfig(resetDefault bool) error {
	path := ConfigPath()
	if err := os.MkdirAll(ConfigDir(), 0755); err != nil {
		return err
	}
	if resetDefault {
		if err := WriteDefault(path); err != nil {
			return err
		}
		return EnsureThemesFile(Default())
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if err := WriteDefault(path); err != nil {
			return err
		}
		return EnsureThemesFile(Default())
	}
	if err != nil {
		return err
	}
	content := string(data)
	updated := ensureMissingConfigKeys(content)
	if updated == content {
		cfg := Default()
		if _, decodeErr := toml.Decode(content, cfg); decodeErr != nil {
			cfg = Default()
		}
		return EnsureThemesFile(cfg)
	}
	if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
		return err
	}
	cfg := Default()
	if _, decodeErr := toml.Decode(updated, cfg); decodeErr != nil {
		cfg = Default()
	}
	return EnsureThemesFile(cfg)
}

func ensureMissingConfigKeys(content string) string {
	content = ensureSectionEntries(content, "keybinds", keybindDefaultLines())
	content = ensureSectionEntries(content, "display", displayDefaultLines())
	content = ensureSectionEntries(content, "apps", appsDefaultLines())
	content = ensureSectionEntries(content, "updates", updatesDefaultLines())
	content = ensureSectionEntries(content, "plugins", pluginsDefaultLines())
	content = ensureSectionEntries(content, "themes", themeDefaultLines())
	return content
}

func ensureSectionEntries(content, section string, entries map[string]string) string {
	for _, key := range orderedKeys(entries) {
		if sectionContainsKey(content, section, key) {
			continue
		}
		content = insertSectionLine(content, section, entries[key])
	}
	return content
}

func sectionContainsKey(content, section, key string) bool {
	inSection := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			inSection = trimmed == "["+section+"]"
			continue
		}
		if !inSection || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, key+"=") || strings.HasPrefix(trimmed, key+" ") {
			return true
		}
	}
	return false
}

func insertSectionLine(content, section, line string) string {
	newline := "\n"
	if strings.Contains(content, "\r\n") {
		newline = "\r\n"
	}
	lines := strings.Split(content, newline)
	sectionHeader := "[" + section + "]"
	sectionIdx := -1
	insertIdx := len(lines)
	for i, lineText := range lines {
		trimmed := strings.TrimSpace(lineText)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			if trimmed == sectionHeader {
				sectionIdx = i
				insertIdx = len(lines)
				continue
			}
			if sectionIdx >= 0 {
				insertIdx = i
				break
			}
		}
	}
	if sectionIdx < 0 {
		if strings.TrimSpace(content) != "" && !strings.HasSuffix(content, newline) {
			content += newline
		}
		return content + newline + sectionHeader + newline + line + newline
	}
	lines = append(lines[:insertIdx], append([]string{line}, lines[insertIdx:]...)...)
	return strings.Join(lines, newline)
}

func orderedKeys(entries map[string]string) []string {
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

func keybindDefaultLines() map[string]string {
	d := Default().Keybinds
	vals := keybindValues(&d)
	lines := map[string]string{}
	for _, e := range keybindEntries {
		lines[e.key] = e.key + " = " + quote(vals[e.key])
	}
	return lines
}

func displayDefaultLines() map[string]string {
	d := Default().Display
	return map[string]string{
		"show_hidden":        "show_hidden = " + boolStr(d.ShowHidden),
		"default_list_mode":  "default_list_mode = " + quote(d.DefaultListMode),
		"search_max_results": "search_max_results = " + itoa(d.SearchMaxResults),
		"parent_depth":       "parent_depth = " + itoa(d.ParentDepth),
	}
}

func appsDefaultLines() map[string]string {
	d := Default().Apps
	return map[string]string{
		"editor": "editor = " + quote(d.Editor),
		"opener": "opener = " + quote(d.Opener),
	}
}

func updatesDefaultLines() map[string]string {
	d := Default().Updates
	return map[string]string{
		"disable_checks": "disable_checks = " + boolStr(d.DisableChecks),
		"current_commit": "current_commit = " + quote(d.CurrentCommit),
		"repo_path":      "repo_path = " + quote(d.RepoPath),
		"terminal":       "terminal = " + quote(d.Terminal),
	}
}

func pluginsDefaultLines() map[string]string {
	d := Default().Plugins
	return map[string]string{
		"fd":     "fd = " + boolStr(d.Fd),
		"rg":     "rg = " + boolStr(d.Rg),
		"zoxide": "zoxide = " + boolStr(d.Zoxide),
	}
}

func themeDefaultLines() map[string]string {
	d := Default().Themes
	return map[string]string{
		"theme_name": "theme_name = " + quote(d.ThemeName),
		"theme_file": "theme_file = " + quote(d.ThemeFile),
	}
}

// SetPluginEnabled updates one plugin toggle in the user's config file.
func SetPluginEnabled(name string, enabled bool) error {
	switch name {
	case "fd", "rg", "zoxide":
	default:
		return errors.New("unknown plugin")
	}
	if err := EnsureConfig(false); err != nil {
		return err
	}
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := setSectionKey(string(data), "plugins", name, boolStr(enabled))
	return os.WriteFile(path, []byte(content), 0644)
}

// RecordUpdateMetadata stores the installed commit and source repo path without
// changing user-facing preferences.
func RecordUpdateMetadata(commit, repoPath string) error {
	if err := EnsureConfig(false); err != nil {
		return err
	}
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	if commit != "" {
		content = setSectionKey(content, "updates", "current_commit", quote(commit))
	}
	if repoPath != "" {
		content = setSectionKey(content, "updates", "repo_path", quote(repoPath))
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func setSectionKey(content, section, key, value string) string {
	if !sectionContainsKey(content, section, key) {
		return insertSectionLine(content, section, key+" = "+value)
	}
	newline := "\n"
	if strings.Contains(content, "\r\n") {
		newline = "\r\n"
	}
	lines := strings.Split(content, newline)
	inSection := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			inSection = trimmed == "["+section+"]"
			continue
		}
		if !inSection || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, key+"=") || strings.HasPrefix(trimmed, key+" ") {
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			comment := ""
			if idx := strings.Index(line, "#"); idx >= 0 {
				comment = " " + strings.TrimSpace(line[idx:])
			}
			lines[i] = indent + key + " = " + value + comment
			break
		}
	}
	return strings.Join(lines, newline)
}

// buildTOML generates the full config file content from a Config, using
// keybindEntries as the authoritative source so the output stays in sync
// automatically when new keybinds are added.
func buildTOML(cfg *Config) string {
	vals := keybindValues(&cfg.Keybinds)
	d := cfg.Display
	a := cfg.Apps
	u := cfg.Updates
	p := cfg.Plugins
	t := cfg.Themes

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
		"opener = " + quote(a.Opener) + "   # leave empty to use xdg-open (Linux) / open (macOS)\n\n" +
		"[updates]\n" +
		"disable_checks = " + boolStr(u.DisableChecks) + "   # true disables startup update checks\n" +
		"current_commit = " + quote(u.CurrentCommit) + "   # installed app commit, maintained by listicles\n" +
		"repo_path = " + quote(u.RepoPath) + "   # source checkout used for updates\n" +
		"terminal = " + quote(u.Terminal) + "   # optional terminal command for detached updates\n\n" +
		"[plugins]\n" +
		"fd = " + boolStr(p.Fd) + "   # use fd for full name search when installed\n" +
		"rg = " + boolStr(p.Rg) + "   # use rg for full content search when installed\n" +
		"zoxide = " + boolStr(p.Zoxide) + "   # use zoxide for -z directory search when installed\n"

	out += "\n[themes]\n" +
		"theme_name = " + quote(t.ThemeName) + "   # terminal, or a named theme from theme_file\n" +
		"theme_file = " + quote(t.ThemeFile) + "   # shared Delbysoft theme file\n" +
		"# Optional overrides applied after the selected theme.\n" +
		"# foreground = \"#ffffff\"\n" +
		"# background = \"#000000\"\n" +
		"# primary = \"#7C9EF0\"\n" +
		"# accent = \"#F0A47C\"\n" +
		"# muted = \"#666688\"\n" +
		"# error = \"#F07C7C\"\n" +
		"# success = \"#7CF09C\"\n" +
		"# file = \"#B0B0CC\"\n" +
		"# border = \"#444466\"\n" +
		"# selected_background = \"#cd0fc1\"\n" +
		"# selected_foreground = \"#EEEEFF\"\n" +
		"# header_background = \"#1A1A2E\"\n" +
		"# hint_key = \"#FFE66D\"\n" +
		"# parent_crumb = \"#3A3A5A\"\n" +
		"# root_directory = \"#555577\"\n" +
		"# clipboard = \"#F0E07C\"\n" +
		"# brand_primary = \"#FFFFFF\"\n" +
		"# brand_secondary = \"#5865F2\"\n" +
		"# selector = \"#FFFFFF\"\n" +
		"# image_background = \"#1A1A2E\"\n"

	return out
}

// keybindValues maps every TOML key name to the current value in k.
// This is the one place that couples struct fields to their TOML names for writing.
func keybindValues(k *Keybinds) map[string]string {
	return map[string]string{
		"up":                 k.Up,
		"down":               k.Down,
		"left":               k.Left,
		"right":              k.Right,
		"confirm":            k.Confirm,
		"parent":             k.Parent,
		"page_up":            k.PageUp,
		"page_down":          k.PageDown,
		"jump_top":           k.JumpTop,
		"jump_bottom":        k.JumpBottom,
		"quit":               k.Quit,
		"options":            k.Options,
		"add":                k.Add,
		"delete":             k.Delete,
		"toggle_list":        k.ToggleList,
		"rename":             k.Rename,
		"edit":               k.Edit,
		"yank":               k.Yank,
		"cut":                k.Cut,
		"paste":              k.Paste,
		"copy_path":          k.CopyPath,
		"details":            k.Details,
		"toggle_hidden":      k.ToggleHidden,
		"search":             k.Search,
		"switch_tabs":        k.SwitchTabs,
		"switch_tabs_global": k.SwitchTabsGlobal,
		"ignore":             k.Ignore,
		"full_search":        k.FullSearch,
		"cd_dir":             k.CDDir,
		"open_explorer":      k.OpenExplorer,
		"bookmark":           k.Bookmark,
		"show_hints":         k.ShowHints,
		"show_updates":       k.ShowUpdates,
		"plugins":            k.Plugins,
		"preview_toggle":     k.PreviewToggle,
		"preview_mode":       k.PreviewMode,
		"theme":              k.Theme,
	}
}

func quote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
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
