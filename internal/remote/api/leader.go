package api

import "time"

// Leader interface for distributed service
type Leader interface {
	WaitForLeader(time.Duration) error
	IsLeader() bool
}
