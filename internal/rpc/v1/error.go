package kv_v1

import (
	"google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

//
// ErrNotALeader error
//
type ErrNotALeader struct{}

// GRPCStatus used by go-rpc to construct error answer
func (e ErrNotALeader) GRPCStatus() *status.Status {
	return status.New(codes.Unavailable, "node is not a leader")
}

func (e ErrNotALeader) Error() string {
	return e.GRPCStatus().Err().Error()
}

//
// ErrNoTable error
//
type ErrNoTable struct{}

// GRPCStatus used by go-rpc to construct error answer
func (e ErrNoTable) GRPCStatus() *status.Status {
	return status.New(codes.NotFound, "table does not exist")
}

func (e ErrNoTable) Error() string {
	return e.GRPCStatus().Err().Error()
}

//
// ErrNoKey error
//
type ErrNoKey struct{}

// GRPCStatus ...
func (e ErrNoKey) GRPCStatus() *status.Status {
	return status.New(codes.NotFound, "key  does not exist")
}

func (e ErrNoKey) Error() string {
	return e.GRPCStatus().Err().Error()
}
