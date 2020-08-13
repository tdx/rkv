package raft

import (
	"github.com/tdx/rkv/internal/discovery"

	"github.com/hashicorp/raft"
)

//
// discovery.Handler implementation
//

var _ discovery.Handler = (*Backend)(nil)

// Join ...
func (d *Backend) Join(id, addr string) error {

	d.logger.Debug("JOIN from", "id", id, "addr", addr)

	if !d.IsLeader() {
		d.logger.Debug(
			"JOIN from", "id", id, "addr", addr, "skip", "not leader")
		return nil
	}

	configFuture := d.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return err
	}

	serverID := raft.ServerID(id)
	serverAddr := raft.ServerAddress(addr)
	servers := configFuture.Configuration().Servers

	d.logger.Debug("JOIN", "servers", servers)

	for _, srv := range servers {
		if srv.ID == serverID || srv.Address == serverAddr {
			if srv.ID == serverID && srv.Address == serverAddr {
				// server has already joined
				d.logger.Debug("JOIN from already joined", "id", id, "addr", addr)
				return nil
			}
			// remove the existing server
			d.logger.Debug("JOIN remove already joined", "id", id, "addr", addr)
			removeFuture := d.raft.RemoveServer(serverID, 0, 0)
			if err := removeFuture.Error(); err != nil {
				return err
			}
		}
	}

	d.logger.Debug("JOIN AddVoter", "id", id, "addr", addr)

	addFuture := d.raft.AddVoter(serverID, serverAddr, 0, 0)
	return addFuture.Error()
}

// Leave ...
func (d *Backend) Leave(id, addr string) error {

	d.logger.Debug("LEAVE from", "id", id, "addr", addr)

	if !d.IsLeader() {
		d.logger.Debug(
			"LEAVE from", "id", id, "addr", addr, "skip", "not leader")
		return nil
	}

	d.logger.Debug("LEAVE RemoveServer", "id", id)

	removeFuture := d.raft.RemoveServer(raft.ServerID(id), 0, 0)
	return removeFuture.Error()
}
