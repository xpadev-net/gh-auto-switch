package ghadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"time"

	"github.com/xpadev-net/gh-auto-switch/internal/apperr"
	"github.com/xpadev-net/gh-auto-switch/internal/procenv"
)

const commandTimeout = 15 * time.Second

type Account struct {
	Login       string `json:"login"`
	Active      bool   `json:"active"`
	State       string `json:"state"`
	TokenSource string `json:"tokenSource"`
}

type Status struct {
	Active   string
	Accounts []Account
}

func CheckTokenEnv() error {
	for _, name := range procenv.TokenEnvNames {
		if _, ok := os.LookupEnv(name); ok {
			return apperr.New(apperr.TokenEnvPresent, name+" is set")
		}
	}
	return nil
}

func StatusFor(host string) (Status, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return Status{}, apperr.New(apperr.GHNotFound, "gh command not found")
	}
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "auth", "status", "--hostname", host, "--json", "hosts")
	cmd.Env = procenv.WithGHHost(os.Environ(), host)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return Status{}, apperr.New(apperr.HostUnauthenticated, "gh auth status timed out")
		}
		return Status{}, apperr.New(apperr.HostUnauthenticated, "host is not authenticated")
	}
	var raw struct {
		Hosts map[string][]Account `json:"hosts"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		return Status{}, apperr.Wrap(apperr.HostUnauthenticated, "could not parse gh auth status", err)
	}
	accounts, ok := raw.Hosts[host]
	if !ok || len(accounts) == 0 {
		return Status{}, apperr.New(apperr.HostUnauthenticated, "host is not authenticated")
	}
	st := Status{Accounts: accounts}
	for _, a := range accounts {
		if a.Login == "" || a.State == "" || a.TokenSource == "" {
			return Status{}, apperr.New(apperr.HostUnauthenticated, "gh auth status has unknown account format")
		}
		if a.State != "success" {
			continue
		}
		if a.Active {
			if st.Active != "" {
				return Status{}, apperr.New(apperr.HostUnauthenticated, "multiple active accounts found")
			}
			st.Active = a.Login
		}
	}
	if st.Active == "" {
		return Status{}, apperr.New(apperr.HostUnauthenticated, "active account not found")
	}
	return st, nil
}

func EnsureAccount(st Status, account string) error {
	for _, a := range st.Accounts {
		if a.Login == account && a.State == "success" && a.TokenSource != "" {
			return nil
		}
	}
	return apperr.New(apperr.AccountUnavailable, "requested account is not available for host")
}

func Switch(host, account string) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return apperr.New(apperr.GHNotFound, "gh command not found")
	}
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "auth", "switch", "--hostname", host, "--user", account)
	cmd.Env = procenv.WithGHHost(os.Environ(), host)
	cmd.Stdin = nil
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return apperr.New(apperr.GHSwitchFailed, "gh auth switch timed out")
		}
		return apperr.New(apperr.GHSwitchFailed, "gh auth switch failed")
	}
	return nil
}
