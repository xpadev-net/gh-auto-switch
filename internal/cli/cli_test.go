package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveJSONMasksTokenLikeURLUser(t *testing.T) {
	tmp := t.TempDir()
	writeConfig(t, tmp, `
version: 1
rules:
  - name: by-user
    host: github.com
    url_user: ghp_secret
    account: login
`)
	bin := filepath.Join(tmp, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	writeExe(t, filepath.Join(bin, "git"), `#!/bin/sh
if [ "$1 $2" = "rev-parse --is-inside-work-tree" ]; then exit 0; fi
if [ "$1 $2 $3" = "remote get-url origin" ]; then echo "https://ghp_secret@github.com/org/repo.git"; exit 0; fi
exit 1
`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	var out, errOut bytes.Buffer
	code := Run([]string{"--json", "resolve"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	if !strings.Contains(out.String(), `"url_user":"***"`) {
		t.Fatalf("token-like url_user not masked: %s", out.String())
	}
	if strings.Contains(out.String(), "ghp_secret") {
		t.Fatalf("raw token-like url_user leaked: %s", out.String())
	}
}

func TestPrintEnvAllowsUnmatchedNoop(t *testing.T) {
	tmp := t.TempDir()
	writeConfig(t, tmp, `
version: 1
defaults:
  on_unmatched: noop
rules:
  - name: other
    host: github.com
    owner: other
    account: user
`)
	bin := filepath.Join(tmp, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	writeExe(t, filepath.Join(bin, "git"), `#!/bin/sh
if [ "$1 $2" = "rev-parse --is-inside-work-tree" ]; then exit 0; fi
if [ "$1 $2 $3" = "remote get-url origin" ]; then echo "git@github.com:org/repo.git"; exit 0; fi
exit 1
`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	var out, errOut bytes.Buffer
	code := Run([]string{"print-env"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	if out.String() != "export GH_HOST='github.com'\n" {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestSwitchFailsWhenTokenEnvPresentBeforeUnmatched(t *testing.T) {
	tmp := t.TempDir()
	writeConfig(t, tmp, `
version: 1
defaults:
  on_unmatched: noop
rules:
  - name: other
    host: github.com
    owner: other
    account: user
`)
	t.Setenv("GH_TOKEN", "x")
	var out, errOut bytes.Buffer
	code := Run([]string{"--json", "switch"}, &out, &errOut)
	if code != 3 {
		t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	if !strings.Contains(out.String(), `"code":"token_env_present"`) {
		t.Fatalf("stdout = %s", out.String())
	}
}

func TestExecPreservesChildJSONFlag(t *testing.T) {
	tmp := t.TempDir()
	writeConfig(t, tmp, `
version: 1
defaults:
  on_unmatched: noop
rules:
  - name: other
    host: github.com
    owner: other
    account: user
`)
	bin := filepath.Join(tmp, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	writeExe(t, filepath.Join(bin, "git"), `#!/bin/sh
if [ "$1 $2" = "rev-parse --is-inside-work-tree" ]; then exit 0; fi
if [ "$1 $2 $3" = "remote get-url origin" ]; then echo "git@github.com:org/repo.git"; exit 0; fi
exit 1
`)
	writeExe(t, filepath.Join(bin, "child"), `#!/bin/sh
printf '%s\n' "$@"
`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	var out, errOut bytes.Buffer
	code := Run([]string{"exec", "--allow-unmatched", "--", "child", "--json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	if out.String() != "--json\n" {
		t.Fatalf("child args not preserved: %q", out.String())
	}
}

func writeConfig(t *testing.T, tmp, content string) {
	t.Helper()
	path := filepath.Join(tmp, "config.yml")
	if err := os.WriteFile(path, []byte(strings.TrimLeft(content, "\n")), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GHAUTOSWITCH_CONFIG", path)
	for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN"} {
		t.Setenv(name, "")
	}
}

func writeExe(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
}
