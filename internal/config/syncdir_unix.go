//go:build darwin || linux || freebsd || openbsd || netbsd

package config

import (
	"os"

	"github.com/xpadev-net/gh-auto-switch/internal/apperr"
)

func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return apperr.Wrap(apperr.ConfigInvalid, "could not open config directory", err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return apperr.Wrap(apperr.ConfigInvalid, "could not sync config directory", err)
	}
	return nil
}
