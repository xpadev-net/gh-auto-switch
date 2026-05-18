//go:build !(darwin || linux || freebsd || openbsd || netbsd)

package config

import "os"

func ownerInvalid(info os.FileInfo) bool {
	return false
}
