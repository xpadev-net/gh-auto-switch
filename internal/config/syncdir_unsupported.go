//go:build !(darwin || linux || freebsd || openbsd || netbsd)

package config

func syncDir(path string) error {
	return nil
}
