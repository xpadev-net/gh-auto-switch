package cli

import (
	"bytes"
	"encoding/json"
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
if [ "$#" -eq 2 ] && [ "$1 $2" = "rev-parse --is-inside-work-tree" ]; then exit 0; fi
if [ "$#" -eq 4 ] && [ "$1 $2 $3 $4" = "remote get-url -- origin" ]; then echo "https://ghp_secret@github.com/org/repo.git"; exit 0; fi
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
if [ "$#" -eq 2 ] && [ "$1 $2" = "rev-parse --is-inside-work-tree" ]; then exit 0; fi
if [ "$#" -eq 4 ] && [ "$1 $2 $3 $4" = "remote get-url -- origin" ]; then echo "git@github.com:org/repo.git"; exit 0; fi
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

func TestSwitchFailsWhenTokenEnvPresentEvenIfEmpty(t *testing.T) {
	tmp := t.TempDir()
	writeConfig(t, tmp, `
version: 1
rules:
  - name: by-owner
    host: github.com
    owner: org
    account: user
`)
	t.Setenv("GH_TOKEN", "")
	var out, errOut bytes.Buffer
	code := Run([]string{"--json", "switch"}, &out, &errOut)
	if code != 3 {
		t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	if !strings.Contains(out.String(), `"code":"token_env_present"`) {
		t.Fatalf("stdout = %s", out.String())
	}
}

func TestSwitchUsesDefaultRuleOutsideGitRepository(t *testing.T) {
	tmp := t.TempDir()
	writeConfig(t, tmp, `
version: 1
rules:
  - name: github-default
    host: github.com
    default: true
    account: target
  - name: by-owner
    host: github.com
    owner: org
    account: org-user
`)
	bin := filepath.Join(tmp, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	writeExe(t, filepath.Join(bin, "git"), `#!/bin/sh
if [ "$#" -eq 2 ] && [ "$1 $2" = "rev-parse --is-inside-work-tree" ]; then exit 1; fi
exit 1
`)
	writeExe(t, filepath.Join(bin, "gh"), `#!/bin/sh
if [ "$#" -eq 6 ] && [ "$1 $2 $3 $4 $5 $6" = "auth status --hostname github.com --json hosts" ]; then
  printf '{"hosts":{"github.com":[{"state":"success","active":true,"host":"github.com","login":"target","tokenSource":"keyring"}]}}\n'
  exit 0
fi
exit 1
`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	var out, errOut bytes.Buffer
	code := Run([]string{"--json", "switch"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["host"] != "github.com" || got["account"] != "target" || got["rule"] != "github-default" {
		t.Fatalf("unexpected JSON result: %#v", got)
	}
	if got["owner"] != nil || got["repo"] != nil || got["matched"] != true || got["action"] != "none" {
		t.Fatalf("unexpected JSON result: %#v", got)
	}
}

func TestSwitchOutsideGitRepositoryFailsWithoutDefaultRule(t *testing.T) {
	tmp := t.TempDir()
	writeConfig(t, tmp, `
version: 1
rules:
  - name: by-owner
    host: github.com
    owner: org
    account: target
`)
	bin := filepath.Join(tmp, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	writeExe(t, filepath.Join(bin, "git"), `#!/bin/sh
if [ "$#" -eq 2 ] && [ "$1 $2" = "rev-parse --is-inside-work-tree" ]; then exit 1; fi
exit 1
`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	var out, errOut bytes.Buffer
	code := Run([]string{"--json", "switch"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	if !strings.Contains(out.String(), `"code":"not_git_repository"`) {
		t.Fatalf("stdout = %s", out.String())
	}
	if !strings.Contains(out.String(), "no default rule configured") {
		t.Fatalf("stdout missing default-rule message: %s", out.String())
	}
}

func TestExecUsesDefaultRuleOutsideGitRepository(t *testing.T) {
	tmp := t.TempDir()
	writeConfig(t, tmp, `
version: 1
rules:
  - name: github-default
    host: github.com
    default: true
    account: target
  - name: by-owner
    host: github.com
    owner: org
    account: org-user
`)
	bin := filepath.Join(tmp, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	writeExe(t, filepath.Join(bin, "git"), `#!/bin/sh
if [ "$#" -eq 2 ] && [ "$1 $2" = "rev-parse --is-inside-work-tree" ]; then exit 1; fi
exit 1
`)
	writeExe(t, filepath.Join(bin, "gh"), `#!/bin/sh
if [ "$#" -eq 6 ] && [ "$1 $2 $3 $4 $5 $6" = "auth status --hostname github.com --json hosts" ]; then
  printf '{"hosts":{"github.com":[{"state":"success","active":true,"host":"github.com","login":"target","tokenSource":"keyring"}]}}\n'
  exit 0
fi
if [ "$1 $2" = "api user" ]; then
  if [ "$GH_HOST" != "github.com" ]; then exit 41; fi
  exit 9
fi
exit 1
`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	var out, errOut bytes.Buffer
	code := Run([]string{"exec", "--", "gh", "api", "user"}, &out, &errOut)
	if code != 9 {
		t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
}

func TestPrintEnvJSONHonorsUnmatchedError(t *testing.T) {
	tmp := t.TempDir()
	writeConfig(t, tmp, `
version: 1
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
if [ "$#" -eq 2 ] && [ "$1 $2" = "rev-parse --is-inside-work-tree" ]; then exit 0; fi
if [ "$#" -eq 4 ] && [ "$1 $2 $3 $4" = "remote get-url -- origin" ]; then echo "git@github.com:org/repo.git"; exit 0; fi
exit 1
`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	var out, errOut bytes.Buffer
	code := Run([]string{"--json", "print-env"}, &out, &errOut)
	if code != 4 {
		t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	if !strings.Contains(out.String(), `"code":"unmatched_rule"`) {
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
if [ "$#" -eq 2 ] && [ "$1 $2" = "rev-parse --is-inside-work-tree" ]; then exit 0; fi
if [ "$#" -eq 4 ] && [ "$1 $2 $3 $4" = "remote get-url -- origin" ]; then echo "git@github.com:org/repo.git"; exit 0; fi
exit 1
`)
	writeExe(t, filepath.Join(bin, "child"), `#!/bin/sh
printf '%s\n' "$@"
`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	var out, errOut bytes.Buffer
	code := Run([]string{"exec", "--allow-unmatched-if-noop", "--", "child", "--json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	if out.String() != "--json\n" {
		t.Fatalf("child args not preserved: %q", out.String())
	}
}

func TestExecAllowUnmatchedPreservesChildJSONFlag(t *testing.T) {
	tmp := t.TempDir()
	writeConfig(t, tmp, `
version: 1
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
if [ "$#" -eq 2 ] && [ "$1 $2" = "rev-parse --is-inside-work-tree" ]; then exit 0; fi
if [ "$#" -eq 4 ] && [ "$1 $2 $3 $4" = "remote get-url -- origin" ]; then echo "git@github.com:org/repo.git"; exit 0; fi
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

func TestSwitchCallsGHAuthSwitchWithHostAndUser(t *testing.T) {
	tmp := t.TempDir()
	writeConfig(t, tmp, `
version: 1
rules:
  - name: by-owner
    host: github.com
    owner: org
    account: target
`)
	bin := filepath.Join(tmp, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	writeExe(t, filepath.Join(bin, "git"), `#!/bin/sh
if [ "$#" -eq 2 ] && [ "$1 $2" = "rev-parse --is-inside-work-tree" ]; then exit 0; fi
if [ "$#" -eq 4 ] && [ "$1 $2 $3 $4" = "remote get-url -- origin" ]; then echo "git@github.com:org/repo.git"; exit 0; fi
exit 1
`)
	state := filepath.Join(tmp, "state")
	if err := os.WriteFile(state, []byte("current"), 0600); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(tmp, "gh.log")
	writeExe(t, filepath.Join(bin, "gh"), `#!/bin/sh
state="`+state+`"
log="`+logPath+`"
printf '%s|GH_HOST=%s\n' "$*" "$GH_HOST" >> "$log"
if [ "$#" -eq 6 ] && [ "$1 $2 $3 $4 $5 $6" = "auth status --hostname github.com --json hosts" ]; then
  active="$(cat "$state")"
  printf '{"hosts":{"github.com":[{"state":"success","active":%s,"host":"github.com","login":"current","tokenSource":"keyring"},{"state":"success","active":%s,"host":"github.com","login":"target","tokenSource":"keyring"}]}}\n' "$([ "$active" = current ] && echo true || echo false)" "$([ "$active" = target ] && echo true || echo false)"
  exit 0
fi
if [ "$1 $2 $3 $4 $5 $6" = "auth switch --hostname github.com --user target" ]; then
  echo target > "$state"
  exit 0
fi
exit 1
`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	var out, errOut bytes.Buffer
	code := Run([]string{"--json", "switch"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	if !strings.Contains(out.String(), `"action":"switched"`) {
		t.Fatalf("stdout = %s", out.String())
	}
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	log := string(logData)
	if !strings.Contains(log, "auth switch --hostname github.com --user target|GH_HOST=github.com") {
		t.Fatalf("gh switch invocation not logged correctly:\n%s", log)
	}
}

func TestMatchedExecPropagatesChildExitAndGHHost(t *testing.T) {
	tmp := t.TempDir()
	writeConfig(t, tmp, `
version: 1
rules:
  - name: by-owner
    host: github.com
    owner: org
    account: target
`)
	bin := filepath.Join(tmp, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	writeExe(t, filepath.Join(bin, "git"), `#!/bin/sh
if [ "$#" -eq 2 ] && [ "$1 $2" = "rev-parse --is-inside-work-tree" ]; then exit 0; fi
if [ "$#" -eq 4 ] && [ "$1 $2 $3 $4" = "remote get-url -- origin" ]; then echo "git@github.com:org/repo.git"; exit 0; fi
exit 1
`)
	writeExe(t, filepath.Join(bin, "gh"), `#!/bin/sh
if [ "$#" -eq 6 ] && [ "$1 $2 $3 $4 $5 $6" = "auth status --hostname github.com --json hosts" ]; then
  printf '{"hosts":{"github.com":[{"state":"success","active":true,"host":"github.com","login":"target","tokenSource":"keyring"}]}}\n'
  exit 0
fi
exit 1
`)
	writeExe(t, filepath.Join(bin, "child"), `#!/bin/sh
if [ "$GH_HOST" != "github.com" ]; then exit 41; fi
exit 7
`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	var out, errOut bytes.Buffer
	code := Run([]string{"exec", "--", "child"}, &out, &errOut)
	if code != 7 {
		t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
}

func TestExecTokenEnvTakesPrecedenceOverEmptyCommand(t *testing.T) {
	tmp := t.TempDir()
	writeConfig(t, tmp, `
version: 1
rules:
  - name: by-owner
    host: github.com
    owner: org
    account: target
`)
	t.Setenv("GH_TOKEN", "x")
	var out, errOut bytes.Buffer
	code := Run([]string{"--json", "exec", "--"}, &out, &errOut)
	if code != 3 {
		t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	if !strings.Contains(out.String(), `"code":"token_env_present"`) {
		t.Fatalf("stdout = %s", out.String())
	}
}

func TestRemoteOverrideAppliesToResolveSwitchExecAndPrintEnv(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "resolve", args: []string{"--json", "resolve", "--remote", "upstream"}},
		{name: "switch", args: []string{"--json", "switch", "--remote", "upstream"}},
		{name: "exec", args: []string{"exec", "--remote", "upstream", "--", "child"}},
		{name: "print-env", args: []string{"print-env", "--remote", "upstream"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			writeConfig(t, tmp, `
version: 1
rules:
  - name: by-owner
    host: github.com
    owner: org
    account: target
`)
			bin := filepath.Join(tmp, "bin")
			if err := os.Mkdir(bin, 0755); err != nil {
				t.Fatal(err)
			}
			gitLog := filepath.Join(tmp, "git.log")
			writeExe(t, filepath.Join(bin, "git"), `#!/bin/sh
printf '%s\n' "$*" >> "`+gitLog+`"
if [ "$#" -eq 2 ] && [ "$1 $2" = "rev-parse --is-inside-work-tree" ]; then exit 0; fi
if [ "$#" -eq 4 ] && [ "$1 $2 $3 $4" = "remote get-url -- upstream" ]; then echo "git@github.com:org/repo.git"; exit 0; fi
exit 1
`)
			writeExe(t, filepath.Join(bin, "gh"), `#!/bin/sh
if [ "$#" -eq 6 ] && [ "$1 $2 $3 $4 $5 $6" = "auth status --hostname github.com --json hosts" ]; then
  printf '{"hosts":{"github.com":[{"state":"success","active":true,"host":"github.com","login":"target","tokenSource":"keyring"}]}}\n'
  exit 0
fi
exit 1
`)
			writeExe(t, filepath.Join(bin, "child"), `#!/bin/sh
exit 0
`)
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			var out, errOut bytes.Buffer
			code := Run(tc.args, &out, &errOut)
			if code != 0 {
				t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
			}
			logData, err := os.ReadFile(gitLog)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(logData), "remote get-url -- upstream\n") {
				t.Fatalf("upstream remote not used; git log:\n%s", string(logData))
			}
		})
	}
}

func TestInvalidRemoteNameRejected(t *testing.T) {
	tmp := t.TempDir()
	writeConfig(t, tmp, `
version: 1
rules:
  - name: by-owner
    host: github.com
    owner: org
    account: target
`)
	var out, errOut bytes.Buffer
	code := Run([]string{"--json", "resolve", "--remote", "-x"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	if !strings.Contains(out.String(), `"code":"invalid_arguments"`) {
		t.Fatalf("stdout = %s", out.String())
	}
}

func TestExecUnmatchedWithoutAllowDoesNotRunChild(t *testing.T) {
	tmp := t.TempDir()
	writeConfig(t, tmp, `
version: 1
rules:
  - name: other
    host: github.com
    owner: other
    account: target
`)
	bin := filepath.Join(tmp, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	childRan := filepath.Join(tmp, "child-ran")
	writeExe(t, filepath.Join(bin, "git"), `#!/bin/sh
if [ "$#" -eq 2 ] && [ "$1 $2" = "rev-parse --is-inside-work-tree" ]; then exit 0; fi
if [ "$#" -eq 4 ] && [ "$1 $2 $3 $4" = "remote get-url -- origin" ]; then echo "git@github.com:org/repo.git"; exit 0; fi
exit 1
`)
	writeExe(t, filepath.Join(bin, "child"), `#!/bin/sh
touch "`+childRan+`"
exit 0
`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	var out, errOut bytes.Buffer
	code := Run([]string{"--json", "exec", "--", "child"}, &out, &errOut)
	if code != 4 {
		t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	if _, err := os.Stat(childRan); !os.IsNotExist(err) {
		t.Fatalf("child should not have run, stat err=%v", err)
	}
}

func TestExecUnmatchedNoopWithoutAllowDoesNotRunChild(t *testing.T) {
	tmp := t.TempDir()
	writeConfig(t, tmp, `
version: 1
defaults:
  on_unmatched: noop
rules:
  - name: other
    host: github.com
    owner: other
    account: target
`)
	bin := filepath.Join(tmp, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	childRan := filepath.Join(tmp, "child-ran")
	writeExe(t, filepath.Join(bin, "git"), `#!/bin/sh
if [ "$#" -eq 2 ] && [ "$1 $2" = "rev-parse --is-inside-work-tree" ]; then exit 0; fi
if [ "$#" -eq 4 ] && [ "$1 $2 $3 $4" = "remote get-url -- origin" ]; then echo "git@github.com:org/repo.git"; exit 0; fi
exit 1
`)
	writeExe(t, filepath.Join(bin, "child"), `#!/bin/sh
touch "`+childRan+`"
exit 0
`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	var out, errOut bytes.Buffer
	code := Run([]string{"--json", "exec", "--", "child"}, &out, &errOut)
	if code != 4 {
		t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	if _, err := os.Stat(childRan); !os.IsNotExist(err) {
		t.Fatalf("child should not have run, stat err=%v", err)
	}
}

func TestArgumentErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		code int
	}{
		{name: "missing subcommand", args: nil, code: 1},
		{name: "unknown subcommand", args: []string{"bogus"}, code: 1},
		{name: "unsupported option", args: []string{"resolve", "--bogus"}, code: 1},
		{name: "exec missing delimiter", args: []string{"exec", "child"}, code: 1},
		{name: "exec empty command", args: []string{"exec", "--"}, code: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clearTokenEnv(t)
			var out, errOut bytes.Buffer
			code := Run(append([]string{"--json"}, tc.args...), &out, &errOut)
			if code != tc.code {
				t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
			}
			if !strings.Contains(out.String(), `"error"`) {
				t.Fatalf("stdout missing JSON error: %s", out.String())
			}
		})
	}
}

func TestInstallPrintsShellSnippets(t *testing.T) {
	tests := []struct {
		name     string
		shell    string
		contains []string
	}{
		{
			name:  "bash",
			shell: "bash",
			contains: []string{
				"# >>> gh-auto-switch hook >>>",
				"gh-auto-switch exec --allow-unmatched-if-noop -- gh \"$@\"",
			},
		},
		{
			name:  "zsh",
			shell: "zsh",
			contains: []string{
				"# >>> gh-auto-switch hook >>>",
				"gh-auto-switch exec --allow-unmatched-if-noop -- gh \"$@\"",
			},
		},
		{
			name:  "fish",
			shell: "fish",
			contains: []string{
				"# >>> gh-auto-switch hook >>>",
				"gh-auto-switch exec --allow-unmatched-if-noop -- gh $argv",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			code := Run([]string{"install", "--print", "--shell", tc.shell}, &out, &errOut)
			if code != 0 {
				t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
			}
			for _, want := range tc.contains {
				if !strings.Contains(out.String(), want) {
					t.Fatalf("stdout missing %q:\n%s", want, out.String())
				}
			}
		})
	}
}

func TestInstallWritesAndReplacesManagedBlock(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	rc := filepath.Join(tmp, ".zshrc")
	if err := os.WriteFile(rc, []byte("before\n# >>> gh-auto-switch hook >>>\nold\n# <<< gh-auto-switch hook <<<\nafter\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	code := Run([]string{"install", "--shell", "zsh"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	if out.String() != "installed: "+rc+"\n" {
		t.Fatalf("stdout = %q", out.String())
	}
	data, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if strings.Count(got, "# >>> gh-auto-switch hook >>>") != 1 {
		t.Fatalf("managed block duplicated:\n%s", got)
	}
	if strings.Contains(got, "\nold\n") {
		t.Fatalf("old managed block was not replaced:\n%s", got)
	}
	if !strings.Contains(got, "before\n") || !strings.Contains(got, "after\n") {
		t.Fatalf("unmanaged content not preserved:\n%s", got)
	}
}

func TestInstallDetectsShellFromEnvAndCreatesFishConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("SHELL", "/opt/homebrew/bin/fish")
	var out, errOut bytes.Buffer
	code := Run([]string{"install"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	path := filepath.Join(tmp, ".config", "fish", "config.fish")
	if out.String() != "installed: "+path+"\n" {
		t.Fatalf("stdout = %q", out.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "gh-auto-switch exec --allow-unmatched-if-noop -- gh $argv") {
		t.Fatalf("fish hook not written:\n%s", string(data))
	}
}

func TestInstallRejectsUnsupportedShell(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Run([]string{"--json", "install", "--shell", "csh"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	if !strings.Contains(out.String(), `"code":"invalid_arguments"`) {
		t.Fatalf("stdout = %s", out.String())
	}
}

func TestResolveUnmatchedNoopJSONUsesNulls(t *testing.T) {
	tmp := t.TempDir()
	writeConfig(t, tmp, `
version: 1
defaults:
  on_unmatched: noop
rules:
  - name: other
    host: github.com
    owner: other
    account: target
`)
	bin := filepath.Join(tmp, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	writeExe(t, filepath.Join(bin, "git"), `#!/bin/sh
if [ "$#" -eq 2 ] && [ "$1 $2" = "rev-parse --is-inside-work-tree" ]; then exit 0; fi
if [ "$#" -eq 4 ] && [ "$1 $2 $3 $4" = "remote get-url -- origin" ]; then echo "git@github.com:org/repo.git"; exit 0; fi
exit 1
`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	var out, errOut bytes.Buffer
	code := Run([]string{"--json", "resolve"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("Run() code=%d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["account"] != nil || got["rule"] != nil || got["url_user"] != nil {
		t.Fatalf("unresolved fields should be null: %#v", got)
	}
	if got["matched"] != false || got["action"] != "unmatched_noop" {
		t.Fatalf("unexpected JSON result: %#v", got)
	}
}

func writeConfig(t *testing.T, tmp, content string) {
	t.Helper()
	path := filepath.Join(tmp, "config.yml")
	if err := os.WriteFile(path, []byte(strings.TrimLeft(content, "\n")), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GHAUTOSWITCH_CONFIG", path)
	clearTokenEnv(t)
}

func writeExe(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
}

func clearTokenEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN"} {
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
