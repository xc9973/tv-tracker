package timeutil

import "time"

var nowFunc = time.Now

// Now returns the current time. It is wrapped to simplify testing and
// allow centralized timezone handling for web and scheduler components.
func Now() time.Time {
	return nowFunc()
}

// SetNowFunc overrides the function used by Now. Passing nil resets it.
func SetNowFunc(fn func() time.Time) {
	if fn == nil {
		nowFunc = time.Now
		return
	}
	nowFunc = fn
}
