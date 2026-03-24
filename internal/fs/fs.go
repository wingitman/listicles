package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// EntryType distinguishes files from directories.
type EntryType int

const (
	EntryDir  EntryType = iota
	EntryFile EntryType = iota
)

// Entry represents a single filesystem entry.
type Entry struct {
	Name     string
	Path     string
	Type     EntryType
	Size     int64
	NumFiles int
	NumDirs  int
}

// IsDir returns true if this entry is a directory.
func (e Entry) IsDir() bool {
	return e.Type == EntryDir
}

// SizeHuman returns a human-readable size string.
func (e Entry) SizeHuman() string {
	return HumanSize(e.Size)
}

// HumanSize formats bytes into a human-readable string.
func HumanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// ScanDir lists entries in dirPath according to showHidden and showFiles flags.
func ScanDir(dirPath string, showHidden bool, showFiles bool) ([]Entry, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var result []Entry
	for _, de := range entries {
		name := de.Name()

		// Skip hidden unless enabled
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}

		fullPath := filepath.Join(dirPath, name)

		if de.IsDir() {
			entry := Entry{
				Name: name,
				Path: fullPath,
				Type: EntryDir,
			}
			result = append(result, entry)
		} else if showFiles {
			info, err := de.Info()
			var size int64
			if err == nil {
				size = info.Size()
			}
			result = append(result, Entry{
				Name: name,
				Path: fullPath,
				Type: EntryFile,
				Size: size,
			})
		}
	}

	// Sort: dirs first, then files; each group alphabetically
	sort.Slice(result, func(i, j int) bool {
		if result[i].Type != result[j].Type {
			return result[i].Type < result[j].Type // dirs (0) before files (1)
		}
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	return result, nil
}

// DirStats returns file count, folder count, and total size for dirPath (non-recursive top-level).
func DirStats(dirPath string) (numFiles int, numDirs int, size int64) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			numDirs++
		} else {
			numFiles++
			if info, err := e.Info(); err == nil {
				size += info.Size()
			}
		}
	}
	return
}

// CreateEntry creates a file or directory at path.
// If name ends with "/" it creates a directory, otherwise a file.
func CreateEntry(dir string, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	fullPath := filepath.Join(dir, name)
	if strings.HasSuffix(name, "/") {
		return os.MkdirAll(strings.TrimSuffix(fullPath, "/"), 0755)
	}
	// Ensure parent exists (in case user typed subpath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	return f.Close()
}

// DeleteEntry removes a file or directory (recursively).
func DeleteEntry(path string) error {
	return os.RemoveAll(path)
}

// RenameEntry renames oldPath to newName within the same directory.
func RenameEntry(oldPath string, newName string) error {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return fmt.Errorf("name cannot be empty")
	}
	newPath := filepath.Join(filepath.Dir(oldPath), newName)
	return os.Rename(oldPath, newPath)
}

// ParentDir returns the parent directory of path.
// Returns path unchanged if already at root.
func ParentDir(path string) string {
	parent := filepath.Dir(path)
	if parent == path {
		return path
	}
	return parent
}

// CopyEntry recursively copies src into dstDir, creating dstDir/basename(src).
// Works for both files and directories.
func CopyEntry(src, dstDir string) error {
	dst := filepath.Join(dstDir, filepath.Base(src))
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst, info.Mode())
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, info.Mode())
	})
}
