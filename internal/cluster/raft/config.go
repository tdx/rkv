package raft

import (
	rkvApi "github.com/tdx/rkv/api"

	"github.com/hashicorp/raft"
)

// Config for distributed Db
type Config struct {
	Raft             raft.Config
	RPCAddr          string
	StreamLayer      *StreamLayer
	Bootstrap        bool
	ApplyRegistrator rkvApi.ApplyRegistrator
}

// ServerID returns raft.ServerID from string
func (c Config) ServerID(id string) raft.ServerID {
	return raft.ServerID(id)
}
