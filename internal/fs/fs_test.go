package fs

import (
	"os"
	"path/filepath"
	"testing"
)

// ─── HumanSize ────────────────────────────────────────────────────────────────

func TestHumanSize(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}
	for _, c := range cases {
		got := HumanSize(c.in)
		if got != c.want {
			t.Errorf("HumanSize(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ─── ScanDir ─────────────────────────────────────────────────────────────────

func TestScanDir_DirsOnly(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, dir, "beta")
	mustMkdir(t, dir, "alpha")
	mustMkdir(t, dir, "gamma")
	mustFile(t, dir, "file.txt")

	entries, err := ScanDir(dir, false, false)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 dir entries, got %d", len(entries))
	}
	// Sorted alphabetically
	wantNames := []string{"alpha", "beta", "gamma"}
	for i, e := range entries {
		if e.Name != wantNames[i] {
			t.Errorf("entry[%d] name = %q, want %q", i, e.Name, wantNames[i])
		}
		if !e.IsDir() {
			t.Errorf("entry[%d] should be a dir", i)
		}
	}
}

func TestScanDir_DirsAndFiles(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, dir, "subdir")
	mustFile(t, dir, "z_file.txt")
	mustFile(t, dir, "a_file.txt")

	entries, err := ScanDir(dir, false, true)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	// Dirs before files
	if !entries[0].IsDir() {
		t.Error("first entry should be a directory")
	}
	if entries[1].IsDir() || entries[2].IsDir() {
		t.Error("last two entries should be files")
	}
	// Files sorted alphabetically
	if entries[1].Name != "a_file.txt" {
		t.Errorf("entries[1] = %q, want a_file.txt", entries[1].Name)
	}
	if entries[2].Name != "z_file.txt" {
		t.Errorf("entries[2] = %q, want z_file.txt", entries[2].Name)
	}
}

func TestScanDir_HiddenFiles(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, dir, ".hidden")
	mustMkdir(t, dir, "visible")

	// showHidden=false should exclude .hidden
	entries, err := ScanDir(dir, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "visible" {
		t.Errorf("expected only 'visible', got %v", entryNames(entries))
	}

	// showHidden=true should include .hidden
	entries, err = ScanDir(dir, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries with hidden, got %d", len(entries))
	}
}

func TestScanDir_Empty(t *testing.T) {
	dir := t.TempDir()
	entries, err := ScanDir(dir, false, false)
	if err != nil {
		t.Fatalf("ScanDir on empty dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestScanDir_SortOrder(t *testing.T) {
	dir := t.TempDir()
	// Create in reverse alpha order to verify sorting
	mustMkdir(t, dir, "c")
	mustMkdir(t, dir, "A") // uppercase should sort case-insensitively
	mustMkdir(t, dir, "b")

	entries, err := ScanDir(dir, false, false)
	if err != nil {
		t.Fatal(err)
	}
	wantNames := []string{"A", "b", "c"}
	for i, e := range entries {
		if e.Name != wantNames[i] {
			t.Errorf("entry[%d] = %q, want %q", i, e.Name, wantNames[i])
		}
	}
}

func TestScanDir_FileSize(t *testing.T) {
	dir := t.TempDir()
	mustFileContent(t, dir, "data.txt", "hello world")

	entries, err := ScanDir(dir, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Size != 11 {
		t.Errorf("file Size = %d, want 11", entries[0].Size)
	}
}

// ─── DirStats ─────────────────────────────────────────────────────────────────

func TestDirStats(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, dir, "subdir1")
	mustMkdir(t, dir, "subdir2")
	mustFileContent(t, dir, "a.txt", "aa")
	mustFileContent(t, dir, "b.txt", "bb")
	mustFileContent(t, dir, "c.txt", "cc")

	nf, nd, size := DirStats(dir)
	if nf != 3 {
		t.Errorf("numFiles = %d, want 3", nf)
	}
	if nd != 2 {
		t.Errorf("numDirs = %d, want 2", nd)
	}
	if size != 6 {
		t.Errorf("size = %d, want 6", size)
	}
}

// ─── CreateEntry ─────────────────────────────────────────────────────────────

func TestCreateEntry_File(t *testing.T) {
	dir := t.TempDir()
	if err := CreateEntry(dir, "newfile.txt"); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, "newfile.txt"))
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if info.IsDir() {
		t.Error("expected a file, got a directory")
	}
}

func TestCreateEntry_Dir(t *testing.T) {
	dir := t.TempDir()
	if err := CreateEntry(dir, "newdir/"); err != nil {
		t.Fatalf("CreateEntry dir: %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, "newdir"))
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected a directory, got a file")
	}
}

func TestCreateEntry_EmptyName(t *testing.T) {
	dir := t.TempDir()
	if err := CreateEntry(dir, ""); err == nil {
		t.Error("expected error for empty name, got nil")
	}
	if err := CreateEntry(dir, "   "); err == nil {
		t.Error("expected error for whitespace-only name, got nil")
	}
}

// ─── DeleteEntry ─────────────────────────────────────────────────────────────

func TestDeleteEntry_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todelete.txt")
	mustFileContent(t, dir, "todelete.txt", "content")

	if err := DeleteEntry(path); err != nil {
		t.Fatalf("DeleteEntry: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file still exists after DeleteEntry")
	}
}

