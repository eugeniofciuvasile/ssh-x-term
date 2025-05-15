package config

import (
	"os/user"
	"strings"
)

// ExpandPath expands ~ to the user's home directory.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		usr, err := user.Current()
		if err == nil {
			return strings.Replace(path, "~", usr.HomeDir, 1)
		}
	}
	return path
}
