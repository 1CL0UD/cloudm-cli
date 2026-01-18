package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// CleanupDumps removes dump files from a migration directory
func CleanupDumps(migrationDir string) error {
	structure, data := GetDumpPaths(migrationDir)
	backup := GetBackupPath(migrationDir)

	files := []string{structure, data, backup}

	for _, file := range files {
		if FileExists(file) {
			if err := os.Remove(file); err != nil {
				return fmt.Errorf("failed to remove %s: %w", file, err)
			}
		}
	}

	return nil
}

// KeepLatestN keeps only the N most recent migration directories
func KeepLatestN(baseDir string, n int) error {
	if n <= 0 {
		return nil
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Filter only directories
	var dirs []os.DirEntry
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry)
		}
	}

	// Sort by name (which includes timestamp) descending
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].Name() > dirs[j].Name()
	})

	// Remove directories beyond the limit
	if len(dirs) > n {
		for _, dir := range dirs[n:] {
			path := filepath.Join(baseDir, dir.Name())
			if err := os.RemoveAll(path); err != nil {
				return fmt.Errorf("failed to remove old migration directory %s: %w", path, err)
			}
		}
	}

	return nil
}

// RemoveDir removes a directory and all its contents
func RemoveDir(path string) error {
	return os.RemoveAll(path)
}