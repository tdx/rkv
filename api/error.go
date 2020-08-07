package api

import "errors"

var (
	// ErrNodeNameEmpty ...
	ErrNodeNameEmpty = errors.New("config NodeName is empty")
	// ErrNodeIsNotALeader for write operation on not leader node
	ErrNodeIsNotALeader = errors.New("node is not aleader")
)
