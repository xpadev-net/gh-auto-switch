package parser

import "testing"

func TestParseSupportedURLs(t *testing.T) {
	tests := []struct {
		raw        string
		scheme     string
		host       string
		owner      string
		repo       string
		urlUser    string
		normalized string
	}{
		{"https://github.com/owner/repo.git", "https", "github.com", "owner", "repo", "", "https://github.com/owner/repo.git"},
		{"https://github.com/owner/repo", "https", "github.com", "owner", "repo", "", "https://github.com/owner/repo.git"},
		{"https://hogehoge@github.com/org/repo.git", "https", "github.com", "org", "repo", "hogehoge", "https://github.com/org/repo.git"},
		{"git@github.com:owner/repo.git", "ssh", "github.com", "owner", "repo", "", "ssh://git@github.com/owner/repo.git"},
		{"ssh://git@github.example.com/owner/repo.git", "ssh", "github.example.com", "owner", "repo", "", "ssh://git@github.example.com/owner/repo.git"},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := Parse(tt.raw)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got.Scheme != tt.scheme || got.Host != tt.host || got.Owner != tt.owner || got.Repo != tt.repo || got.URLUser != tt.urlUser || got.NormalizedURL != tt.normalized {
				t.Fatalf("Parse() = %+v", got)
			}
		})
	}
}

func TestParseRejectsUnsupportedURLs(t *testing.T) {
	tests := []string{
		"https://github.com:443/owner/repo.git",
		"ssh://git@github.com:22/owner/repo.git",
		"https://github.com/owner/repo.git?x=1",
		"https://github.com/owner/repo/extra.git",
		"https://github.com/owner/repo/",
		"https://github.com/owner/repo.git.git",
		"https://github.com/owner%2Frepo.git",
		"git@github.com:22/owner/repo.git",
	}
	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			if _, err := Parse(raw); err == nil {
				t.Fatalf("Parse() succeeded")
			}
		})
	}
}

func TestTokenLikeURLUserIsMasked(t *testing.T) {
	got, err := Parse("https://ghp_secret@github.com/owner/repo.git")
	if err != nil {
		t.Fatal(err)
	}
	if got.URLUser != "ghp_secret" {
		t.Fatalf("raw url user not preserved internally: %q", got.URLUser)
	}
	if got.MaskedURLUser != "***" {
		t.Fatalf("masked url user = %q", got.MaskedURLUser)
	}
}
