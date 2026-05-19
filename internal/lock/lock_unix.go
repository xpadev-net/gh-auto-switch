//go:build darwin || linux

package lock

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/xpadev-net/gh-auto-switch/internal/apperr"
)

type Lock struct {
	file *os.File
}

func AcquireAuthStore(timeout time.Duration) (*Lock, error) {
	key, err := authStoreKey()
	if err != nil {
		return nil, err
	}
	return acquire(key, timeout)
}

func acquire(key string, timeout time.Duration) (*Lock, error) {
	dir := baseDir()
	if _, err := os.Lstat(dir); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, apperr.Wrap(apperr.InternalError, "could not stat lock directory", err)
		}
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, apperr.Wrap(apperr.InternalError, "could not create lock directory", err)
		}
		if err := os.Chmod(dir, 0700); err != nil {
			return nil, apperr.Wrap(apperr.InternalError, "could not secure lock directory", err)
		}
	}
	if err := validateDir(dir); err != nil {
		return nil, err
	}
	name := key + ".lock"
	f, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, apperr.Wrap(apperr.InternalError, "could not open lock file", err)
	}
	deadline := time.Now().Add(timeout)
	for {
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return &Lock{file: f}, nil
		}
		if time.Now().After(deadline) {
			_ = f.Close()
			return nil, apperr.New(apperr.LockTimeout, "timed out acquiring auth store lock; another gh-auto-switch command may still be running")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (l *Lock) Release() {
	if l == nil || l.file == nil {
		return
	}
	_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	_ = l.file.Close()
}

func baseDir() string {
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		return filepath.Join(d, "ghautoswitch")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("ghautoswitch-%d", os.Getuid()))
}

func authStoreKey() (string, error) {
	dir := os.Getenv("GH_CONFIG_DIR")
	if dir == "" {
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			dir = filepath.Join(xdg, "gh")
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", apperr.Wrap(apperr.InternalError, "could not resolve gh config directory", err)
			}
			dir = filepath.Join(home, ".config", "gh")
		}
	}
	// Match gh behavior: a relative GH_CONFIG_DIR is interpreted from each
	// process's current working directory, so lock scope follows that semantics.
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", apperr.Wrap(apperr.InternalError, "could not resolve gh config directory", err)
	}
	// If the config directory does not exist yet, keep the absolute path so all
	// pre-creation callers contend on the same key. A later creation through a
	// symlinked parent can canonicalize differently, but failing before gh creates
	// its config directory would be a worse startup behavior.
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", apperr.Wrap(apperr.InternalError, "could not resolve gh config directory", err)
	}
	if err == nil {
		abs = resolved
	}
	sum := sha256.Sum256([]byte(abs))
	return "authstore-" + hex.EncodeToString(sum[:]), nil
}

func validateDir(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return apperr.Wrap(apperr.InternalError, "could not stat lock directory", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return apperr.New(apperr.InternalError, "lock directory is insecure")
	}
	if st, ok := info.Sys().(*syscall.Stat_t); ok && int(st.Uid) != os.Getuid() {
		return apperr.New(apperr.InternalError, "lock directory owner is invalid")
	}
	return nil
}
