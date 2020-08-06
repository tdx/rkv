package raft

import (
	"github.com/hashicorp/raft"
)

// Config for distributed Db
type Config struct {
	Raft struct {
		raft.Config
		BindAddr    string
		StreamLayer *StreamLayer
		Bootstrap   bool
	}
}

// ServerID returns raft.ServerID from string
func (c Config) ServerID(id string) raft.ServerID {
	return raft.ServerID(id)
}
