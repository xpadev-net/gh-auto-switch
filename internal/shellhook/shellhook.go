package shellhook

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/xpadev-net/gh-auto-switch/internal/apperr"
)

const (
	beginMarker = "# >>> gh-auto-switch hook >>>"
	endMarker   = "# <<< gh-auto-switch hook <<<"
)

type Shell string

const (
	Bash Shell = "bash"
	Zsh  Shell = "zsh"
	Fish Shell = "fish"
)

type Options struct {
	Shell     string
	PrintOnly bool
}

func Install(opts Options) (string, string, error) {
	sh, err := Detect(opts.Shell)
	if err != nil {
		return "", "", err
	}
	snippet := Snippet(sh)
	if opts.PrintOnly {
		return "", snippet, nil
	}
	path, err := ConfigPath(sh)
	if err != nil {
		return "", "", err
	}
	if err := installTo(path, snippet); err != nil {
		return "", "", err
	}
	return path, snippet, nil
}

func Detect(name string) (Shell, error) {
	if name == "" {
		name = filepath.Base(os.Getenv("SHELL"))
	}
	switch name {
	case "bash":
		return Bash, nil
	case "zsh":
		return Zsh, nil
	case "fish":
		return Fish, nil
	default:
		return "", apperr.New(apperr.InvalidArguments, "shell must be bash, zsh, or fish")
	}
}

func ConfigPath(sh Shell) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", apperr.Wrap(apperr.InvalidArguments, "could not resolve home directory", err)
	}
	switch sh {
	case Bash:
		return filepath.Join(home, ".bashrc"), nil
	case Zsh:
		return filepath.Join(home, ".zshrc"), nil
	case Fish:
		return filepath.Join(home, ".config", "fish", "config.fish"), nil
	default:
		return "", apperr.New(apperr.InvalidArguments, "shell must be bash, zsh, or fish")
	}
}

func Snippet(sh Shell) string {
	switch sh {
	case Fish:
		return strings.Join([]string{
			beginMarker,
			"function gh",
			"    gh-auto-switch exec -- gh $argv",
			"end",
			endMarker,
			"",
		}, "\n")
	default:
		return strings.Join([]string{
			beginMarker,
			"gh() {",
			"    gh-auto-switch exec -- gh \"$@\"",
			"}",
			endMarker,
			"",
		}, "\n")
	}
}

func installTo(path, snippet string) error {
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return apperr.Wrap(apperr.InvalidArguments, "could not create shell config directory", err)
	}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return apperr.Wrap(apperr.InvalidArguments, "could not read shell config file", err)
	}
	next, err := upsertBlock(string(data), snippet)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(next), 0644); err != nil {
		return apperr.Wrap(apperr.InvalidArguments, "could not write shell config file", err)
	}
	return nil
}

func upsertBlock(content, snippet string) (string, error) {
	begin := strings.Index(content, beginMarker)
	end := strings.Index(content, endMarker)
	if begin >= 0 || end >= 0 {
		if begin < 0 || end < 0 || end < begin {
			return "", apperr.New(apperr.InvalidArguments, "existing gh-auto-switch hook block is malformed")
		}
		end += len(endMarker)
		if end < len(content) && content[end] == '\r' {
			end++
		}
		if end < len(content) && content[end] == '\n' {
			end++
		}
		return content[:begin] + snippet + content[end:], nil
	}
	if content == "" {
		return snippet, nil
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return fmt.Sprintf("%s\n%s", content, snippet), nil
}
