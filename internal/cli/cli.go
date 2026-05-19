package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/xpadev-net/gh-auto-switch/internal/apperr"
	"github.com/xpadev-net/gh-auto-switch/internal/config"
	"github.com/xpadev-net/gh-auto-switch/internal/ghadapter"
	"github.com/xpadev-net/gh-auto-switch/internal/gitremote"
	"github.com/xpadev-net/gh-auto-switch/internal/lock"
	"github.com/xpadev-net/gh-auto-switch/internal/matcher"
	"github.com/xpadev-net/gh-auto-switch/internal/output"
	"github.com/xpadev-net/gh-auto-switch/internal/parser"
	"github.com/xpadev-net/gh-auto-switch/internal/procenv"
	"github.com/xpadev-net/gh-auto-switch/internal/shellhook"
)

type globals struct {
	json    bool
	verbose bool
}

type resolved struct {
	cfg    config.Config
	remote parser.Remote
	match  matcher.Match
}

func Run(args []string, stdout, stderr io.Writer) int {
	code, err := run(args, stdout, stderr)
	if err == nil {
		return code
	}
	app := apperr.From(err)
	if globalsFromArgs(args).json {
		output.JSONError(stdout, app)
	} else {
		fmt.Fprintln(stderr, app.Message)
	}
	return apperr.ExitCode(app.Code)
}

func run(args []string, stdout, stderr io.Writer) (int, error) {
	if len(args) == 0 {
		return 1, apperr.New(apperr.InvalidArguments, "subcommand is required")
	}
	g, args := splitGlobals(args)
	if len(args) == 0 {
		return 1, apperr.New(apperr.InvalidArguments, "subcommand is required")
	}
	switch args[0] {
	case "resolve":
		return cmdResolve(args[1:], g, stdout, stderr)
	case "switch":
		return cmdSwitch(args[1:], g, stdout, stderr)
	case "exec":
		return cmdExec(args[1:], g, stdout, stderr)
	case "print-env":
		return cmdPrintEnv(args[1:], g, stdout, stderr)
	case "check":
		return cmdCheck(args[1:], g, stdout, stderr)
	case "init":
		return cmdInit(args[1:], g, stdout, stderr)
	case "install":
		return cmdInstall(args[1:], g, stdout, stderr)
	default:
		return 1, apperr.New(apperr.InvalidArguments, "unknown subcommand")
	}
}

func splitGlobals(args []string) (globals, []string) {
	var g globals
	out := make([]string, 0, len(args))
	afterDashDash := false
	for _, a := range args {
		if afterDashDash {
			out = append(out, a)
			continue
		}
		if a == "--" {
			afterDashDash = true
			out = append(out, a)
			continue
		}
		switch a {
		case "--json":
			g.json = true
		case "--verbose":
			g.verbose = true
		default:
			out = append(out, a)
		}
	}
	return g, out
}

func globalsFromArgs(args []string) globals {
	g, _ := splitGlobals(args)
	return g
}

func flagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func cmdResolve(args []string, g globals, stdout, stderr io.Writer) (int, error) {
	fs := flagSet("resolve")
	remoteFlag := fs.String("remote", "", "")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 {
		return 1, apperr.New(apperr.InvalidArguments, "invalid resolve arguments")
	}
	if *remoteFlag != "" && !config.ValidRemoteName(*remoteFlag) {
		return 1, apperr.New(apperr.InvalidArguments, "remote name is invalid")
	}
	r, err := resolve(*remoteFlag, g, stderr)
	if err != nil {
		return 1, err
	}
	res := resultFor(r, "resolved")
	if !r.match.Matched {
		if r.cfg.Defaults.OnUnmatched == "error" {
			return 4, apperr.New(apperr.UnmatchedRule, "no matching rule")
		}
		res.Action = "unmatched_noop"
	}
	writeResult(stdout, g, res)
	return 0, nil
}

