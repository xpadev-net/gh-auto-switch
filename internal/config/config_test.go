package config

import "testing"

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
