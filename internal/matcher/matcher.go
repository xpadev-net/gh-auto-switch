package matcher

import (
	"path"

	"github.com/xpadev-net/gh-auto-switch/internal/config"
	"github.com/xpadev-net/gh-auto-switch/internal/parser"
)

type Match struct {
	Rule    *config.Rule
	Matched bool
}

func Resolve(cfg config.Config, remote parser.Remote) Match {
	bestIdx := -1
	bestRank := -1
	for i := range cfg.Rules {
		r := &cfg.Rules[i]
		rank, ok := ruleRank(*r, remote)
		if !ok {
			continue
		}
		if rank > bestRank {
			bestRank = rank
			bestIdx = i
		}
	}
	if bestIdx < 0 {
		return Match{}
	}
	return Match{Rule: &cfg.Rules[bestIdx], Matched: true}
}

func ruleRank(r config.Rule, remote parser.Remote) (int, bool) {
	if r.Host != remote.Host {
		return 0, false
	}
	rank := 1
	if r.URLUser != "" {
		if r.URLUser != remote.URLUser {
			return 0, false
		}
		rank = 4
	}
	if r.RemoteURL != "" {
		ok, err := path.Match(r.RemoteURL, remote.NormalizedURL)
		if err != nil || !ok {
			return 0, false
		}
		if rank < 3 {
			rank = 3
		}
	}
	if r.Owner != "" {
		ok, err := path.Match(r.Owner, remote.Owner)
		if err != nil || !ok {
			return 0, false
		}
		if rank < 2 {
			rank = 2
		}
	}
	return rank, true
}
