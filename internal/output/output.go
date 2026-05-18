package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/xpadev-net/gh-auto-switch/internal/apperr"
)

type Result struct {
	Host    *string `json:"host"`
	Owner   *string `json:"owner"`
	Repo    *string `json:"repo"`
	URLUser *string `json:"url_user"`
	Account *string `json:"account"`
	Rule    *string `json:"rule"`
	Matched bool    `json:"matched"`
	Action  string  `json:"action"`
}

func Ptr(s string) *string { return &s }

func JSON(w io.Writer, v any) {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func JSONError(w io.Writer, e *apperr.Error) {
	JSON(w, map[string]any{
		"error": map[string]any{
			"code":    string(e.Code),
			"message": e.Message,
		},
	})
}

func HumanResult(w io.Writer, r Result) {
	if r.Host != nil {
		fmt.Fprintf(w, "host: %s\n", *r.Host)
	}
	if r.Owner != nil {
		fmt.Fprintf(w, "owner: %s\n", *r.Owner)
	}
	if r.Repo != nil {
		fmt.Fprintf(w, "repo: %s\n", *r.Repo)
	}
	if r.URLUser != nil {
		fmt.Fprintf(w, "url_user: %s\n", *r.URLUser)
	}
	if r.Account != nil {
		fmt.Fprintf(w, "account: %s\n", *r.Account)
	}
	if r.Rule != nil {
		fmt.Fprintf(w, "rule: %s\n", *r.Rule)
	}
	fmt.Fprintf(w, "matched: %t\n", r.Matched)
	fmt.Fprintf(w, "action: %s\n", r.Action)
}
