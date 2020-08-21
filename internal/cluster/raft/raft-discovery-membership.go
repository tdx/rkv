package raft

import (
	clusterApi "github.com/tdx/rkv/internal/cluster/api"
	"github.com/tdx/rkv/internal/discovery"

	"github.com/hashicorp/raft"
)

//
// discovery.Handler implementation
//

var _ discovery.Handler = (*Backend)(nil)

// Join ...
func (d *Backend) Join(id, addr, rpcAddr string, local bool) error {

	srv, ok := d.servers[id]
	if !ok {
		srv = &clusterApi.Server{
			ID: id,
		}
	}
	oldRPCAddr := srv.RPCAddr
	srv.RaftAddr = addr
	srv.RPCAddr = rpcAddr
	d.servers[id] = srv

	if oldRPCAddr == "" && srv.IsLeader {
		d.leaderChanged(raft.ServerAddress(srv.RaftAddr))
	}

	d.logger.Info("JOIN from", "id", id, "addr", addr, "local", local)

	for k, v := range d.servers {
		d.logger.Info("JOIN servers", "id", k, "raftAddr", v.RaftAddr,
			"rpcAddr", v.RPCAddr, "isLeader", v.IsLeader)
	}

	if local {
		return nil
	}

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
func (d *Backend) Leave(id, addr string, local bool) error {

	server, ok := d.servers[id]
	delete(d.servers, id)

	d.logger.Info("LEAVE from", "id", id, "addr", addr, "local", local)

	for k, v := range d.servers {
		d.logger.Info("LEAVE servers", "id", k, "raftAddr", v.RaftAddr,
			"rpcAddr", v.RPCAddr, "isLeader", v.IsLeader)
	}

	if local {
		return nil
	}

	if ok {
		d.logger.Info("LEAVE from", "id", id, "was-leader", server.IsLeader)
		if server.IsLeader {
			d.closeLeaderCon()
		}
	}

	if !d.IsLeader() {
		d.logger.Debug(
			"LEAVE from", "id", id, "addr", addr, "skip", "not leader")
		return nil
	}

	d.logger.Debug("LEAVE RemoveServer", "id", id)

	removeFuture := d.raft.RemoveServer(raft.ServerID(id), 0, 0)

	return removeFuture.Error()
}

//
func (d *Backend) checkMembers() {

	future := d.raft.GetConfiguration()
	if err := future.Error(); err != nil {
		d.logger.Error("checkMembers", "GetConfiguration", err)
		return
	}

	// find members not in cluster
	for _, server := range future.Configuration().Servers {
		_, ok := d.servers[string(server.ID)]
		if !ok {
			d.logger.Info("checkMembers", "remove-server", server.Address)
			removeFuture := d.raft.RemoveServer(server.ID, 0, 0)
			if err := removeFuture.Error(); err != nil {
				d.logger.Error("checkMembers",
					"remove-server", server.Address, "error", err)
			}
		}
	}

}
