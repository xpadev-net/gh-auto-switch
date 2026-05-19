package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRejectsUnknownOnUnmatched(t *testing.T) {
	cfg := Config{
		Version:  1,
		Defaults: Defaults{OnUnmatched: "bad", OnUnauthenticated: "error"},
		Rules:    []Rule{{Name: "r", Host: "github.com", Account: "u"}},
	}
	if err := Validate(&cfg); err == nil {
		t.Fatalf("Validate() succeeded")
	}
}

func TestValidateNormalizesDefaultsAndHost(t *testing.T) {
	cfg := Config{
		Version: 1,
		Rules:   []Rule{{Name: "r", Host: "GitHub.COM", Account: "u"}},
	}
	if err := Validate(&cfg); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if cfg.Defaults.Remote != "origin" || cfg.Defaults.OnUnmatched != "error" || cfg.Defaults.OnUnauthenticated != "error" {
		t.Fatalf("defaults not normalized: %+v", cfg.Defaults)
	}
	if cfg.Rules[0].Host != "github.com" {
		t.Fatalf("host not normalized: %q", cfg.Rules[0].Host)
	}
}

func TestValidateRejectsOptionLikeRemoteName(t *testing.T) {
	cfg := Config{
		Version:  1,
		Defaults: Defaults{Remote: "-x"},
		Rules:    []Rule{{Name: "r", Host: "github.com", Account: "u"}},
	}
	if err := Validate(&cfg); err == nil {
		t.Fatalf("Validate() succeeded")
	}
}

func TestLoadAcceptsDefaultRuleField(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yml")
	if err := os.WriteFile(path, []byte(`
version: 1
rules:
  - name: fallback
    host: github.com
    default: true
    account: u
`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GHAUTOSWITCH_CONFIG", path)

	cfg, _, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Rules[0].Default {
		t.Fatalf("default rule field not loaded: %+v", cfg.Rules[0])
	}
}

func TestDefaultRuleRequiresDefaultField(t *testing.T) {
	cfg := Config{Rules: []Rule{
		{Name: "host-only", Host: "github.com", Account: "wrong"},
		{Name: "fallback", Host: "github.com", Default: true, Account: "right"},
	}}
	got := DefaultRule(cfg)
	if got == nil || got.Name != "fallback" {
		t.Fatalf("DefaultRule() = %+v", got)
	}
}
