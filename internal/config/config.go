package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"

	"github.com/xpadev-net/gh-auto-switch/internal/apperr"
	"github.com/xpadev-net/gh-auto-switch/internal/parser"
)

type Config struct {
	Version  int      `yaml:"version"`
	Defaults Defaults `yaml:"defaults"`
	Rules    []Rule   `yaml:"rules"`
}

type Defaults struct {
	Remote            string `yaml:"remote"`
	OnUnmatched       string `yaml:"on_unmatched"`
	OnUnauthenticated string `yaml:"on_unauthenticated"`
}

type Rule struct {
	Name      string `yaml:"name"`
	Host      string `yaml:"host"`
	URLUser   string `yaml:"url_user"`
	RemoteURL string `yaml:"remote_url"`
	Owner     string `yaml:"owner"`
	Account   string `yaml:"account"`
}

func DefaultPath() (string, error) {
	if p := os.Getenv("GHAUTOSWITCH_CONFIG"); p != "" {
		return filepath.Abs(p)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "ghautoswitch", "config.yml"), nil
}

func Load() (Config, string, error) {
	path, err := DefaultPath()
	if err != nil {
		return Config{}, "", apperr.Wrap(apperr.ConfigInvalid, "could not resolve config path", err)
	}
	if err := checkSecure(path, false); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Config{}, path, apperr.New(apperr.ConfigNotFound, "config file not found")
		}
		return Config{}, path, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Config{}, path, apperr.New(apperr.ConfigNotFound, "config file not found")
		}
		return Config{}, path, apperr.Wrap(apperr.ConfigInvalid, "could not read config file", err)
	}
	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, path, apperr.Wrap(apperr.ConfigInvalid, "config YAML is invalid", err)
	}
	if err := Validate(&cfg); err != nil {
		return Config{}, path, err
	}
	return cfg, path, nil
}

func Validate(cfg *Config) error {
	if cfg.Version != 1 {
		return apperr.New(apperr.ConfigInvalid, "version must be integer 1")
	}
	if cfg.Defaults.Remote == "" {
		cfg.Defaults.Remote = "origin"
	}
	if cfg.Defaults.OnUnmatched == "" {
		cfg.Defaults.OnUnmatched = "error"
	}
	if cfg.Defaults.OnUnauthenticated == "" {
		cfg.Defaults.OnUnauthenticated = "error"
	}
	if !ValidRemoteName(cfg.Defaults.Remote) {
		return apperr.New(apperr.ConfigInvalid, "defaults.remote is invalid")
	}
	if cfg.Defaults.OnUnmatched != "noop" && cfg.Defaults.OnUnmatched != "error" {
		return apperr.New(apperr.ConfigInvalid, "defaults.on_unmatched must be noop or error")
	}
	if cfg.Defaults.OnUnauthenticated != "error" {
		return apperr.New(apperr.ConfigInvalid, "defaults.on_unauthenticated must be error")
	}
	if len(cfg.Rules) == 0 {
		return apperr.New(apperr.ConfigInvalid, "rules must contain at least one rule")
	}
	seen := map[string]bool{}
	for i := range cfg.Rules {
		r := &cfg.Rules[i]
		if !validSimple(r.Name) {
			return apperr.New(apperr.ConfigInvalid, "rule name is invalid")
		}
		if seen[r.Name] {
			return apperr.New(apperr.ConfigInvalid, "rule names must be unique")
		}
		seen[r.Name] = true
		host, err := parser.NormalizeHost(r.Host)
		if err != nil {
			return apperr.New(apperr.ConfigInvalid, "rule host is invalid")
		}
		r.Host = host
		if !validSimple(r.Account) {
			return apperr.New(apperr.ConfigInvalid, "rule account is invalid")
		}
		if r.URLUser != "" && !validSimple(r.URLUser) {
			return apperr.New(apperr.ConfigInvalid, "rule url_user is invalid")
		}
		if r.RemoteURL != "" {
			if _, err := path.Match(r.RemoteURL, "https://github.com/owner/repo.git"); err != nil {
				return apperr.Wrap(apperr.ConfigInvalid, "rule remote_url glob is invalid", err)
			}
		}
		if r.Owner != "" {
			if _, err := path.Match(r.Owner, "owner"); err != nil {
				return apperr.Wrap(apperr.ConfigInvalid, "rule owner glob is invalid", err)
			}
		}
	}
	return nil
}