func cmdSwitch(args []string, g globals, stdout, stderr io.Writer) (int, error) {
	fs := flagSet("switch")
	remoteFlag := fs.String("remote", "", "")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 {
		return 1, apperr.New(apperr.InvalidArguments, "invalid switch arguments")
	}
	if *remoteFlag != "" && !config.ValidRemoteName(*remoteFlag) {
		return 1, apperr.New(apperr.InvalidArguments, "remote name is invalid")
	}
	if err := ghadapter.CheckTokenEnv(); err != nil {
		return 3, err
	}
	r, err := resolve(*remoteFlag, g, stderr)
	if err != nil {
		if apperr.From(err).Code == apperr.NotGitRepository {
			return switchDefaultAccount(g, stdout)
		}
		return 1, err
	}
	if !r.match.Matched {
		if r.cfg.Defaults.OnUnmatched == "noop" {
			res := resultFor(r, "unmatched_noop")
			writeResult(stdout, g, res)
			return 0, nil
		}
		return 4, apperr.New(apperr.UnmatchedRule, "no matching rule")
	}
	l, err := lock.AcquireAuthStore(10 * time.Second)
	if err != nil {
		return 1, err
	}
	defer l.Release()
	action, err := ensureSwitched(r.remote.Host, r.match.Rule.Account)
	if err != nil {
		return 3, err
	}
	res := resultFor(r, action)
	writeResult(stdout, g, res)
	return 0, nil
}

func switchDefaultAccount(g globals, stdout io.Writer) (int, error) {
	cfg, _, err := config.Load()
	if err != nil {
		return 2, err
	}
	rule := config.DefaultRule(cfg)
	if rule == nil {
		return 1, apperr.New(apperr.NotGitRepository, "not a git repository")
	}
	l, err := lock.AcquireAuthStore(10 * time.Second)
	if err != nil {
		return 1, err
	}
	defer l.Release()
	action, err := ensureSwitched(rule.Host, rule.Account)
	if err != nil {
		return 3, err
	}
	res := output.Result{
		Host:    output.Ptr(rule.Host),
		Account: output.Ptr(rule.Account),
		Rule:    output.Ptr(rule.Name),
		Matched: true,
		Action:  action,
	}
	writeResult(stdout, g, res)
	return 0, nil
}

func cmdExec(args []string, g globals, stdout, stderr io.Writer) (int, error) {
	idx := indexOf(args, "--")
	if idx < 0 {
		return 1, apperr.New(apperr.InvalidArguments, "exec requires -- before command")
	}
	pre, cmdArgs := args[:idx], args[idx+1:]
	fs := flagSet("exec")
	remoteFlag := fs.String("remote", "", "")
	allowUnmatched := fs.Bool("allow-unmatched", false, "")
	allowUnmatchedIfNoop := fs.Bool("allow-unmatched-if-noop", false, "")
	if err := fs.Parse(pre); err != nil || fs.NArg() != 0 {
		return 1, apperr.New(apperr.InvalidArguments, "invalid exec arguments")
	}
	if *remoteFlag != "" && !config.ValidRemoteName(*remoteFlag) {
		return 1, apperr.New(apperr.InvalidArguments, "remote name is invalid")
	}
	if err := ghadapter.CheckTokenEnv(); err != nil {
		return 3, err
	}
	if len(cmdArgs) == 0 {
		return 1, apperr.New(apperr.ExecCommandEmpty, "exec command is empty")
	}
	r, err := resolve(*remoteFlag, g, stderr)
	if err != nil {
		if apperr.From(err).Code == apperr.NotGitRepository && cmdArgs[0] == "gh" {
			return execDefaultAccount(cmdArgs, stdout, stderr)
		}
		return 1, err
	}
	if !r.match.Matched {
		allowByConfig := *allowUnmatchedIfNoop && r.cfg.Defaults.OnUnmatched == "noop"
		if !*allowUnmatched && !allowByConfig {
			return 4, apperr.New(apperr.UnmatchedRule, "no matching rule")
		}
		l, err := lock.AcquireAuthStore(10 * time.Second)
		if err != nil {
			return 1, err
		}
		defer l.Release()
		return runChild(cmdArgs, r.remote.Host, stdout, stderr)
	}
	l, err := lock.AcquireAuthStore(10 * time.Second)
	if err != nil {
		return 1, err
	}
	defer l.Release()
	return runChildWithAccount(cmdArgs, r.remote.Host, r.match.Rule.Account, stdout, stderr)
}

