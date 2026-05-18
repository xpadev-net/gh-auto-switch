package ghadapter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xpadev-net/gh-auto-switch/internal/procenv"
)

func TestCheckTokenEnvDetectsEmptyPresence(t *testing.T) {
	clearTokenEnv(t)
	t.Setenv("GH_TOKEN", "")
	if err := CheckTokenEnv(); err == nil {
		t.Fatalf("CheckTokenEnv() succeeded")
	}
}

func TestStatusForParsesActiveAccount(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	writeExe(t, filepath.Join(bin, "gh"), `#!/bin/sh
if [ "$1 $2 $3 $4 $5 $6" = "auth status --hostname github.com --json hosts" ]; then
	  printf '{"hosts":{"github.com":[{"state":"success","active":true,"host":"github.com","login":"user","tokenSource":"keyring"}]}}\n'
  exit 0
fi
exit 1
`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	got, err := StatusFor("github.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.Active != "user" || len(got.Accounts) != 1 {
		t.Fatalf("StatusFor() = %+v", got)
	}
}

func TestStatusForFailsClosedOnMissingActive(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	writeExe(t, filepath.Join(bin, "gh"), `#!/bin/sh
printf '{"hosts":{"github.com":[{"state":"success","active":false,"host":"github.com","login":"user","tokenSource":"keyring"}]}}\n'
`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if _, err := StatusFor("github.com"); err == nil {
		t.Fatalf("StatusFor() succeeded")
	}
}

func TestEnsureAccountRequiresSuccessfulState(t *testing.T) {
	st := Status{Active: "ok", Accounts: []Account{{Login: "bad", State: "failure", TokenSource: "keyring"}, {Login: "ok", State: "success", TokenSource: "keyring"}}}
	if err := EnsureAccount(st, "ok"); err != nil {
		t.Fatalf("EnsureAccount(ok) error = %v", err)
	}
	if err := EnsureAccount(st, "bad"); err == nil {
		t.Fatalf("EnsureAccount(bad) succeeded")
	}
}

func TestStatusForFailsClosedOnMissingTokenSource(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	writeExe(t, filepath.Join(bin, "gh"), `#!/bin/sh
printf '{"hosts":{"github.com":[{"state":"success","active":true,"host":"github.com","login":"user"}]}}\n'
`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if _, err := StatusFor("github.com"); err == nil {
		t.Fatalf("StatusFor() succeeded")
	}
}

func TestEnsureAccountRequiresTokenSource(t *testing.T) {
	st := Status{Active: "user", Accounts: []Account{{Login: "user", State: "success"}}}
	if err := EnsureAccount(st, "user"); err == nil {
		t.Fatalf("EnsureAccount() succeeded")
	}
}

func writeExe(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
}

func clearTokenEnv(t *testing.T) {
	t.Helper()
	for _, name := range procenv.TokenEnvNames {
		old, ok := os.LookupEnv(name)
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func(name, old string, ok bool) func() {
			return func() {
				if ok {
					_ = os.Setenv(name, old)
				} else {
					_ = os.Unsetenv(name)
				}
			}
		}(name, old, ok))
	}
}
