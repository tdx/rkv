package api

import "time"

// Server describes cluster server
type Server struct {
	ID       string
	RPCAddr  string
	IsLeader bool
}

// Cluster interface for distributed service
type Cluster interface {
	WaitForLeader(time.Duration) error
	IsLeader() bool
	LeaderAddr() string
	Servers() ([]*Server, error)
}
