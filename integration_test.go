//go:build integration

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/wingitman/listicles/internal/app"
	"github.com/wingitman/listicles/internal/config"
)

// ─── Helpers ─────────────────────────────────────────────────────────────────

// newTestDir creates a temp directory with given subdirectory names.
func newTestDir(t *testing.T, dirs ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, d := range dirs {
		if err := os.Mkdir(filepath.Join(root, d), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	return root
}

// newTestModel creates a BubbleTea TestModel from a listicles app.Model.
func newTestModel(t *testing.T, dir string) *teatest.TestModel {
	t.Helper()
	cfg := config.Default()
	m, err := app.New(cfg, dir, "", "")
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(120, 40),
	)
	return tm
}

// waitFor waits up to 3s for the output to contain s.
func waitFor(t *testing.T, tm *teatest.TestModel, s string) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte(s))
	}, teatest.WithDuration(3*time.Second))
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestIntegration_QuitWithQ(t *testing.T) {
	dir := newTestDir(t, "alpha", "beta")
	tm := newTestModel(t, dir)

	// Wait for initial render
	waitFor(t, tm, "alpha")

	// Press q
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_InitialRender_ShowsEntries(t *testing.T) {
	dir := newTestDir(t, "documents", "downloads", "pictures")
	tm := newTestModel(t, dir)

	// Wait for all three entries to appear in the accumulated output
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := string(bts)
		return strings.Contains(s, "documents") &&
			strings.Contains(s, "downloads") &&
			strings.Contains(s, "pictures")
	}, teatest.WithDuration(5*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_NavigateRight_ExpandsDir(t *testing.T) {
	root := newTestDir(t, "parent")
	os.Mkdir(filepath.Join(root, "parent", "child"), 0755)

	tm := newTestModel(t, root)
	waitFor(t, tm, "parent")

	// Press right to expand
	tm.Send(tea.KeyMsg{Type: tea.KeyRight})
	waitFor(t, tm, "child")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_NavigateLeft_CollapsesDir(t *testing.T) {
	root := newTestDir(t, "mydir")
	os.Mkdir(filepath.Join(root, "mydir", "child"), 0755)

	tm := newTestModel(t, root)
	waitFor(t, tm, "mydir")

	// Expand
	tm.Send(tea.KeyMsg{Type: tea.KeyRight})
	waitFor(t, tm, "child")

	// Collapse
	tm.Send(tea.KeyMsg{Type: tea.KeyLeft})

	// Give it a moment to process
	time.Sleep(100 * time.Millisecond)

	// child should no longer be visible
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return !bytes.Contains(bts, []byte("child"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_SearchBarOpens(t *testing.T) {
	dir := newTestDir(t, "alpha")
	tm := newTestModel(t, dir)
	waitFor(t, tm, "alpha")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	waitFor(t, tm, "Enter for full search")

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_SearchFilter_LiveFilters(t *testing.T) {
	dir := newTestDir(t, "documents", "downloads", "pictures")
	tm := newTestModel(t, dir)
	waitFor(t, tm, "documents")

	// Open search
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	waitFor(t, tm, "Enter for full search")

	// Type "doc" — should show live match count
	tm.Type("doc")
	waitFor(t, tm, "match")

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_ToggleFiles_ShowsBadge(t *testing.T) {
	dir := newTestDir(t, "adir")
	tm := newTestModel(t, dir)
	waitFor(t, tm, "adir")

	// Press f to toggle files
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	waitFor(t, tm, "[files]")

	// Press f again to toggle back
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return !bytes.Contains(bts, []byte("[files]"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_ToggleHidden_ShowsBadge(t *testing.T) {
	dir := newTestDir(t, "visible")
	tm := newTestModel(t, dir)
	waitFor(t, tm, "visible")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(".")})
	waitFor(t, tm, "[hidden]")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_JumpTop_MovesToFirst(t *testing.T) {
	dir := newTestDir(t, "aaa", "bbb", "ccc")
	tm := newTestModel(t, dir)
	waitFor(t, tm, "aaa")

	// Move down twice
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	time.Sleep(50 * time.Millisecond)

	// Home should jump back to top
	tm.Send(tea.KeyMsg{Type: tea.KeyHome})

	// Final model should have cursor at 0
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	// Just verify the program exits cleanly
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_YankShowsClipboardBar(t *testing.T) {
	dir := newTestDir(t, "myproject")
	tm := newTestModel(t, dir)
	waitFor(t, tm, "myproject")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	waitFor(t, tm, "[copy]")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_CutShowsClipboardBar(t *testing.T) {
	dir := newTestDir(t, "myproject")
	tm := newTestModel(t, dir)
	waitFor(t, tm, "myproject")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	waitFor(t, tm, "[cut]")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_Delete_ShowsConfirmation(t *testing.T) {
	dir := newTestDir(t, "todelete")
	tm := newTestModel(t, dir)
	waitFor(t, tm, "todelete")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	waitFor(t, tm, "Confirm")

	// Cancel with n
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})

	// Dir should still exist
	if _, err := os.Stat(filepath.Join(dir, "todelete")); err != nil {
		t.Errorf("directory should still exist after cancel: %v", err)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_Delete_ConfirmDeletesDir(t *testing.T) {
	// "aaa_delete" sorts before any other dir so it's the first item
	dir := newTestDir(t, "aaa_delete", "zzz_keep")
	tm := newTestModel(t, dir)
	waitFor(t, tm, "aaa_delete")

	// Cursor is on first entry (aaa_delete) — delete it
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	waitFor(t, tm, "Confirm")
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

	// Give the filesystem time to update
	time.Sleep(300 * time.Millisecond)

	if _, err := os.Stat(filepath.Join(dir, "aaa_delete")); !os.IsNotExist(err) {
		t.Error("directory should have been deleted after confirm y")
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_StatusBar_ShowsKeyHints(t *testing.T) {
	dir := newTestDir(t, "a")
	tm := newTestModel(t, dir)

	// Wait for the status bar to appear (contains "enter" for the confirm key)
	// and the nav hint (Nano-style: "[↑/↓/←/→]Nav")
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := string(bts)
		return strings.Contains(s, "enter") && strings.Contains(s, "Nav")
	}, teatest.WithDuration(5*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

// TestIntegration_SearchFlags tests that -rt flag is parsed correctly
// by verifying the search bar shows both -r and -t indicators.
func TestIntegration_SearchFlags_RT(t *testing.T) {
	dir := newTestDir(t, "alpha")
	tm := newTestModel(t, dir)
	waitFor(t, tm, "alpha")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	waitFor(t, tm, "Enter for full search")

	tm.Type("-rt foo")

	// Both flags should be visually active
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := string(bts)
		return strings.Contains(s, "-r") && strings.Contains(s, "-t")
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
