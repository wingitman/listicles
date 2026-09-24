// Package search runs filesystem and content searches, preferring rg/fd/zoxide
// when available and using native Go fallbacks otherwise.
package search

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wingitman/listicles/internal/fs"
)

// Tools caches which fast tools are available.
type Tools struct {
	HasFd     bool
	HasRg     bool
	HasZoxide bool
}

// DetectTools checks for optional search tools on PATH once at startup.
func DetectTools() Tools {
	t := Tools{}
	if _, err := exec.LookPath("fd"); err == nil {
		t.HasFd = true
	}
	if _, err := exec.LookPath("rg"); err == nil {
		t.HasRg = true
	}
	if _, err := exec.LookPath("zoxide"); err == nil {
		t.HasZoxide = true
	}
	return t
}

// Request describes a single search operation.
type Request struct {
	Dir       string // root directory to search from
	Query     string // search term
	Recursive bool   // -r flag: search subdirectories
	TextMode  bool   // -t flag: search file contents instead of names
	Zoxide    bool   // -z flag: search zoxide's known directory database
	Hidden    bool   // include hidden files/dirs
}

// Result is returned for every matched path.
type Result struct {
	Path    string
	Line    string // non-empty for text-in-file matches (the matched line)
	LineNum int    // line number for text matches
}

// Run executes the search and returns results (blocking, suitable for a goroutine).
// On each result the caller-supplied callback is invoked.
// If query is empty, Run returns immediately with no results.
func Run(t Tools, req Request, emit func(Result)) error {
	if strings.TrimSpace(req.Query) == "" {
		return nil
	}

	if req.Zoxide {
		return runZoxideSearch(t, req, emit)
	}
	if req.TextMode {
		return runTextSearch(t, req, emit)
	}
	return runNameSearch(t, req, emit)
}

// ─── Zoxide directory search ─────────────────────────────────────────────────

func runZoxideSearch(t Tools, req Request, emit func(Result)) error {
	if !t.HasZoxide {
		return nil
	}
	args := append([]string{"query", "-l"}, strings.Fields(req.Query)...)
	cmd := exec.Command("zoxide", args...)
	return streamLines(cmd, req.Dir, func(line string) {
		path := strings.TrimSpace(line)
		if path == "" {
			return
		}
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			return
		}
		emit(Result{Path: path})
	})
}

// ─── Name search ─────────────────────────────────────────────────────────────

func runNameSearch(t Tools, req Request, emit func(Result)) error {
	var cmd *exec.Cmd

	if t.HasFd {
		args := []string{}
		if req.Hidden {
			args = append(args, "--hidden")
		}
		if !req.Recursive {
			args = append(args, "--max-depth", "1")
		}
		// fd glob pattern
		args = append(args, "--glob", "*"+req.Query+"*", req.Dir)
		cmd = exec.Command("fd", args...)
	} else {
		return runNativeNameSearch(req, emit)
	}

	return streamLines(cmd, req.Dir, func(line string) {
		line = strings.TrimSpace(line)
		if line == "" || line == req.Dir {
			return
		}
		emit(Result{Path: line})
	})
}

// ─── Text-in-file search ─────────────────────────────────────────────────────

func runTextSearch(t Tools, req Request, emit func(Result)) error {
	var cmd *exec.Cmd

	if t.HasRg {
		args := []string{
			"--json",
			"--line-number",
			"--color", "never",
		}
		if req.Hidden {
			args = append(args, "--hidden")
		}
		if !req.Recursive {
			args = append(args, "--max-depth", "1")
		}
		args = append(args, req.Query, req.Dir)
		cmd = exec.Command("rg", args...)
	} else {
		return runNativeTextSearch(req, emit)
	}

	if t.HasRg {
		return streamLines(cmd, req.Dir, func(raw string) {
			if r, ok := parseRgJSONLine(raw); ok {
				emit(r)
			}
		})
	}

	// grep output: path:line_num:line_content
	return streamLines(cmd, req.Dir, func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		if r, ok := parseTextMatchLine(raw); ok {
			emit(r)
		}
	})
}

// Native fallbacks avoid shell commands, so search works on minimal Windows
// installs where neither PowerShell nor the optional tools are available.
func runNativeNameSearch(req Request, emit func(Result)) error {
	query := strings.ToLower(req.Query)
	return filepath.WalkDir(req.Dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == req.Dir {
			return nil
		}
		rel, _ := filepath.Rel(req.Dir, path)
		if !req.Hidden && hasHiddenComponent(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !req.Recursive && filepath.Dir(rel) != "." {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.Contains(strings.ToLower(d.Name()), query) {
			emit(Result{Path: path})
		}
		return nil
	})
}

func runNativeTextSearch(req Request, emit func(Result)) error {
	return filepath.WalkDir(req.Dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == req.Dir {
			return nil
		}
		rel, _ := filepath.Rel(req.Dir, path)
		if !req.Hidden && hasHiddenComponent(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if !req.Recursive && filepath.Dir(rel) != "." {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			return nil
		}
		if !req.Recursive && filepath.Dir(rel) != "." {
			return nil
		}
		f, openErr := os.Open(path)
		if openErr != nil {
			return nil
		}
		defer f.Close()
		s := bufio.NewScanner(f)
		lineNum := 0
		for s.Scan() {
			lineNum++
			if strings.Contains(s.Text(), req.Query) {
				emit(Result{Path: path, Line: s.Text(), LineNum: lineNum})
			}
		}
		if err := s.Err(); err != nil && err != io.EOF {
			return nil
		}
		return nil
	})
}

func hasHiddenComponent(rel string) bool {
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if strings.HasPrefix(part, ".") && part != "." {
			return true
		}
	}
	return false
}