func execDefaultAccount(cmdArgs []string, stdout, stderr io.Writer) (int, error) {
	cfg, _, err := config.Load()
	if err != nil {
		return 2, err
	}
	rule := config.DefaultRule(cfg)
	if rule == nil {
		return 1, apperr.New(apperr.NotGitRepository, "not a git repository")
	}
	l, err := lock.AcquireAuthStore(10 * time.Second)
	if err != nil {
		return 1, err
	}
	defer l.Release()
	return runChildWithAccount(cmdArgs, rule.Host, rule.Account, stdout, stderr)
}

func runChildWithAccount(cmdArgs []string, host, account string, stdout, stderr io.Writer) (int, error) {
	if _, err := ensureSwitched(host, account); err != nil {
		return 3, err
	}
	st, err := ghadapter.StatusFor(host)
	if err != nil {
		return 3, err
	}
	if st.Active != account {
		return 3, apperr.New(apperr.HostUnauthenticated, "active account changed before exec")
	}
	return runChild(cmdArgs, host, stdout, stderr)
}

func cmdPrintEnv(args []string, g globals, stdout, stderr io.Writer) (int, error) {
	fs := flagSet("print-env")
	remoteFlag := fs.String("remote", "", "")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 {
		return 1, apperr.New(apperr.InvalidArguments, "invalid print-env arguments")
	}
	if *remoteFlag != "" && !config.ValidRemoteName(*remoteFlag) {
		return 1, apperr.New(apperr.InvalidArguments, "remote name is invalid")
	}
	r, err := resolve(*remoteFlag, g, stderr)
	if err != nil {
		return 1, err
	}
	if !r.match.Matched && r.cfg.Defaults.OnUnmatched == "error" {
		return 4, apperr.New(apperr.UnmatchedRule, "no matching rule")
	}
	if g.json {
		res := resultFor(r, "resolved")
		if !r.match.Matched {
			res.Action = "unmatched_noop"
		}
		writeResult(stdout, g, res)
		return 0, nil
	}
	fmt.Fprintf(stdout, "export GH_HOST='%s'\n", shellQuoteValue(r.remote.Host))
	return 0, nil
}

func cmdCheck(args []string, g globals, stdout, stderr io.Writer) (int, error) {
	fs := flagSet("check")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 {
		return 1, apperr.New(apperr.InvalidArguments, "invalid check arguments")
	}
	cfg, _, err := config.Load()
	if err != nil {
		return 2, err
	}
	if err := ghadapter.CheckTokenEnv(); err != nil {
		return 3, err
	}
	hosts := map[string]map[string]bool{}
	for _, r := range cfg.Rules {
		if hosts[r.Host] == nil {
			hosts[r.Host] = map[string]bool{}
		}
		hosts[r.Host][r.Account] = true
	}
	var problems []string
	problemCode := apperr.HostUnauthenticated
	for _, host := range sortedKeys(hosts) {
		st, err := ghadapter.StatusFor(host)
		if err != nil {
			app := apperr.From(err)
			problems = append(problems, host+": "+app.Message)
			problemCode = app.Code
			continue
		}
		for _, account := range sortedAccountKeys(hosts[host]) {
			if err := ghadapter.EnsureAccount(st, account); err != nil {
				app := apperr.From(err)
				problems = append(problems, host+"/"+account+": "+app.Message)
				if problemCode == apperr.HostUnauthenticated {
					problemCode = app.Code
				}
			}
		}
	}
	if len(problems) > 0 {
		return 3, apperr.New(problemCode, "check failed: "+strings.Join(problems, "; "))
	}
	res := output.Result{Matched: true, Action: "check_passed"}
	writeResult(stdout, g, res)
	return 0, nil
}

func cmdInit(args []string, g globals, stdout, stderr io.Writer) (int, error) {
	fs := flagSet("init")
	force := fs.Bool("force", false, "")
	printOnly := fs.Bool("print", false, "")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 {
		return 1, apperr.New(apperr.InvalidArguments, "invalid init arguments")
	}
	v, err := config.Init(*printOnly, *force)
	if err != nil {
		return 2, err
	}
	if *printOnly {
		if g.json {
			writeResult(stdout, g, output.Result{Matched: true, Action: "none"})
			return 0, nil
		}
		fmt.Fprint(stdout, v)
		return 0, nil
	}
	res := output.Result{Matched: true, Action: "init_created"}
	if g.json {
		writeResult(stdout, g, res)
	} else {
		fmt.Fprintf(stdout, "created: %s\n", v)
	}
	return 0, nil
}

