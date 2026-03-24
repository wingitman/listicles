// Package search runs filesystem and content searches, preferring rg/fd when
// available and falling back to POSIX find/grep.
package search

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/listicle/listicle/internal/fs"
)

// Tools caches which fast tools are available.
type Tools struct {
	HasFd bool
	HasRg bool
}

// DetectTools checks for fd and rg on PATH once at startup.
func DetectTools() Tools {
	t := Tools{}
	if _, err := exec.LookPath("fd"); err == nil {
		t.HasFd = true
	}
	if _, err := exec.LookPath("rg"); err == nil {
		t.HasRg = true
	}
	return t
}

// Request describes a single search operation.
type Request struct {
	Dir       string // root directory to search from
	Query     string // search term
	Recursive bool   // -r flag: search subdirectories
	TextMode  bool   // -t flag: search file contents instead of names
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

	if req.TextMode {
		return runTextSearch(t, req, emit)
	}
	return runNameSearch(t, req, emit)
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
		// POSIX find fallback
		args := []string{req.Dir}
		if !req.Recursive {
			args = append(args, "-maxdepth", "1")
		}
		args = append(args, "-iname", "*"+req.Query+"*")
		if !req.Hidden {
			// Exclude hidden entries
			args = append([]string{req.Dir, "-not", "-path", "*/.*"}, args[1:]...)
		}
		cmd = exec.Command("find", args...)
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
			"--line-number",
			"--no-heading",
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
		// grep fallback
		args := []string{"-rn", "--include=*"}
		if !req.Recursive {
			// grep has no maxdepth; use find+grep pipeline via shell
			// For simplicity, use -r on the dir but limit with find piped in
			args = []string{"-n"}
			// We'll just grep non-recursively against files in dir
			entries, _ := os.ReadDir(req.Dir)
			var files []string
			for _, e := range entries {
				if !e.IsDir() {
					files = append(files, filepath.Join(req.Dir, e.Name()))
				}
			}
			if len(files) == 0 {
				return nil
			}
			args = append(args, req.Query)
			args = append(args, files...)
			cmd = exec.Command("grep", args...)
		} else {
			args = append(args, req.Query, req.Dir)
			cmd = exec.Command("grep", args...)
		}
	}

	// rg/grep output: path:line_num:line_content
	return streamLines(cmd, req.Dir, func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		// Parse "path:linenum:content"
		parts := strings.SplitN(raw, ":", 3)
		if len(parts) < 2 {
			return
		}
		path := parts[0]
		lineContent := ""
		lineNum := 0
		if len(parts) >= 3 {
			lineContent = strings.TrimSpace(parts[2])
			fmt.Sscanf(parts[1], "%d", &lineNum)
		}
		emit(Result{Path: path, Line: lineContent, LineNum: lineNum})
	})
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

// ResultsToEntries converts search results to fs.Entry slices for display.
func ResultsToEntries(results []Result) []fs.Entry {
	seen := map[string]bool{}
	var entries []fs.Entry
	for _, r := range results {
		key := r.Path
		if r.LineNum > 0 {
			// For text matches, show each match line as a separate entry
			// but use a synthetic key to avoid dedup
			key = r.Path + ":" + itoa(r.LineNum)
		}
		if seen[key] {
			continue
		}
		seen[key] = true

		info, err := os.Stat(r.Path)
		entryType := fs.EntryFile
		var size int64
		if err == nil {
			if info.IsDir() {
				entryType = fs.EntryDir
			}
			size = info.Size()
		}

		name := filepath.Base(r.Path)
		if r.LineNum > 0 {
			// Show line number in name for text matches
			name = filepath.Base(r.Path) + ":" + itoa(r.LineNum)
		}

		entries = append(entries, fs.Entry{
			Name: name,
			Path: r.Path,
			Type: entryType,
			Size: size,
		})
	}
	return entries
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