type rgJSONLine struct {
	Type string `json:"type"`
	Data struct {
		Path struct {
			Text string `json:"text"`
		} `json:"path"`
		LineNumber int `json:"line_number"`
		Lines      struct {
			Text string `json:"text"`
		} `json:"lines"`
	} `json:"data"`
}

func parseRgJSONLine(raw string) (Result, bool) {
	var msg rgJSONLine
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		return Result{}, false
	}
	if msg.Type != "match" || msg.Data.Path.Text == "" {
		return Result{}, false
	}
	return Result{
		Path:    msg.Data.Path.Text,
		Line:    strings.TrimSpace(msg.Data.Lines.Text),
		LineNum: msg.Data.LineNumber,
	}, true
}

func parseTextMatchLine(raw string) (Result, bool) {
	first := strings.IndexByte(raw, ':')
	if first < 0 {
		return Result{}, false
	}
	start := 0
	if len(raw) >= 3 && raw[1] == ':' && (raw[2] == '\\' || raw[2] == '/') {
		start = 2
	}
	lineSepRel := strings.IndexByte(raw[start:], ':')
	if lineSepRel < 0 {
		return Result{}, false
	}
	lineSep := start + lineSepRel
	path := raw[:lineSep]
	rest := raw[lineSep+1:]
	contentSep := strings.IndexByte(rest, ':')
	lineNumText := rest
	lineContent := ""
	if contentSep >= 0 {
		lineNumText = rest[:contentSep]
		lineContent = rest[contentSep+1:]
	}
	lineNum, err := strconv.Atoi(lineNumText)
	if path == "" || err != nil {
		return Result{}, false
	}
	return Result{Path: path, Line: strings.TrimSpace(lineContent), LineNum: lineNum}, true
}

// streamLines runs cmd and calls cb for each output line.
func streamLines(cmd *exec.Cmd, _ string, cb func(string)) error {
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = nil // suppress stderr noise
	if err := cmd.Start(); err != nil {
		return err
	}
	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		cb(scanner.Text())
	}
	_ = cmd.Wait() // non-zero exit is fine (no matches)
	return nil
}

// ResultsToEntries converts name-search results to fs.Entry slices for display.
// For text-search results, use GroupTextResults instead.
func ResultsToEntries(results []Result) []fs.Entry {
	seen := map[string]bool{}
	var entries []fs.Entry
	for _, r := range results {
		if seen[r.Path] {
			continue
		}
		seen[r.Path] = true

		info, err := os.Stat(r.Path)
		entryType := fs.EntryFile
		var size int64
		if err == nil {
			if info.IsDir() {
				entryType = fs.EntryDir
			}
			size = info.Size()
		}

		entries = append(entries, fs.Entry{
			Name: filepath.Base(r.Path),
			Path: r.Path,
			Type: entryType,
			Size: size,
		})
	}
	return entries
}

// GroupedTextResult groups all text-search matches for a single file.
type GroupedTextResult struct {
	Path         string
	Entry        fs.Entry
	TotalMatches int
	Matches      []Result // all matching lines, sorted by LineNum
}

// GroupTextResults groups text-search results by file path and returns them
// sorted by the order in which files first appeared in results.
func GroupTextResults(results []Result) []GroupedTextResult {
	order := []string{}
	byPath := map[string]*GroupedTextResult{}

	for _, r := range results {
		if _, ok := byPath[r.Path]; !ok {
			info, err := os.Stat(r.Path)
			entryType := fs.EntryFile
			var size int64
			if err == nil {
				if info.IsDir() {
					entryType = fs.EntryDir
				}
				size = info.Size()
			}
			g := &GroupedTextResult{
				Path: r.Path,
				Entry: fs.Entry{
					Name: filepath.Base(r.Path),
					Path: r.Path,
					Type: entryType,
					Size: size,
				},
			}
			byPath[r.Path] = g
			order = append(order, r.Path)
		}
		g := byPath[r.Path]
		g.Matches = append(g.Matches, r)
		g.TotalMatches++
	}

	out := make([]GroupedTextResult, 0, len(order))
	for _, p := range order {
		out = append(out, *byPath[p])
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	b := []byte{}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
