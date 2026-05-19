//go:build darwin || linux

package lock

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAcquireRejectsPreexistingInsecureDirectory(t *testing.T) {
	tmp := t.TempDir()
	runtimeDir := filepath.Join(tmp, "runtime")
	lockDir := filepath.Join(runtimeDir, "ghautoswitch")
	if err := os.MkdirAll(lockDir, 0777); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(lockDir, 0777); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	if l, err := AcquireAuthStore(time.Millisecond); err == nil {
		l.Release()
		t.Fatalf("AcquireAuthStore() succeeded")
	}
}

func TestAcquireAuthStoreSerializesSameGHConfigDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(tmp, "runtime"))
	t.Setenv("GH_CONFIG_DIR", filepath.Join(tmp, "gh-config"))
	l, err := AcquireAuthStore(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Release()
	if l2, err := AcquireAuthStore(time.Millisecond); err == nil {
		l2.Release()
		t.Fatalf("AcquireAuthStore() succeeded while same auth store was locked")
	} else if !strings.Contains(err.Error(), "another gh-auto-switch command may still be running") {
		t.Fatalf("AcquireAuthStore() error = %v", err)
	}
}

func TestAcquireAuthStoreAllowsDifferentGHConfigDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(tmp, "runtime"))
	t.Setenv("GH_CONFIG_DIR", filepath.Join(tmp, "gh-config-a"))
	l, err := AcquireAuthStore(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Release()
	t.Setenv("GH_CONFIG_DIR", filepath.Join(tmp, "gh-config-b"))
	l2, err := AcquireAuthStore(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	l2.Release()
}

func TestAcquireAuthStoreUsesXDGConfigHomeDefault(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(tmp, "runtime"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "xdg"))
	t.Setenv("GH_CONFIG_DIR", "")
	l, err := AcquireAuthStore(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Release()
	t.Setenv("GH_CONFIG_DIR", filepath.Join(tmp, "xdg", "gh"))
	if l2, err := AcquireAuthStore(time.Millisecond); err == nil {
		l2.Release()
		t.Fatalf("AcquireAuthStore() used a different key for XDG default and explicit GH_CONFIG_DIR")
	}
}

func TestAcquireAuthStoreResolvesGHConfigDirSymlink(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(tmp, "runtime"))
	realDir := filepath.Join(tmp, "real-gh")
	linkDir := filepath.Join(tmp, "link-gh")
	if err := os.Mkdir(realDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GH_CONFIG_DIR", realDir)
	l, err := AcquireAuthStore(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Release()
	t.Setenv("GH_CONFIG_DIR", linkDir)
	if l2, err := AcquireAuthStore(time.Millisecond); err == nil {
		l2.Release()
		t.Fatalf("AcquireAuthStore() used a different key for a symlinked GH_CONFIG_DIR")
	}
}
