package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSelectedFilePreservesBytesAndMissingFileErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "input.txt")
	if err := os.WriteFile(path, []byte("first\nsecond\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := readSelectedFile(path)
	if err != nil || string(raw) != "first\nsecond\n" || countLines(path) != 2 {
		t.Fatal("selected local bytes changed", err)
	}
	missing := filepath.Join(dir, "missing.txt")
	if raw, err := readSelectedFile(missing); err == nil || raw != nil || countLines(missing) != 0 {
		t.Fatal("missing selected file was accepted")
	}
	if raw, err := readSelectedFile(filepath.Join(missing, "input.txt")); err == nil || raw != nil {
		t.Fatal("missing parent directory was accepted")
	}
}

// All files and links are fixed, test-owned temporary DATA. No external file,
// privileged process or network service participates in these unit scenarios.
func TestSelectedFileKeepsLeafLinksWithinSelectedParent(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "selected")
	if err := os.Mkdir(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "input.txt"), []byte("selected"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "outside.txt"), []byte("other owned DATA"), 0o600); err != nil {
		t.Fatal(err)
	}
	inside := filepath.Join(dir, "inside-link")
	if err := os.Symlink("input.txt", inside); err != nil {
		t.Fatal(err)
	}
	if raw, err := readSelectedFile(inside); err != nil || string(raw) != "selected" {
		t.Fatal("ordinary link inside selected parent failed", err)
	}
	outside := filepath.Join(dir, "outside-link")
	if err := os.Symlink("../outside.txt", outside); err != nil {
		t.Fatal(err)
	}
	if raw, err := readSelectedFile(outside); err == nil || raw != nil || countLines(outside) != 0 {
		t.Fatal("leaf link selected bytes outside its parent")
	}
}
