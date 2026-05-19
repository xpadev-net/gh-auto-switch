package matcher

import (
	"testing"

	"github.com/xpadev-net/gh-auto-switch/internal/config"
	"github.com/xpadev-net/gh-auto-switch/internal/parser"
)

func TestResolvePrecedence(t *testing.T) {
	cfg := config.Config{Rules: []config.Rule{
		{Name: "host", Host: "github.com", Account: "host-user"},
		{Name: "owner", Host: "github.com", Owner: "org", Account: "owner-user"},
		{Name: "remote", Host: "github.com", RemoteURL: "https://github.com/org/*", Account: "remote-user"},
		{Name: "url-user", Host: "github.com", URLUser: "hint", Account: "hint-user"},
	}}
	remote := parser.Remote{
		Host:          "github.com",
		Owner:         "org",
		Repo:          "repo",
		URLUser:       "hint",
		NormalizedURL: "https://github.com/org/repo.git",
	}
	got := Resolve(cfg, remote)
	if !got.Matched || got.Rule.Name != "url-user" {
		t.Fatalf("Resolve() = %+v", got)
	}
}

func TestResolveSameRankKeepsFirst(t *testing.T) {
	cfg := config.Config{Rules: []config.Rule{
		{Name: "first", Host: "github.com", Owner: "org", Account: "a"},
		{Name: "second", Host: "github.com", Owner: "org", Account: "b"},
	}}
	got := Resolve(cfg, parser.Remote{Host: "github.com", Owner: "org"})
	if !got.Matched || got.Rule.Name != "first" {
		t.Fatalf("Resolve() = %+v", got)
	}
}

func TestResolveTreatsMultipleConditionsAsAnd(t *testing.T) {
	cfg := config.Config{Rules: []config.Rule{
		{Name: "and-fails", Host: "github.com", URLUser: "hint", Owner: "other", Account: "a"},
		{Name: "host", Host: "github.com", Account: "b"},
	}}
	got := Resolve(cfg, parser.Remote{Host: "github.com", Owner: "org", URLUser: "hint"})
	if !got.Matched || got.Rule.Name != "host" {
		t.Fatalf("Resolve() = %+v", got)
	}
}

func TestResolveDefaultRuleIsLowestHostFallback(t *testing.T) {
	cfg := config.Config{Rules: []config.Rule{
		{Name: "default", Host: "github.com", Default: true, Account: "default-user"},
		{Name: "host", Host: "github.com", Account: "host-user"},
		{Name: "owner", Host: "github.com", Owner: "org", Account: "owner-user"},
	}}

	got := Resolve(cfg, parser.Remote{Host: "github.com", Owner: "other"})
	if !got.Matched || got.Rule.Name != "host" {
		t.Fatalf("Resolve() host fallback = %+v", got)
	}

	got = Resolve(cfg, parser.Remote{Host: "gitlab.com", Owner: "org"})
	if got.Matched {
		t.Fatalf("Resolve() matched different host: %+v", got)
	}
}

func TestResolveDefaultRuleMatchesWhenNoOtherHostRuleMatches(t *testing.T) {
	cfg := config.Config{Rules: []config.Rule{
		{Name: "owner", Host: "github.com", Owner: "org", Account: "owner-user"},
		{Name: "default", Host: "github.com", Default: true, Account: "default-user"},
	}}
	got := Resolve(cfg, parser.Remote{Host: "github.com", Owner: "other"})
	if !got.Matched || got.Rule.Name != "default" {
		t.Fatalf("Resolve() = %+v", got)
	}
}

func TestResolveDefaultRuleKeepsFallbackRankWithConditions(t *testing.T) {
	tests := []struct {
		name   string
		rule   config.Rule
		remote parser.Remote
	}{
		{
			name:   "url_user",
			rule:   config.Rule{Name: "default-url-user", Host: "github.com", Default: true, URLUser: "hint", Account: "default-user"},
			remote: parser.Remote{Host: "github.com", URLUser: "hint"},
		},
		{
			name:   "remote_url",
			rule:   config.Rule{Name: "default-remote-url", Host: "github.com", Default: true, RemoteURL: "https://github.com/org/*", Account: "default-user"},
			remote: parser.Remote{Host: "github.com", NormalizedURL: "https://github.com/org/repo.git"},
		},
		{
			name:   "owner",
			rule:   config.Rule{Name: "default-owner", Host: "github.com", Default: true, Owner: "org", Account: "default-user"},
			remote: parser.Remote{Host: "github.com", Owner: "org"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Config{Rules: []config.Rule{
				tc.rule,
				{Name: "host", Host: "github.com", Account: "host-user"},
			}}
			got := Resolve(cfg, tc.remote)
			if !got.Matched || got.Rule.Name != "host" {
				t.Fatalf("Resolve() = %+v", got)
			}
		})
	}
}
