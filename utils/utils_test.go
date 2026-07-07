package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileExists(t *testing.T) {
	dir := t.TempDir()

	file := filepath.Join(dir, "exists.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !FileExists(file) {
		t.Errorf("FileExists(%q) = false, want true", file)
	}
	if FileExists(filepath.Join(dir, "missing.txt")) {
		t.Error("FileExists on missing file = true, want false")
	}
	// a directory is not a file
	if FileExists(dir) {
		t.Error("FileExists on directory = true, want false")
	}
	// regression: a stat error other than not-exist must not panic
	if FileExists(filepath.Join(file, "impossible-child")) {
		t.Error("FileExists on path under a file = true, want false")
	}
}
