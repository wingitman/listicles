package search

import (
	"os"
	"path/filepath"
	"testing"
)

// ─── ResultsToEntries ─────────────────────────────────────────────────────────

func TestResultsToEntries_EmptyInput(t *testing.T) {
	entries := ResultsToEntries(nil)
	if entries != nil && len(entries) != 0 {
		t.Errorf("expected empty/nil, got %d entries", len(entries))
	}
}

func TestResultsToEntries_FileResult(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	results := []Result{{Path: path}}
	entries := ResultsToEntries(results)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "test.txt" {
		t.Errorf("Name = %q, want test.txt", entries[0].Name)
	}
	if entries[0].Path != path {
		t.Errorf("Path = %q, want %q", entries[0].Path, path)
	}
	if entries[0].IsDir() {
		t.Error("file result should not be a dir")
	}
}

func TestResultsToEntries_DirResult(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	results := []Result{{Path: subdir}}
	entries := ResultsToEntries(results)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if !entries[0].IsDir() {
		t.Error("directory result should be a dir")
	}
}

func TestResultsToEntries_DeduplicatesSamePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	results := []Result{
		{Path: path},
		{Path: path}, // duplicate
	}
	entries := ResultsToEntries(results)
	if len(entries) != 1 {
		t.Errorf("expected 1 deduplicated entry, got %d", len(entries))
	}
}

func TestResultsToEntries_TextMatchLineNum(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "source.go")
	if err := os.WriteFile(path, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	results := []Result{
		{Path: path, Line: "func main() {}", LineNum: 2},
	}
	entries := ResultsToEntries(results)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	// Name should contain the line number
	if entries[0].Name != "source.go:2" {
		t.Errorf("Name = %q, want source.go:2", entries[0].Name)
	}
}

func TestResultsToEntries_TextMatchMultipleLines_NoDedupAcrossLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.go")
	if err := os.WriteFile(path, []byte("a\nb\nc\n"), 0644); err != nil {
		t.Fatal(err)
	}

	results := []Result{
		{Path: path, Line: "a", LineNum: 1},
		{Path: path, Line: "b", LineNum: 2},
		{Path: path, Line: "c", LineNum: 3},
	}
	entries := ResultsToEntries(results)
	// Different line numbers = different entries
	if len(entries) != 3 {
		t.Errorf("expected 3 entries for 3 line matches, got %d", len(entries))
	}
}

// ─── DetectTools ─────────────────────────────────────────────────────────────

func TestDetectTools_DoesNotPanic(t *testing.T) {
	tools := DetectTools()
	// We can't assert specific values since fd/rg may or may not be installed
	// Just verify the struct is populated without panicking
	_ = tools.HasFd
	_ = tools.HasRg
	t.Logf("fd available: %v, rg available: %v", tools.HasFd, tools.HasRg)
}

// ─── Run (name search, non-recursive) ────────────────────────────────────────

func TestRun_NameSearch_FindsMatch(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "documents"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "downloads"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "pictures"), 0755); err != nil {
		t.Fatal(err)
	}

	tools := DetectTools()
	req := Request{
		Dir:       dir,
		Query:     "do",
		Recursive: false,
		TextMode:  false,
	}

	var results []Result
	if err := Run(tools, req, func(r Result) {
		results = append(results, r)
	}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Should find "documents" and "downloads" but not "pictures"
	if len(results) < 2 {
		t.Errorf("expected at least 2 results for 'do', got %d: %v", len(results), resultPaths(results))
	}
	for _, r := range results {
		base := filepath.Base(r.Path)
		if base != "documents" && base != "downloads" {
			t.Errorf("unexpected result: %q", r.Path)
		}
	}
}

func TestRun_EmptyQuery_NoResults(t *testing.T) {
	dir := t.TempDir()
	tools := DetectTools()
	req := Request{Dir: dir, Query: "", Recursive: false}

	var count int
	_ = Run(tools, req, func(r Result) { count++ })
	if count != 0 {
		t.Errorf("empty query should yield 0 results, got %d", count)
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func resultPaths(results []Result) []string {
	paths := make([]string, len(results))
	for i, r := range results {
		paths[i] = r.Path
	}
	return paths
}
