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
	if l, err := Acquire("github.com", time.Millisecond); err == nil {
		l.Release()
		t.Fatalf("Acquire() succeeded")
	}
}
