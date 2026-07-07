package utils

import (
	"errors"
	"io/fs"
	"os"
)

// FileExists reports whether filename exists and is a regular file (not a directory).
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false
		}
		// Stat failed for another reason (e.g. permission denied);
		// the file cannot be used either way.
		return false
	}
	return !info.IsDir()
}