func TestDeleteEntry_DirRecursive(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "deep")
	mustMkdir(t, dir, "deep")
	mustFileContent(t, subdir, "file.txt", "x")

	if err := DeleteEntry(subdir); err != nil {
		t.Fatalf("DeleteEntry recursive: %v", err)
	}
	if _, err := os.Stat(subdir); !os.IsNotExist(err) {
		t.Error("directory still exists after DeleteEntry")
	}
}

// ─── RenameEntry ─────────────────────────────────────────────────────────────

func TestRenameEntry(t *testing.T) {
	dir := t.TempDir()
	mustFileContent(t, dir, "old.txt", "data")

	if err := RenameEntry(filepath.Join(dir, "old.txt"), "new.txt"); err != nil {
		t.Fatalf("RenameEntry: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "new.txt")); err != nil {
		t.Errorf("new name does not exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "old.txt")); !os.IsNotExist(err) {
		t.Error("old name still exists after rename")
	}
}

func TestRenameEntry_EmptyName(t *testing.T) {
	dir := t.TempDir()
	mustFileContent(t, dir, "file.txt", "x")
	if err := RenameEntry(filepath.Join(dir, "file.txt"), ""); err == nil {
		t.Error("expected error for empty new name, got nil")
	}
}

// ─── ParentDir ────────────────────────────────────────────────────────────────

func TestParentDir_Normal(t *testing.T) {
	got := ParentDir("/a/b/c")
	if got != "/a/b" {
		t.Errorf("ParentDir(/a/b/c) = %q, want /a/b", got)
	}
}

func TestParentDir_Root(t *testing.T) {
	got := ParentDir("/")
	if got != "/" {
		t.Errorf("ParentDir(/) = %q, want /", got)
	}
}

func TestParentDir_SingleLevel(t *testing.T) {
	got := ParentDir("/foo")
	if got != "/" {
		t.Errorf("ParentDir(/foo) = %q, want /", got)
	}
}

// ─── CopyEntry ────────────────────────────────────────────────────────────────

func TestCopyEntry_File(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	mustFileContent(t, src, "source.txt", "hello listicle")

	if err := CopyEntry(filepath.Join(src, "source.txt"), dst); err != nil {
		t.Fatalf("CopyEntry file: %v", err)
	}

	copied := filepath.Join(dst, "source.txt")
	data, err := os.ReadFile(copied)
	if err != nil {
		t.Fatalf("copied file not readable: %v", err)
	}
	if string(data) != "hello listicle" {
		t.Errorf("copied content = %q, want %q", string(data), "hello listicle")
	}
}

func TestCopyEntry_Dir(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Build source tree: src/mydir/sub/file.txt
	mydir := filepath.Join(src, "mydir")
	mustMkdir(t, src, "mydir")
	mustMkdir(t, mydir, "sub")
	mustFileContent(t, filepath.Join(mydir, "sub"), "file.txt", "nested content")

	if err := CopyEntry(mydir, dst); err != nil {
		t.Fatalf("CopyEntry dir: %v", err)
	}

	// Check structure reproduced
	nestedFile := filepath.Join(dst, "mydir", "sub", "file.txt")
	data, err := os.ReadFile(nestedFile)
	if err != nil {
		t.Fatalf("nested file not copied: %v", err)
	}
	if string(data) != "nested content" {
		t.Errorf("nested content = %q, want %q", string(data), "nested content")
	}
}

func TestCopyEntry_PreservesMode(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	path := filepath.Join(src, "exec.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := CopyEntry(path, dst); err != nil {
		t.Fatalf("CopyEntry: %v", err)
	}

	info, err := os.Stat(filepath.Join(dst, "exec.sh"))
	if err != nil {
		t.Fatal(err)
	}
	// Check executable bits preserved
	if info.Mode()&0100 == 0 {
		t.Errorf("execute bit not preserved: mode = %v", info.Mode())
	}
}

// ─── Entry helpers ────────────────────────────────────────────────────────────

func TestEntry_IsDir(t *testing.T) {
	d := Entry{Type: EntryDir}
	f := Entry{Type: EntryFile}
	if !d.IsDir() {
		t.Error("EntryDir.IsDir() should be true")
	}
	if f.IsDir() {
		t.Error("EntryFile.IsDir() should be false")
	}
}

func TestEntry_SizeHuman(t *testing.T) {
	e := Entry{Size: 2048}
	if got := e.SizeHuman(); got != "2.0 KB" {
		t.Errorf("SizeHuman() = %q, want 2.0 KB", got)
	}
}

// ─── Test helpers ─────────────────────────────────────────────────────────────

func mustMkdir(t *testing.T, parent, name string) {
	t.Helper()
	if err := os.Mkdir(filepath.Join(parent, name), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", name, err)
	}
}

func mustFile(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte{}, 0644); err != nil {
		t.Fatalf("create file %s: %v", name, err)
	}
}

func mustFileContent(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("create file %s: %v", name, err)
	}
}

func entryNames(entries []Entry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	return names
}
