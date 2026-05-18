//go:build darwin || linux || freebsd || openbsd || netbsd

package config

import (
	"os"
	"syscall"
)

func ownerInvalid(info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(st.Uid) != os.Getuid()
}
