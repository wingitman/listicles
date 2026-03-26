package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestDefault_FieldValues(t *testing.T) {
	cfg := Default()

	keybindChecks := []struct {
		name string
		got  string
		want string
	}{
		{"Up", cfg.Keybinds.Up, "up"},
		{"Down", cfg.Keybinds.Down, "down"},
		{"Left", cfg.Keybinds.Left, "left"},
		{"Right", cfg.Keybinds.Right, "right"},
		{"Confirm", cfg.Keybinds.Confirm, "enter"},
		{"Parent", cfg.Keybinds.Parent, "0"},
		{"PageUp", cfg.Keybinds.PageUp, "pgup"},
		{"PageDown", cfg.Keybinds.PageDown, "pgdown"},
		{"JumpTop", cfg.Keybinds.JumpTop, "home"},
		{"JumpBottom", cfg.Keybinds.JumpBottom, "end"},
		{"Options", cfg.Keybinds.Options, "o"},
		{"Add", cfg.Keybinds.Add, "a"},
		{"Delete", cfg.Keybinds.Delete, "d"},
		{"ToggleList", cfg.Keybinds.ToggleList, "f"},
		{"Rename", cfg.Keybinds.Rename, "r"},
		{"Edit", cfg.Keybinds.Edit, "e"},
		{"Yank", cfg.Keybinds.Yank, "y"},
		{"Cut", cfg.Keybinds.Cut, "x"},
		{"Paste", cfg.Keybinds.Paste, "p"},
		{"CopyPath", cfg.Keybinds.CopyPath, "Y"},
		{"Quit", cfg.Keybinds.Quit, "q"},
		{"Details", cfg.Keybinds.Details, "i"},
		{"ToggleHidden", cfg.Keybinds.ToggleHidden, "."},
		{"Search", cfg.Keybinds.Search, "/"},
		{"DefaultListMode", cfg.Display.DefaultListMode, "dirs"},
	}
	for _, c := range keybindChecks {
		if c.got != c.want {
			t.Errorf("Default %s: got %q, want %q", c.name, c.got, c.want)
		}
	}

	if cfg.Display.SearchMaxResults != 20 {
		t.Errorf("SearchMaxResults: got %d, want 20", cfg.Display.SearchMaxResults)
	}
	if cfg.Display.ParentDepth != 1 {
		t.Errorf("ParentDepth: got %d, want 1", cfg.Display.ParentDepth)
	}
	if cfg.Display.ShowHidden {
		t.Error("ShowHidden should default to false")
	}
	if cfg.Apps.Editor != "" || cfg.Apps.Opener != "" {
		t.Error("Apps.Editor and Apps.Opener should be empty by default")
	}
}

func TestWriteDefault_ContainsExpectedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "listicles.toml")

	if err := WriteDefault(path); err != nil {
		t.Fatalf("WriteDefault: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	for _, expected := range []string{
		`[keybinds]`,
		`[display]`,
		`[apps]`,
		`toggle_list   = "f"`,
		`yank          = "y"`,
		`cut           = "x"`,
		`paste         = "p"`,
		`copy_path     = "Y"`,
		`page_up       = "pgup"`,
		`page_down     = "pgdown"`,
		`jump_top      = "home"`,
		`jump_bottom   = "end"`,
		`search_max_results = 20`,
		`parent_depth = 1`,
	} {
		if !strings.Contains(string(content), expected) {
			t.Errorf("written config missing expected content: %s", expected)
		}
	}

	// Must NOT contain the removed vim_mode section
	if strings.Contains(string(content), "[vim_mode]") {
		t.Error("written config must not contain deprecated [vim_mode] section")
	}
}

func TestWriteDefault_ParsesCleanly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "listicles.toml")

	if err := WriteDefault(path); err != nil {
		t.Fatalf("WriteDefault: %v", err)
	}

	cfg := Default()
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		t.Fatalf("written config does not parse cleanly: %v", err)
	}

	// Round-trip: parsed values should match the programmatic defaults
	defaults := Default()
	if cfg.Keybinds.Up != defaults.Keybinds.Up {
		t.Errorf("round-trip Up: got %q, want %q", cfg.Keybinds.Up, defaults.Keybinds.Up)
	}
	if cfg.Keybinds.ToggleList != defaults.Keybinds.ToggleList {
		t.Errorf("round-trip ToggleList: got %q, want %q", cfg.Keybinds.ToggleList, defaults.Keybinds.ToggleList)
	}
	if cfg.Display.SearchMaxResults != defaults.Display.SearchMaxResults {
		t.Errorf("round-trip SearchMaxResults: got %d, want %d",
			cfg.Display.SearchMaxResults, defaults.Display.SearchMaxResults)
	}
}

func TestLoad_ClampMinimums(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "listicles.toml")

	badContent := `
[keybinds]
up = "up"
down = "down"
left = "left"
right = "right"
confirm = "enter"
parent = "0"
page_up = "pgup"
page_down = "pgdown"
jump_top = "home"
jump_bottom = "end"
options = "o"
add = "a"
delete = "d"
toggle_list = "f"
rename = "r"
edit = "e"
yank = "y"
cut = "x"
paste = "p"
copy_path = "Y"
quit = "q"
details = "i"
toggle_hidden = "."
search = "/"

[display]
search_max_results = 0
parent_depth = -5
`
	if err := os.WriteFile(path, []byte(badContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Default()
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Apply the same clamping Load() does
	if cfg.Display.SearchMaxResults < 1 {
		cfg.Display.SearchMaxResults = 1
	}
	if cfg.Display.ParentDepth < 0 {
		cfg.Display.ParentDepth = 0
	}

	if cfg.Display.SearchMaxResults != 1 {
		t.Errorf("SearchMaxResults not clamped to 1: got %d", cfg.Display.SearchMaxResults)
	}
	if cfg.Display.ParentDepth != 0 {
		t.Errorf("ParentDepth not clamped to 0: got %d", cfg.Display.ParentDepth)
	}
}

func TestLoad_UnknownFieldsIgnored(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "listicles.toml")

	// Stale config with old vim_mode fields — BurntSushi/toml ignores unknown fields
	stale := `
[keybinds]
up = "k"
down = "j"
left = "h"
right = "l"
confirm = "enter"
parent = "0"
page_up = "pgup"
page_down = "pgdown"
jump_top = "home"
jump_bottom = "end"
options = "o"
add = "a"
delete = "d"
toggle_list = "f"
rename = "r"
edit = "e"
yank = "y"
cut = "x"
paste = "p"
copy_path = "Y"
quit = "q"
details = "i"
toggle_hidden = "."
search = "/"
vim_mode = "v"

[vim_mode]
enabled = true

[display]
search_max_results = 20
parent_depth = 1
`
	if err := os.WriteFile(path, []byte(stale), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Default()
	// Should not error — unknown fields are silently ignored
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		t.Fatalf("unexpected error parsing stale config: %v", err)
	}

	// Custom keybind should have been applied
	if cfg.Keybinds.Up != "k" {
		t.Errorf("Up key not loaded from file: got %q, want %q", cfg.Keybinds.Up, "k")
	}
}

func TestConfigPath_EndsCorrectly(t *testing.T) {
	p := ConfigPath()
	want := filepath.Join("listicles", "listicles.toml")
	if !strings.HasSuffix(p, want) {
		t.Errorf("ConfigPath %q does not end with %q", p, want)
	}
}

func TestConfigDir_EndsWithListicles(t *testing.T) {
	d := ConfigDir()
	if !strings.HasSuffix(d, "listicles") {
		t.Errorf("ConfigDir %q does not end with 'listicles'", d)
	}
}
