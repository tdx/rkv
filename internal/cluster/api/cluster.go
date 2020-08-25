package api

import (
	"time"
)

// Server describes cluster server
type Server struct {
	ID       string `json:"id"`
	IP       string `json:"ip"`
	Host     string `json:"host"`
	RaftPort string `json:"raft-port"`
	RPCPort  string `json:"rpc-port"`
	HTTPPort string `json:"http-port"`
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
