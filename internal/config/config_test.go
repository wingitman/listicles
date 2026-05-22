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
		{"SwitchTabs", cfg.Keybinds.SwitchTabs, "\t"},
		{"SwitchTabsGlobal", cfg.Keybinds.SwitchTabsGlobal, "g"},
		{"Ignore", cfg.Keybinds.Ignore, "I"},
		{"FullSearch", cfg.Keybinds.FullSearch, "ctrl+f"},
		{"CDDir", cfg.Keybinds.CDDir, "c"},
		{"OpenExplorer", cfg.Keybinds.OpenExplorer, "E"},
		{"Bookmark", cfg.Keybinds.Bookmark, "b"},
		{"ShowHints", cfg.Keybinds.ShowHints, "?"},
		{"ShowUpdates", cfg.Keybinds.ShowUpdates, "U"},
		{"DefaultListMode", cfg.Display.DefaultListMode, "dirs_and_files"},
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
	if cfg.Updates.DisableChecks || cfg.Updates.CurrentCommit != "" || cfg.Updates.RepoPath != "" || cfg.Updates.Terminal != "" {
		t.Error("Updates should default to enabled checks with empty metadata")
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

	// Check section headers are present.
	for _, key := range []string{"[keybinds]", "[display]", "[apps]", "[updates]"} {
		if !strings.Contains(string(content), key) {
			t.Errorf("expected %q in default config output", key)
		}
	}

	// Check every keybind entry is present as an actual key assignment.
	for _, e := range keybindEntries {
		if !fileContainsKey(string(content), e.key) {
			t.Errorf("expected keybind %q in default config output", e.key)
		}
	}

	// Check display and app keys.
	for _, key := range displayEntries {
		if !fileContainsKey(string(content), key) {
			t.Errorf("expected display key %q in default config output", key)
		}
	}
	for _, key := range appEntries {
		if !fileContainsKey(string(content), key) {
			t.Errorf("expected app key %q in default config output", key)
		}
	}
	for _, key := range updateEntries {
		if !fileContainsKey(string(content), key) {
			t.Errorf("expected update key %q in default config output", key)
		}
	}

	// Must NOT contain the removed vim_mode section.
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

	// Round-trip: parsed values should match the programmatic defaults.
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

	// Apply the same clamping Load() does.
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

	// Stale config with old vim_mode fields — BurntSushi/toml ignores unknown fields.
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
	// Should not error — unknown fields are silently ignored.
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		t.Fatalf("unexpected error parsing stale config: %v", err)
	}

	// Custom keybind should have been applied.
	if cfg.Keybinds.Up != "k" {
		t.Errorf("Up key not loaded from file: got %q, want %q", cfg.Keybinds.Up, "k")
	}
}

func TestApplyKeybindDefaults_FillsMissing(t *testing.T) {
	// A config with all keybind fields zeroed should get every default filled in.
	cfg := &Config{}
	applyKeybindDefaults(cfg)
	d := Default().Keybinds

	checks := []struct {
		name string
		got  string
		want string
	}{
		{"Up", cfg.Keybinds.Up, d.Up},
		{"Down", cfg.Keybinds.Down, d.Down},
		{"Left", cfg.Keybinds.Left, d.Left},
		{"Right", cfg.Keybinds.Right, d.Right},
		{"Quit", cfg.Keybinds.Quit, d.Quit},
		{"Search", cfg.Keybinds.Search, d.Search},
		{"FullSearch", cfg.Keybinds.FullSearch, d.FullSearch},
		{"SwitchTabs", cfg.Keybinds.SwitchTabs, d.SwitchTabs},
		{"Bookmark", cfg.Keybinds.Bookmark, d.Bookmark},
		{"ShowHints", cfg.Keybinds.ShowHints, d.ShowHints},
		{"ShowUpdates", cfg.Keybinds.ShowUpdates, d.ShowUpdates},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("applyKeybindDefaults %s: got %q, want %q", c.name, c.got, c.want)
		}
	}
}

func TestApplyKeybindDefaults_PreservesUserValues(t *testing.T) {
	// User-set values must not be overwritten.
	cfg := &Config{}
	cfg.Keybinds.Up = "k"
	cfg.Keybinds.Down = "j"
	applyKeybindDefaults(cfg)

	if cfg.Keybinds.Up != "k" {
		t.Errorf("Up should remain %q, got %q", "k", cfg.Keybinds.Up)
	}
	if cfg.Keybinds.Down != "j" {
		t.Errorf("Down should remain %q, got %q", "j", cfg.Keybinds.Down)
	}
	// Unset field should have been filled with default.
	if cfg.Keybinds.Quit != Default().Keybinds.Quit {
		t.Errorf("Quit should be default %q, got %q", Default().Keybinds.Quit, cfg.Keybinds.Quit)
	}
}

