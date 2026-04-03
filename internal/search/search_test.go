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

func TestResultsToEntries_TextMatchDeduplicatesByFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "source.go")
	if err := os.WriteFile(path, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Multiple line matches for the same file — ResultsToEntries deduplicates
	// by file path now; use GroupTextResults to get per-line grouping.
	results := []Result{
		{Path: path, Line: "func main() {}", LineNum: 2},
		{Path: path, Line: "package main", LineNum: 1},
	}
	entries := ResultsToEntries(results)
	if len(entries) != 1 {
		t.Fatalf("expected 1 deduplicated entry, got %d", len(entries))
	}
	if entries[0].Name != "source.go" {
		t.Errorf("Name = %q, want source.go", entries[0].Name)
	}
}

func TestGroupTextResults_GroupsByFile(t *testing.T) {
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
	groups := GroupTextResults(results)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].TotalMatches != 3 {
		t.Errorf("TotalMatches = %d, want 3", groups[0].TotalMatches)
	}
	if groups[0].Entry.Name != "file.go" {
		t.Errorf("Entry.Name = %q, want file.go", groups[0].Entry.Name)
	}
}

func TestGroupTextResults_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.go")
	pathB := filepath.Join(dir, "b.go")
	for _, p := range []string{pathA, pathB} {
		if err := os.WriteFile(p, []byte("match\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	results := []Result{
		{Path: pathA, Line: "match", LineNum: 1},
		{Path: pathB, Line: "match", LineNum: 1},
		{Path: pathA, Line: "match2", LineNum: 2},
	}
	groups := GroupTextResults(results)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	// Order should reflect first appearance: a.go then b.go
	if groups[0].Entry.Name != "a.go" {
		t.Errorf("groups[0].Name = %q, want a.go", groups[0].Entry.Name)
	}
	if groups[0].TotalMatches != 2 {
		t.Errorf("groups[0].TotalMatches = %d, want 2", groups[0].TotalMatches)
	}
	if groups[1].Entry.Name != "b.go" {
		t.Errorf("groups[1].Name = %q, want b.go", groups[1].Entry.Name)
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
