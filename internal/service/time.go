package service

import (
	"time"

	"tv-tracker/internal/timeutil"
)

// Deprecated: use timeutil.SetNowFunc instead.
func SetNowFunc(fn func() time.Time) {
	timeutil.SetNowFunc(fn)
}
