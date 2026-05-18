package apperr

import "fmt"

type Code string

const (
	NotGitRepository     Code = "not_git_repository"
	RemoteNotFound       Code = "remote_not_found"
	RemoteURLUnparseable Code = "remote_url_unparseable"
	ConfigNotFound       Code = "config_not_found"
	ConfigInvalid        Code = "config_invalid"
	UnmatchedRule        Code = "unmatched_rule"
	TokenEnvPresent      Code = "token_env_present"
	HostUnauthenticated  Code = "host_unauthenticated"
	AccountUnavailable   Code = "account_unavailable"
	GHNotFound           Code = "gh_not_found"
	GHSwitchFailed       Code = "gh_switch_failed"
	LockTimeout          Code = "lock_timeout"
	ExecCommandEmpty     Code = "exec_command_empty"
	ExecSpawnFailed      Code = "exec_spawn_failed"
	InvalidArguments     Code = "invalid_arguments"
	InternalError        Code = "internal_error"
)

type Error struct {
	Code    Code
	Message string
	Err     error
}

func New(code Code, msg string) *Error {
	return &Error{Code: code, Message: msg}
}

func Wrap(code Code, msg string, err error) *Error {
	return &Error{Code: code, Message: msg, Err: err}
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func ExitCode(code Code) int {
	switch code {
	case ConfigNotFound, ConfigInvalid:
		return 2
	case TokenEnvPresent, HostUnauthenticated, AccountUnavailable, GHSwitchFailed:
		return 3
	case UnmatchedRule:
		return 4
	default:
		return 1
	}
}

func From(err error) *Error {
	if err == nil {
		return nil
	}
	if e, ok := err.(*Error); ok {
		return e
	}
	return Wrap(InternalError, "internal error", err)
}
