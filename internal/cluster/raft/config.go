package raft

import (
	rkvApi "github.com/tdx/rkv/api"

	"github.com/hashicorp/raft"
)

// Config for distributed Db
type Config struct {
	Raft             raft.Config
	RPCAddr          string
	RaftAddr         string // hostname for bootstrap by name, not ip
	StreamLayer      *StreamLayer
	Bootstrap        bool
	ApplyRegistrator rkvApi.ApplyRegistrator
	OnLeaderChangeFn func(isLeader bool)
}

// ServerID returns raft.ServerID from string
func (c Config) ServerID(id string) raft.ServerID {
	return raft.ServerID(id)
}
