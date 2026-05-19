//go:build !darwin && !linux

package lock

import (
	"time"

	"github.com/xpadev-net/gh-auto-switch/internal/apperr"
)

type Lock struct{}

func AcquireAuthStore(timeout time.Duration) (*Lock, error) {
	return nil, apperr.New(apperr.InternalError, "auth store locks are unsupported on this platform")
}

func (l *Lock) Release() {}
