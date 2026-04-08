// Package state handles persistence of recents and bookmarks to state.json
// in the same config directory as listicles.toml.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const maxRecents = 100

// RecentEntry records a file that was opened.
type RecentEntry struct {
	Path       string    `json:"path"`
	RootDir    string    `json:"root_dir"`
	AccessedAt time.Time `json:"accessed_at"`
}

// BookmarkEntry is a manually pinned path.
type BookmarkEntry struct {
	Path    string    `json:"path"`
	RootDir string    `json:"root_dir"`
	Name    string    `json:"name"` // custom label; defaults to basename when empty
	AddedAt time.Time `json:"added_at"`
}

// State is the root persistence struct written to state.json.
type State struct {
	Recents   []RecentEntry   `json:"recents"`
	Bookmarks []BookmarkEntry `json:"bookmarks"`
}

// statePath returns the path to state.json given the config directory.
func statePath(configDir string) string {
	return filepath.Join(configDir, "state.json")
}

// Load reads state.json from configDir. Returns an empty State when the file
// does not exist (first run) and an error only on parse failure.
func Load(configDir string) (*State, error) {
	s := &State{}
	data, err := os.ReadFile(statePath(configDir))
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		// Corrupt state — start fresh rather than crashing.
		return &State{}, nil
	}
	return s, nil
}

// Save atomically writes state to configDir/state.json.
func Save(configDir string, s *State) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := statePath(configDir) + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, statePath(configDir))
}

// ─── Recents ──────────────────────────────────────────────────────────────────

// AddRecent prepends path to s.Recents, deduplicates by path, and caps at
// maxRecents. rootDir should be the project root (git root or cwd).
func AddRecent(s *State, path, rootDir string) {
	// Remove any existing entry for this path.
	filtered := s.Recents[:0]
	for _, r := range s.Recents {
		if r.Path != path {
			filtered = append(filtered, r)
		}
	}
	// Prepend the new entry.
	entry := RecentEntry{
		Path:       path,
		RootDir:    rootDir,
		AccessedAt: time.Now(),
	}
	s.Recents = append([]RecentEntry{entry}, filtered...)
	// Cap length.
	if len(s.Recents) > maxRecents {
		s.Recents = s.Recents[:maxRecents]
	}
}

// RemoveRecent removes the entry with the given path from s.Recents.
func RemoveRecent(s *State, path string) {
	out := s.Recents[:0]
	for _, r := range s.Recents {
		if r.Path != path {
			out = append(out, r)
		}
	}
	s.Recents = out
}

// RecentsForRoot returns recents whose RootDir matches rootDir.
func RecentsForRoot(s *State, rootDir string) []RecentEntry {
	var out []RecentEntry
	for _, r := range s.Recents {
		if r.RootDir == rootDir {
			out = append(out, r)
		}
	}
	return out
}

// ─── Bookmarks ────────────────────────────────────────────────────────────────

// AddBookmark appends a bookmark for path. If a bookmark for path already
// exists it is moved to the end and its name updated.
func AddBookmark(s *State, path, rootDir, name string) {
	// Remove existing entry for this path.
	out := s.Bookmarks[:0]
	for _, b := range s.Bookmarks {
		if b.Path != path {
			out = append(out, b)
		}
	}
	s.Bookmarks = append(out, BookmarkEntry{
		Path:    path,
		RootDir: rootDir,
		Name:    name,
		AddedAt: time.Now(),
	})
}

// RemoveBookmark removes the bookmark with the given path.
func RemoveBookmark(s *State, path string) {
	out := s.Bookmarks[:0]
	for _, b := range s.Bookmarks {
		if b.Path != path {
			out = append(out, b)
		}
	}
	s.Bookmarks = out
}

// RenameBookmark sets a custom display name for the bookmark at path.
func RenameBookmark(s *State, path, newName string) {
	for i := range s.Bookmarks {
		if s.Bookmarks[i].Path == path {
			s.Bookmarks[i].Name = newName
			return
		}
	}
}

// BookmarksForRoot returns bookmarks whose RootDir matches rootDir.
func BookmarksForRoot(s *State, rootDir string) []BookmarkEntry {
	var out []BookmarkEntry
	for _, b := range s.Bookmarks {
		if b.RootDir == rootDir {
			out = append(out, b)
		}
	}
	return out
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// FormatTimeAgo returns a human-readable relative time string.
func FormatTimeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
	case d < 48*time.Hour:
		return "yesterday"
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}
