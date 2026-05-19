# gh-auto-switch

`gh-auto-switch` switches the active GitHub CLI account for the current Git repository's remote URL.

## Build

Install the latest version:

```bash
go install github.com/xpadev-net/gh-auto-switch/cmd/gh-auto-switch@master
```

Make sure your Go binary directory, usually `$(go env GOPATH)/bin`, is on `PATH`.

Build from a local checkout:

```bash
go build ./cmd/gh-auto-switch
```

## Config

The default config path is `~/.config/ghautoswitch/config.yml`. Set `GHAUTOSWITCH_CONFIG` to use another path.

```yaml
version: 1

defaults:
  remote: origin
  on_unmatched: error
  on_unauthenticated: error

rules:
  - name: github-personal-by-url-user
    host: github.com
    url_user: your-login
    account: your-login

  - name: github-work-by-owner
    host: github.com
    owner: your-company
    account: work-login

  - name: github-default
    host: github.com
    default: true
    account: your-login
```

Set `default: true` on a rule to use it as the lowest-priority fallback for that host. More specific rules using `url_user`, `remote_url`, or `owner` still take precedence.

Outside a Git repository, `switch` and the installed `gh` hook use only an unconditional rule marked `default: true` as the account fallback. A bare host-only rule is not used for this outside-repository fallback.

Config files must not be symlinks or group/world writable. Files created by `gh-auto-switch init` use `0600`; the default config directory uses `0700`.

## Commands

```bash
gh-auto-switch init
gh-auto-switch resolve
gh-auto-switch switch
gh-auto-switch switch --remote upstream
gh-auto-switch exec -- gh pr list
eval "$(gh-auto-switch print-env)"
gh-auto-switch check --json
gh-auto-switch install
gh-auto-switch install --shell fish --print
```

`print-env` only prints `GH_HOST`; it does not switch accounts. `switch` changes `gh` state for the target host but cannot change the parent shell environment.

`install` adds a managed `gh` shell function to bash, zsh, or fish so normal `gh ...` commands run through `gh-auto-switch exec`. It updates `~/.bashrc`, `~/.zshrc`, or `~/.config/fish/config.fish` by default; use `--print` to print the hook without writing files.

## Authentication Notes

`switch`, `exec`, and `check` fail closed when `GH_TOKEN`, `GITHUB_TOKEN`, `GH_ENTERPRISE_TOKEN`, or `GITHUB_ENTERPRISE_TOKEN` is set. `GH_CONFIG_DIR` is inherited by subprocesses, so it affects which `gh` authentication store is inspected and switched.

`gh` stores the active account in a user-level authentication store. `gh-auto-switch switch` and `exec` hold a lock for the current OS user's effective `GH_CONFIG_DIR` store; `exec` keeps that lock while the child process runs. Tools that bypass `gh-auto-switch` can still change `gh` state.

When the installed hook wraps a long-running or interactive `gh` command, other `gh-auto-switch` commands using the same auth store wait for the lock and can time out after 10 seconds.
