package parser

import (
	"net"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/net/idna"

	"github.com/xpadev-net/gh-auto-switch/internal/apperr"
)

type Remote struct {
	Scheme        string
	Host          string
	Owner         string
	Repo          string
	URLUser       string
	MaskedURLUser string
	NormalizedURL string
}

var scpLike = regexp.MustCompile(`^([^@:/\s]+)@([^:/\s]+):([^:]+)$`)

func Parse(raw string) (Remote, error) {
	if strings.TrimSpace(raw) == "" {
		return Remote{}, apperr.New(apperr.RemoteURLUnparseable, "remote URL is empty")
	}
	if m := scpLike.FindStringSubmatch(raw); m != nil {
		if strings.EqualFold(m[1], "git") {
			return parseParts("ssh", m[2], m[3], "")
		}
		return Remote{}, apperr.New(apperr.RemoteURLUnparseable, "unsupported SSH user")
	}

	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return Remote{}, apperr.New(apperr.RemoteURLUnparseable, "remote URL is unparseable")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "https" && scheme != "ssh" {
		return Remote{}, apperr.New(apperr.RemoteURLUnparseable, "unsupported remote URL scheme")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return Remote{}, apperr.New(apperr.RemoteURLUnparseable, "query and fragment are unsupported")
	}
	if u.Port() != "" {
		return Remote{}, apperr.New(apperr.RemoteURLUnparseable, "ports are unsupported")
	}
	if strings.Contains(u.Host, ":") {
		return Remote{}, apperr.New(apperr.RemoteURLUnparseable, "ports and IPv6 literals are unsupported")
	}
	if scheme == "ssh" && u.User != nil && u.User.Username() != "git" {
		return Remote{}, apperr.New(apperr.RemoteURLUnparseable, "unsupported SSH user")
	}
	urlUser := ""
	if scheme == "https" && u.User != nil {
		urlUser = u.User.Username()
	}
	return parseParts(scheme, u.Hostname(), u.EscapedPath(), urlUser)
}

func parseParts(scheme, host, path, urlUser string) (Remote, error) {
	host, err := NormalizeHost(host)
	if err != nil {
		return Remote{}, err
	}
	if strings.HasSuffix(path, "/") || strings.Contains(path, "//") {
		return Remote{}, apperr.New(apperr.RemoteURLUnparseable, "remote path is invalid")
	}
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return Remote{}, apperr.New(apperr.RemoteURLUnparseable, "remote path must be owner/repo")
	}
	owner, repo := parts[0], parts[1]
	if strings.Contains(strings.ToLower(path), "%2f") {
		return Remote{}, apperr.New(apperr.RemoteURLUnparseable, "encoded path separators are unsupported")
	}
	if strings.HasSuffix(repo, ".git") {
		repo = strings.TrimSuffix(repo, ".git")
	}
	if strings.HasSuffix(repo, ".git") {
		return Remote{}, apperr.New(apperr.RemoteURLUnparseable, "repository.git.git is unsupported")
	}
	if !validPathPart(owner) || !validPathPart(repo) {
		return Remote{}, apperr.New(apperr.RemoteURLUnparseable, "owner or repository is invalid")
	}
	normalizedScheme := "https"
	if scheme == "ssh" {
		normalizedScheme = "ssh"
	}
	prefix := "https://" + host
	if normalizedScheme == "ssh" {
		prefix = "ssh://git@" + host
	}
	masked := ""
	if urlUser != "" {
		masked = urlUser
		if LooksLikeToken(urlUser) {
			masked = "***"
		}
	}
	return Remote{
		Scheme:        normalizedScheme,
		Host:          host,
		Owner:         owner,
		Repo:          repo,
		URLUser:       urlUser,
		MaskedURLUser: masked,
		NormalizedURL: prefix + "/" + owner + "/" + repo + ".git",
	}, nil
}

func NormalizeHost(host string) (string, error) {
	if host == "" || strings.Contains(host, "/") || strings.ContainsAny(host, ";&|`$<>\\\"'(){}[]*?!") {
		return "", apperr.New(apperr.RemoteURLUnparseable, "host is invalid")
	}
	for _, r := range host {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return "", apperr.New(apperr.RemoteURLUnparseable, "host is invalid")
		}
	}
	if ip := net.ParseIP(host); ip != nil {
		return "", apperr.New(apperr.RemoteURLUnparseable, "IP literals are unsupported")
	}
	ascii, err := idna.Lookup.ToASCII(strings.ToLower(host))
	if err != nil || ascii == "" || strings.Contains(ascii, "..") {
		return "", apperr.New(apperr.RemoteURLUnparseable, "host is invalid")
	}
	for _, label := range strings.Split(ascii, ".") {
		if label == "" || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "", apperr.New(apperr.RemoteURLUnparseable, "host is invalid")
		}
	}
	return ascii, nil
}

func validPathPart(s string) bool {
	if s == "" || s == "." || s == ".." || strings.Contains(s, "/") || strings.Contains(s, "\\") {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func LooksLikeToken(s string) bool {
	if len(s) >= 40 {
		return true
	}
	for _, p := range []string{"ghp_", "github_pat_", "gho_", "ghu_", "ghs_", "ghr_"} {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func MaskURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	name := u.User.Username()
	if LooksLikeToken(name) {
		name = "***"
	}
	if _, ok := u.User.Password(); ok {
		u.User = url.UserPassword(name, "***")
	} else {
		u.User = url.User(name)
	}
	return u.String()
}
