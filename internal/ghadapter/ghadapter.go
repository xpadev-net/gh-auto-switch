package ghadapter

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	"github.com/xpadev-net/gh-auto-switch/internal/apperr"
)

var TokenEnvNames = []string{"GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN"}

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
	for _, name := range TokenEnvNames {
		if os.Getenv(name) != "" {
			return apperr.New(apperr.TokenEnvPresent, name+" is set")
		}
	}
	return nil
}

func StatusFor(host string) (Status, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return Status{}, apperr.New(apperr.GHNotFound, "gh command not found")
	}
	cmd := exec.Command("gh", "auth", "status", "--hostname", host, "--json", "hosts")
	cmd.Env = withGHHost(os.Environ(), host)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
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
		if a.Login == "" || a.State == "" {
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
		if a.Login == account && a.State == "success" {
			return nil
		}
	}
	return apperr.New(apperr.AccountUnavailable, "requested account is not available for host")
}

func Switch(host, account string) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return apperr.New(apperr.GHNotFound, "gh command not found")
	}
	cmd := exec.Command("gh", "auth", "switch", "--hostname", host, "--user", account)
	cmd.Env = withGHHost(os.Environ(), host)
	cmd.Stdin = nil
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = "gh auth switch failed"
		}
		return apperr.New(apperr.GHSwitchFailed, msg)
	}
	return nil
}

func withGHHost(env []string, host string) []string {
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if strings.HasPrefix(e, "GH_HOST=") {
			continue
		}
		out = append(out, e)
	}
	return append(out, "GH_HOST="+host)
}
