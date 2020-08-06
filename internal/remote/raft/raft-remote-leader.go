package raft

import "github.com/hashicorp/raft"

// IsLeader returns if raft is leader
func (d *Backend) IsLeader() bool {
	return d.raft.State() == raft.Leader
}
