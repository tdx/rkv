package api

import "errors"

var (
	// ErrNodeNameEmpty ...
	ErrNodeNameEmpty = errors.New("config NodeName is empty")
	// ErrBackendEmpty ...
	ErrBackendEmpty = errors.New("config Backend is empty")
	// ErrNodeIsNotALeader for write operation on not leader node
	ErrNodeIsNotALeader = errors.New("node is not a leader")
)
