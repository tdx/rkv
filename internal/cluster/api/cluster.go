package api

import (
	"time"
)

// Server describes cluster server
type Server struct {
	ID           string `json:"id"`
	IP           string `json:"ip"`
	Host         string `json:"host"`
	RaftPort     string `json:"raft-port"`
	RPCPort      string `json:"rpc-port"`
	HTTPBindAddr string `json:"http-bind-addr"`
	SerfBindAddr string `json:"serf-bind-addr"`
	IsLeader     bool   `json:"leader"`
	Online       bool   `json:"online"`
	IsLocal      bool   `json:"local"`
}

// Cluster interface for distributed service
type Cluster interface {
	WaitForLeader(time.Duration) error
	IsLeader() bool
	Leader() (host, ip string)
	Servers() ([]*Server, error)
	Restarted() bool
}
