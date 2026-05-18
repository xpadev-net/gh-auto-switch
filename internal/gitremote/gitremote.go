package gitremote

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/xpadev-net/gh-auto-switch/internal/apperr"
)

func RemoteURL(remote string) (string, error) {
	if err := exec.Command("git", "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		return "", apperr.New(apperr.NotGitRepository, "not a git repository")
	}
	cmd := exec.Command("git", "remote", "get-url", remote)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", apperr.New(apperr.RemoteNotFound, "remote not found")
	}
	return strings.TrimSpace(string(out)), nil
}