func cmdInstall(args []string, g globals, stdout, stderr io.Writer) (int, error) {
	fs := flagSet("install")
	shellName := fs.String("shell", "", "")
	printOnly := fs.Bool("print", false, "")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 {
		return 1, apperr.New(apperr.InvalidArguments, "invalid install arguments")
	}
	path, snippet, err := shellhook.Install(shellhook.Options{Shell: *shellName, PrintOnly: *printOnly})
	if err != nil {
		return 1, err
	}
	if *printOnly {
		if g.json {
			writeResult(stdout, g, output.Result{Matched: true, Action: "none"})
			return 0, nil
		}
		fmt.Fprint(stdout, snippet)
		return 0, nil
	}
	res := output.Result{Matched: true, Action: "install_created"}
	if g.json {
		writeResult(stdout, g, res)
	} else {
		fmt.Fprintf(stdout, "installed: %s\n", path)
	}
	return 0, nil
}

func resolve(remoteFlag string, g globals, stderr io.Writer) (resolved, error) {
	cfg, _, err := config.Load()
	if err != nil {
		return resolved{}, err
	}
	remoteName := cfg.Defaults.Remote
	if remoteFlag != "" {
		remoteName = remoteFlag
	}
	raw, err := gitremote.RemoteURL(remoteName)
	if err != nil {
		return resolved{}, err
	}
	remote, err := parser.Parse(raw)
	if err != nil {
		return resolved{}, err
	}
	if g.verbose {
		fmt.Fprintf(stderr, "remote: %s\n", parser.MaskURL(raw))
		fmt.Fprintf(stderr, "normalized_remote: %s\n", remote.NormalizedURL)
	}
	return resolved{cfg: cfg, remote: remote, match: matcher.Resolve(cfg, remote)}, nil
}

func ensureSwitched(host, account string) (string, error) {
	st, err := ghadapter.StatusFor(host)
	if err != nil {
		return "", err
	}
	if err := ghadapter.EnsureAccount(st, account); err != nil {
		return "", err
	}
	if st.Active == account {
		return "none", nil
	}
	if err := ghadapter.Switch(host, account); err != nil {
		return "", err
	}
	st, err = ghadapter.StatusFor(host)
	if err != nil {
		return "", err
	}
	if st.Active != account {
		return "", apperr.New(apperr.GHSwitchFailed, "account did not become active after switch")
	}
	return "switched", nil
}

func resultFor(r resolved, action string) output.Result {
	res := output.Result{
		Host:    output.Ptr(r.remote.Host),
		Owner:   output.Ptr(r.remote.Owner),
		Repo:    output.Ptr(r.remote.Repo),
		Matched: r.match.Matched,
		Action:  action,
	}
	if r.remote.MaskedURLUser != "" {
		res.URLUser = output.Ptr(r.remote.MaskedURLUser)
	}
	if r.match.Matched {
		res.Account = output.Ptr(r.match.Rule.Account)
		res.Rule = output.Ptr(r.match.Rule.Name)
	}
	return res
}

func writeResult(stdout io.Writer, g globals, res output.Result) {
	if g.json {
		output.JSON(stdout, res)
	} else {
		output.HumanResult(stdout, res)
	}
}

func runChild(args []string, host string, stdout, stderr io.Writer) (int, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = procenv.WithGHHost(os.Environ(), host)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode(), nil
		}
		return 1, apperr.Wrap(apperr.ExecSpawnFailed, "could not start exec command", err)
	}
	return 0, nil
}

func indexOf(xs []string, s string) int {
	for i, x := range xs {
		if x == s {
			return i
		}
	}
	return -1
}

func shellQuoteValue(s string) string {
	return strings.ReplaceAll(s, "'", `'\''`)
}

func sortedKeys(m map[string]map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedAccountKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
