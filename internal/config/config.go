package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Keybinds holds all configurable key mappings.
// Key values use BubbleTea key names: "up", "down", "left", "right", "enter",
// "pgup", "pgdown", "home", "end", "ctrl+u", or single characters like "q".
type Keybinds struct {
	Up           string `toml:"up"`
	Down         string `toml:"down"`
	Left         string `toml:"left"`
	Right        string `toml:"right"`
	Confirm      string `toml:"confirm"`
	Parent       string `toml:"parent"`
	PageUp       string `toml:"page_up"`
	PageDown     string `toml:"page_down"`
	JumpTop      string `toml:"jump_top"`
	JumpBottom   string `toml:"jump_bottom"`
	Options      string `toml:"options"`
	Add          string `toml:"add"`
	Delete       string `toml:"delete"`
	ToggleList   string `toml:"toggle_list"`
	Rename       string `toml:"rename"`
	Edit         string `toml:"edit"`
	Yank         string `toml:"yank"`
	Cut          string `toml:"cut"`
	Paste        string `toml:"paste"`
	CopyPath     string `toml:"copy_path"`
	Quit         string `toml:"quit"`
	Details      string `toml:"details"`
	ToggleHidden string `toml:"toggle_hidden"`
	Search       string `toml:"search"`
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

// Default returns a Config with all default values.
func Default() *Config {
	return &Config{
		Keybinds: Keybinds{
			Up:           "up",
			Down:         "down",
			Left:         "left",
			Right:        "right",
			Confirm:      "enter",
			Parent:       "0",
			PageUp:       "pgup",
			PageDown:     "pgdown",
			JumpTop:      "home",
			JumpBottom:   "end",
			Options:      "o",
			Add:          "a",
			Delete:       "d",
			ToggleList:   "f",
			Rename:       "r",
			Edit:         "e",
			Yank:         "y",
			Cut:          "x",
			Paste:        "p",
			CopyPath:     "Y",
			Quit:         "q",
			Details:      "i",
			ToggleHidden: ".",
			Search:       "/",
		},
		Display: Display{
			ShowHidden:       false,
			DefaultListMode:  "dirs",
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
		// Fallback: construct manually from home dir
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

// resolvePath returns the config file path to use. It checks the primary
// (OS-native) path first, then falls back to the legacy Unix-style path
// (~/.config/delbysoft/listicles.toml) for users who already have a config
// there from an earlier install. Returns the primary path when neither exists
// so that WriteDefault writes to the correct location.
func resolvePath() string {
	primary := ConfigPath()
	if _, err := os.Stat(primary); err == nil {
		return primary
	}

	// Legacy path: always ~/.config/delbysoft/listicles.toml
	if home, err := os.UserHomeDir(); err == nil {
		legacy := filepath.Join(home, ".config", "listicles", "listicles.toml")
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
	if cfg.Display.SearchMaxResults < 1 {
		cfg.Display.SearchMaxResults = 1
	}
	if cfg.Display.ParentDepth < 0 {
		cfg.Display.ParentDepth = 0
	}
	return cfg, nil
}

// WriteDefault writes the default config file to path.
func WriteDefault(path string) error {
	content := `# listicles configuration file
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
default_list_mode  = "dirs"   # "dirs" or "dirs_and_files"

# Max results shown during live / subprocess search (min 1)
search_max_results = 20

# Greyed-out ancestor directories shown above the tree.
# Set to 0 to disable. Default 1 shows the immediate parent.
parent_depth = 1

[apps]
editor = ""   # leave empty to use $EDITOR env var
opener = ""   # leave empty to use xdg-open (Linux) / open (macOS)
`
	return os.WriteFile(path, []byte(content), 0644)
}