func validSimple(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func ValidRemoteName(s string) bool {
	return validSimple(s) && !strings.HasPrefix(s, "-")
}

func checkSecure(path string, allowMissingFile bool) error {
	parent := filepath.Dir(path)
	if err := checkPath(parent, true); err != nil {
		return err
	}
	if allowMissingFile {
		if _, err := os.Lstat(path); errors.Is(err, fs.ErrNotExist) {
			return nil
		}
	}
	return checkPath(path, false)
}

func checkPath(path string, wantDir bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return apperr.New(apperr.ConfigInvalid, fmt.Sprintf("%s must not be a symlink", path))
	}
	if wantDir && !info.IsDir() {
		return apperr.New(apperr.ConfigInvalid, fmt.Sprintf("%s must be a directory", path))
	}
	if !wantDir && !info.Mode().IsRegular() {
		return apperr.New(apperr.ConfigInvalid, fmt.Sprintf("%s must be a regular file", path))
	}
	if info.Mode().Perm()&0022 != 0 {
		return apperr.New(apperr.ConfigInvalid, fmt.Sprintf("%s must not be group/world writable", path))
	}
	if ownerInvalid(info) {
		return apperr.New(apperr.ConfigInvalid, fmt.Sprintf("%s owner is invalid", path))
	}
	return nil
}

func Template() string {
	return strings.TrimLeft(`
# gh-auto-switch global configuration
version: 1

defaults:
  remote: origin
  on_unmatched: error
  on_unauthenticated: error

rules:
  - name: github-personal-by-url-user
    host: github.com
    url_user: your-login
    account: your-login

  - name: github-work-by-owner
    host: github.com
    owner: your-company
    account: work-login
`, "\n")
}

func Init(printOnly, force bool) (string, error) {
	path, err := DefaultPath()
	if err != nil {
		return "", apperr.Wrap(apperr.ConfigInvalid, "could not resolve config path", err)
	}
	content := Template()
	if printOnly {
		return content, nil
	}
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return "", apperr.Wrap(apperr.ConfigInvalid, "could not create config directory", err)
	}
	_ = os.Chmod(parent, 0700)
	if err := checkSecure(path, true); err != nil {
		return "", err
	}
	if err := writeFileAtomic(path, content, force); err != nil {
		return "", err
	}
	return path, nil
}

func writeFileAtomic(path, content string, force bool) error {
	parent := filepath.Dir(path)
	tmp, err := os.CreateTemp(parent, ".config.yml.*")
	if err != nil {
		return apperr.Wrap(apperr.ConfigInvalid, "could not create temporary config file", err)
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return apperr.Wrap(apperr.ConfigInvalid, "could not secure temporary config file", err)
	}
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return apperr.Wrap(apperr.ConfigInvalid, "could not write config file", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return apperr.Wrap(apperr.ConfigInvalid, "could not sync config file", err)
	}
	if err := tmp.Close(); err != nil {
		return apperr.Wrap(apperr.ConfigInvalid, "could not close config file", err)
	}
	if force {
		if err := os.Rename(tmpPath, path); err != nil {
			return apperr.Wrap(apperr.ConfigInvalid, "could not replace config file", err)
		}
		cleanup = false
		return nil
	}
	if err := os.Link(tmpPath, path); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return apperr.New(apperr.ConfigInvalid, "config file already exists")
		}
		return apperr.Wrap(apperr.ConfigInvalid, "could not create config file", err)
	}
	_ = os.Remove(tmpPath)
	cleanup = false
	return nil
}
