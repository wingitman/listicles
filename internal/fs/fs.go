package fs

import (
	"bufio"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"runtime"
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
	Ignored  bool // true if matched by a .gitignore pattern
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

// FileModTime returns the modification time of path as a formatted string.
func FileModTime(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "N/A"
	}
	return info.ModTime().Format("Jan 02 2006 15:04")
}

// FileBirthTime returns the creation/birth time of path as a formatted string.
// The implementation is platform-specific (see birthtime_*.go files).
// On Linux, birth time is not reliably available so "N/A (see modified)" is returned.

// FilePermissions returns a human-readable permission string for path,
// e.g. "rwxr-xr-- (754)". Returns "N/A" on Windows or on error.
func FilePermissions(path string) string {
	if runtime.GOOS == "windows" {
		return "N/A"
	}
	info, err := os.Stat(path)
	if err != nil {
		return "N/A"
	}
	perm := info.Mode().Perm()
	sym := permSymbolic(perm)
	oct := fmt.Sprintf("%o", perm)
	return fmt.Sprintf("%s (%s)", sym, oct)
}

// permSymbolic converts a 9-bit os.FileMode into a symbolic rwx string like "rwxr-xr--".
func permSymbolic(mode os.FileMode) string {
	const rwx = "rwxrwxrwx"
	buf := []byte("---------")
	for i := 0; i < 9; i++ {
		if mode&(1<<uint(8-i)) != 0 {
			buf[i] = rwx[i]
		}
	}
	return string(buf)
}

// FileMimeType returns a short, friendly description of the file type based on
// its extension (e.g. "Go source", "PNG image", "JSON data").
// Directories always return "Directory".
func FileMimeType(path string, isDir bool) string {
	if isDir {
		return "Directory"
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return "Unknown"
	}
	// Check friendly name overrides first.
	if label, ok := friendlyMimeLabels[ext]; ok {
		return label
	}
	// Fall back to stdlib mime package.
	mtype := mime.TypeByExtension(ext)
	if mtype == "" {
		return "Unknown"
	}
	// Strip parameters (e.g. "; charset=utf-8").
	if i := strings.Index(mtype, ";"); i != -1 {
		mtype = strings.TrimSpace(mtype[:i])
	}
	return mtype
}

// friendlyMimeLabels maps common file extensions to short human-readable labels.
var friendlyMimeLabels = map[string]string{
	// Source code
	".go":    "Go source",
	".py":    "Python source",
	".js":    "JavaScript",
	".ts":    "TypeScript",
	".jsx":   "JSX",
	".tsx":   "TSX",
	".rs":    "Rust source",
	".c":     "C source",
	".cpp":   "C++ source",
	".cc":    "C++ source",
	".h":     "C/C++ header",
	".hpp":   "C++ header",
	".java":  "Java source",
	".kt":    "Kotlin source",
	".swift": "Swift source",
	".rb":    "Ruby source",
	".php":   "PHP source",
	".cs":    "C# source",
	".lua":   "Lua source",
	".sh":    "Shell script",
	".bash":  "Bash script",
	".zsh":   "Zsh script",
	".fish":  "Fish script",
	".pl":    "Perl script",
	".r":     "R source",
	// Data / config
	".json": "JSON data",
	".yaml": "YAML data",
	".yml":  "YAML data",
	".toml": "TOML config",
	".xml":  "XML data",
	".csv":  "CSV data",
	".sql":  "SQL script",
	".env":  "Env config",
	".ini":  "INI config",
	".conf": "Config file",
	".lock": "Lock file",
	// Documents
	".md":   "Markdown",
	".txt":  "Plain text",
	".rst":  "reStructuredText",
	".pdf":  "PDF document",
	".doc":  "Word document",
	".docx": "Word document",
	".xls":  "Excel spreadsheet",
	".xlsx": "Excel spreadsheet",
	// Images — raster
	".png":  "PNG image",
	".jpg":  "JPEG image",
	".jpeg": "JPEG image",
	".gif":  "GIF image",
	".webp": "WebP image",
	".ico":  "Icon image",
	".bmp":  "BMP image",
	".tiff": "TIFF image",
	".tif":  "TIFF image",
	// Images — vector / compressed vector
	".svg":  "SVG image",
	".svgz": "SVG image (compressed)",
	// Images — Apple HEIF
	".heic": "HEIC image",
	".heif": "HEIF image",
	// Images — design / pixel art
	".psd":      "Photoshop document",
	".psb":      "Photoshop document (large)",
	".xcf":      "GIMP document",
	".ase":      "Aseprite sprite",
	".aseprite": "Aseprite sprite",
	// Archives
	".zip": "ZIP archive",
	".tar": "TAR archive",
	".gz":  "Gzip archive",
	".bz2": "Bzip2 archive",
	".xz":  "XZ archive",
	".7z":  "7-Zip archive",
	".rar": "RAR archive",
	// Media
	".mp3":  "MP3 audio",
	".wav":  "WAV audio",
	".flac": "FLAC audio",
	".ogg":  "OGG audio",
	".mp4":  "MP4 video",
	".mkv":  "MKV video",
	".mov":  "MOV video",
	".avi":  "AVI video",
	// Web
	".html": "HTML file",
	".htm":  "HTML file",
	".css":  "CSS stylesheet",
	// Executables / binaries
	".exe":   "Windows executable",
	".so":    "Shared library",
	".a":     "Static library",
	".dylib": "Dynamic library",
	// Other
	".log": "Log file",
	".tmp": "Temp file",
	".bak": "Backup file",
	".pem": "PEM certificate",
	".key": "Key file",
}

