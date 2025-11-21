//go:build windows
// +build windows

// File: sftp_windows.go
package ssh

import (
	"strconv"

	"github.com/pkg/sftp"
)

func getOwnerGroup(sys interface{}) (string, string) {
	if stat, ok := sys.(*sftp.FileStat); ok {
		return strconv.Itoa(int(stat.UID)), strconv.Itoa(int(stat.GID))
	}
	return "-", "-"
}
