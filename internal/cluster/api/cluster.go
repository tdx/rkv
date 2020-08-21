package api

import "time"

// Server describes cluster server
type Server struct {
	ID       string `json:"id"`
	RaftAddr string `json:"raft-addr"`
	RPCAddr  string `json:"rpc-addr"`
	IsLeader bool   `json:"leader"`
	Online   bool   `json:"online"`
}

// Cluster interface for distributed service
type Cluster interface {
	WaitForLeader(time.Duration) error
	IsLeader() bool
	LeaderAddr() string
	Servers() ([]*Server, error)
}