// imageExtensions is the set of file extensions that can be decoded and
// rendered as images in the preview panel.
var imageExtensions = map[string]bool{
	// Standard raster
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".webp": true,
	".bmp":  true,
	".tiff": true,
	".tif":  true,
	// Vector
	".svg":  true,
	".svgz": true,
	// Apple / HEIF
	".heic": true,
	".heif": true,
	// Photoshop
	".psd": true,
	".psb": true,
	// Aseprite pixel art
	".ase":      true,
	".aseprite": true,
}

// IsImageFile reports whether the file at path has an image extension that the
// preview panel can render (PNG, JPEG, GIF, WebP, BMP, ICO, TIFF).
func IsImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return imageExtensions[ext]
}

// ScanDir lists entries in dirPath according to showHidden and showFiles flags.
// gitignorePatterns is a slice of patterns from .gitignore; matching entries
// have their Ignored field set to true but are still included in the result so
// they can be rendered dimmed when show_hidden is on.
func ScanDir(dirPath string, showHidden bool, showFiles bool, gitignorePatterns []string) ([]Entry, error) {
	if IsDriveListRoot(dirPath) {
		return ListDrives()
	}

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
		ignored := matchesGitignore(name, fullPath, gitignorePatterns)

		// Skip gitignored entries unless show_hidden is on (same logic as
		// hidden files — they are only visible when the user opts in).
		if ignored && !showHidden {
			continue
		}

		if de.IsDir() {
			result = append(result, Entry{
				Name:    name,
				Path:    fullPath,
				Type:    EntryDir,
				Ignored: ignored,
			})
		} else if showFiles {
			info, err := de.Info()
			var size int64
			if err == nil {
				size = info.Size()
			}
			result = append(result, Entry{
				Name:    name,
				Path:    fullPath,
				Type:    EntryFile,
				Size:    size,
				Ignored: ignored,
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

// DriveListRoot is a virtual root used on Windows to list all available drives.
const DriveListRoot = "::drives::"

func IsDriveListRoot(path string) bool {
	return path == DriveListRoot
}

func ListDrives() ([]Entry, error) {
	mask, err := logicalDrivesMask()
	if err != nil {
		return nil, err
	}
	return DrivesFromMask(mask), nil
}

func DrivesFromMask(mask uint32) []Entry {
	entries := []Entry{}
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		letter := string(rune('A' + i))
		path := letter + ":" + string(filepath.Separator)
		entries = append(entries, Entry{
			Name: letter + ":",
			Path: path,
			Type: EntryDir,
		})
	}
	return entries
}

func IsWindowsDriveRoot(path string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	cleaned := filepath.Clean(path)
	vol := filepath.VolumeName(cleaned)
	if len(vol) != 2 || vol[1] != ':' {
		return false
	}
	rest := strings.TrimPrefix(cleaned, vol)
	return rest == string(filepath.Separator) || rest == ""
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
	if IsDriveListRoot(path) {
		return path
	}
	if IsWindowsDriveRoot(path) {
		return DriveListRoot
	}
	parent := filepath.Dir(path)
	if parent == path {
		return path
	}
	return parent
}

// CopyEntry recursively copies src into dstDir, creating dstDir/basename(src).
// Works for both files and directories.
func CopyEntry(src, dstDir string) error {
	return CopyEntryAs(src, dstDir, filepath.Base(src))
}

// CopyEntryAs recursively copies src into dstDir with the given newName,
// creating dstDir/newName. Works for both files and directories.
func CopyEntryAs(src, dstDir, newName string) error {
	dst := filepath.Join(dstDir, newName)
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

// ─── Git helpers ─────────────────────────────────────────────────────────────

// FindGitRoot walks up from path looking for a directory containing ".git".
// Returns the git root path, or "" if not inside a git repository.
func FindGitRoot(path string) string {
	cur := path
	for {
		if _, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return ""
		}
		cur = parent
	}
}

// ReadGitignorePatterns reads <gitRoot>/.gitignore and returns the non-comment,
// non-empty lines as patterns. Returns nil when the file doesn't exist.
func ReadGitignorePatterns(gitRoot string) []string {
	if gitRoot == "" {
		return nil
	}
	f, err := os.Open(filepath.Join(gitRoot, ".gitignore"))
	if err != nil {
		return nil
	}
	defer f.Close()

	var patterns []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

// AppendGitignore appends relPath as a new line to <gitRoot>/.gitignore,
// creating the file if it does not exist. relPath should be relative to gitRoot.
func AppendGitignore(gitRoot, relPath string) error {
	p := filepath.Join(gitRoot, ".gitignore")
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	// Ensure there is a trailing newline before our entry.
	info, err := f.Stat()
	if err == nil && info.Size() > 0 {
		// Read last byte to check for newline.
		buf := make([]byte, 1)
		rf, rerr := os.Open(p)
		if rerr == nil {
			if _, serr := rf.Seek(-1, 2); serr == nil {
				rf.Read(buf) //nolint:errcheck
			}
			rf.Close()
		}
		if buf[0] != '\n' {
			fmt.Fprintln(f)
		}
	}
	_, err = fmt.Fprintln(f, relPath)
	return err
}

// matchesGitignore reports whether name or fullPath matches any of the given
// gitignore patterns. This implements a lightweight subset of gitignore rules:
//   - exact name match (e.g. "node_modules")
//   - trailing-slash dir pattern (e.g. "dist/") matched by name
//   - leading-slash anchored patterns (e.g. "/vendor") matched by name
//   - simple glob via filepath.Match on the base name
func matchesGitignore(name, fullPath string, patterns []string) bool {
	for _, pat := range patterns {
		if pat == "" {
			continue
		}
		// Negation patterns (!) are not supported in this lightweight impl.
		if strings.HasPrefix(pat, "!") {
			continue
		}
		// Strip trailing slash (marks directory patterns, but we match both).
		clean := strings.TrimSuffix(pat, "/")
		// Strip leading slash (anchors to repo root — we match by name only).
		clean = strings.TrimPrefix(clean, "/")
		if clean == "" {
			continue
		}
		// Exact name match.
		if clean == name {
			return true
		}
		// Glob match on base name.
		if matched, err := filepath.Match(clean, name); err == nil && matched {
			return true
		}
		// Glob match on full path (for patterns containing path separators).
		if strings.Contains(clean, "/") {
			if matched, err := filepath.Match(clean, fullPath); err == nil && matched {
				return true
			}
		}
	}
	return false
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
