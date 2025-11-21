package sshutil

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// scanSSHKeys scans ~/.ssh and returns a sorted list of candidate private key paths.
// It excludes files ending with .pub and common SSH config files.
func ScanSSHKeys() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	sshDir := filepath.Join(home, ".ssh")

	var keys []string

	// Basic exclude set
	exclude := map[string]struct{}{
		"known_hosts":     {},
		"known_hosts.old": {},
		"authorized_keys": {},
		"config":          {},
		"README":          {},
	}

	_ = filepath.WalkDir(sshDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// ignore permissions errors etc.
			return nil
		}

		// skip directories (we only want files)
		if d.IsDir() {
			// don't traverse into subdirectories, only top-level files
			if path != sshDir {
				return filepath.SkipDir
			}
			return nil
		}

		base := filepath.Base(path)
		if _, ok := exclude[base]; ok {
			return nil
		}
		// exclude public keys
		if strings.HasSuffix(base, ".pub") {
			return nil
		}
		// Only include regular files
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		// Append as candidate with ~/.ssh/ prefix instead of full path
		relativePath := "~/.ssh/" + filepath.Base(path)
		keys = append(keys, relativePath)
		return nil
	})

	// sort for consistent order
	sort.Strings(keys)
	return keys
}