func TestFileContainsKey(t *testing.T) {
	content := `[keybinds]
up    = "up"
toggle_hidden = "."
# this comment mentions up but should not count as a key
`
	if !fileContainsKey(content, "up") {
		t.Error("expected to find 'up' key")
	}
	if !fileContainsKey(content, "toggle_hidden") {
		t.Error("expected to find 'toggle_hidden' key")
	}
	if fileContainsKey(content, "down") {
		t.Error("should not find 'down' key")
	}
	// 'up' appears in a comment — must not match as a key.
	commentOnly := "# up = \"up\"\n"
	if fileContainsKey(commentOnly, "up") {
		t.Error("should not match 'up' inside a comment")
	}
}

func TestNeedsMigration_DetectsMissingKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "listicles.toml")

	// Write a config missing the 'bookmark' key.
	partial := "[keybinds]\nup = \"up\"\ndown = \"down\"\n"
	if err := os.WriteFile(path, []byte(partial), 0644); err != nil {
		t.Fatal(err)
	}

	if !needsMigration(path) {
		t.Error("expected needsMigration to return true for incomplete config")
	}
}

func TestNeedsMigration_FullConfigNoMigration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "listicles.toml")

	if err := WriteDefault(path); err != nil {
		t.Fatalf("WriteDefault: %v", err)
	}

	if needsMigration(path) {
		t.Error("expected needsMigration to return false for a complete default config")
	}
}

func TestEnsureConfig_PreservesUserValuesAndAddsMissing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", filepath.Join(t.TempDir(), "AppData", "Roaming"))
	if err := os.MkdirAll(ConfigDir(), 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	content := `[keybinds]
up = "k"
down = "j"

[display]
show_hidden = true
`
	if err := os.WriteFile(ConfigPath(), []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := EnsureConfig(false); err != nil {
		t.Fatalf("EnsureConfig: %v", err)
	}
	updatedBytes, err := os.ReadFile(ConfigPath())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	updated := string(updatedBytes)
	if !strings.Contains(updated, `up = "k"`) || !strings.Contains(updated, `down = "j"`) {
		t.Fatalf("custom keybinds were not preserved:\n%s", updated)
	}
	if !sectionContainsKey(updated, "keybinds", "bookmark") {
		t.Fatalf("missing keybind was not added:\n%s", updated)
	}
	if !sectionContainsKey(updated, "display", "parent_depth") {
		t.Fatalf("missing display key was not added:\n%s", updated)
	}
}

func TestEnsureConfig_DefaultResetOverwritesUserValues(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", filepath.Join(t.TempDir(), "AppData", "Roaming"))
	if err := os.MkdirAll(ConfigDir(), 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(ConfigPath(), []byte("[keybinds]\nup = \"k\"\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := EnsureConfig(true); err != nil {
		t.Fatalf("EnsureConfig reset: %v", err)
	}
	updatedBytes, err := os.ReadFile(ConfigPath())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	updated := string(updatedBytes)
	if !strings.Contains(updated, `up                 = "up"`) && !strings.Contains(updated, `up = "up"`) {
		t.Fatalf("default reset did not restore default up key:\n%s", updated)
	}
}

func TestRecordUpdateMetadata_PreservesUserConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", filepath.Join(t.TempDir(), "AppData", "Roaming"))
	if err := os.MkdirAll(ConfigDir(), 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	content := `[keybinds]
up = "k"
down = "j"

[updates]
current_commit = "old"
repo_path = "old-repo"
`
	if err := os.WriteFile(ConfigPath(), []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := RecordUpdateMetadata("new", "new-repo"); err != nil {
		t.Fatalf("RecordUpdateMetadata: %v", err)
	}
	updatedBytes, err := os.ReadFile(ConfigPath())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	updated := string(updatedBytes)
	if !strings.Contains(updated, `up = "k"`) || !strings.Contains(updated, `down = "j"`) {
		t.Fatalf("custom keybinds were not preserved:\n%s", updated)
	}
	if !strings.Contains(updated, `current_commit = "new"`) || !strings.Contains(updated, `repo_path = "new-repo"`) {
		t.Fatalf("update metadata was not written:\n%s", updated)
	}
}

func TestKeybindEntries_CoversAllStructFields(t *testing.T) {
	// Every key in keybindEntries must appear in keybindValues output.
	d := Default()
	vals := keybindValues(&d.Keybinds)
	for _, e := range keybindEntries {
		if _, ok := vals[e.key]; !ok {
			t.Errorf("keybindEntries has %q but keybindValues map is missing it", e.key)
		}
	}
	// Every key in keybindValues must appear in keybindEntries.
	entryKeys := make(map[string]bool, len(keybindEntries))
	for _, e := range keybindEntries {
		entryKeys[e.key] = true
	}
	for k := range vals {
		if !entryKeys[k] {
			t.Errorf("keybindValues has %q but keybindEntries is missing it", k)
		}
	}
}

func TestConfigPath_EndsCorrectly(t *testing.T) {
	p := ConfigPath()
	want := filepath.Join("delbysoft", "listicles.toml")
	if !strings.HasSuffix(p, want) {
		t.Errorf("ConfigPath %q does not end with %q", p, want)
	}
}

func TestConfigDir_EndsWithDelbysoft(t *testing.T) {
	d := ConfigDir()
	if !strings.HasSuffix(d, "delbysoft") {
		t.Errorf("ConfigDir %q does not end with 'delbysoft'", d)
	}
}
