package server

import "time"

// timeAfterFunc is a thin wrapper for testability.
func timeAfterFunc(d time.Duration, f func()) *time.Timer {
	return time.AfterFunc(d, f)
}
