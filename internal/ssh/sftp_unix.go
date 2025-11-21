//go:build !windows
// +build !windows

// File: sftp_unix.go
package ssh

import (
	"strconv"
	"syscall"
)

func getOwnerGroup(sys interface{}) (string, string) {
	if stat, ok := sys.(*syscall.Stat_t); ok {
		return strconv.Itoa(int(stat.Uid)), strconv.Itoa(int(stat.Gid))
	}
	return "-", "-"
}
