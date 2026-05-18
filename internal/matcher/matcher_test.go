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
