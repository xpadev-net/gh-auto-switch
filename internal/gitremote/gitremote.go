package gitremote

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/xpadev-net/gh-auto-switch/internal/apperr"
)

const commandTimeout = 10 * time.Second

func RemoteURL(remote string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	if err := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		return "", apperr.New(apperr.NotGitRepository, "not a git repository")
	}
	ctx, cancel = context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "--", remote)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", apperr.New(apperr.RemoteNotFound, "remote not found")
	}
	return strings.TrimSpace(string(out)), nil
}
