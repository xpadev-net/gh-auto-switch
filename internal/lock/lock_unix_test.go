//go:build darwin || linux

package lock

import (
	"os"
	"path/filepath"
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
