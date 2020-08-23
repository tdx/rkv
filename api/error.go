package api

import "errors"

// errors
var (
	ErrNodeNameEmpty    = errors.New("config NodeName is empty")
	ErrBackendEmpty     = errors.New("config Backend is empty")
	ErrNodeIsNotALeader = errors.New("node is not a leader")
	ErrInternalError    = errors.New("internal error")
)
