package raft

import (
	"fmt"
	"net"

	clusterApi "github.com/tdx/rkv/internal/cluster/api"
	"github.com/tdx/rkv/internal/discovery"

	"github.com/hashicorp/raft"
)

//
// discovery.Handler implementation
//

var _ discovery.Handler = (*Backend)(nil)

// Join ...
// func (d *Backend) Join(id, raftAddr, rpcAddr string, local bool) error {
func (d *Backend) Join(id string, tags map[string]string, local bool) error {

	var (
		rpcAddr  = tags["rpc_addr"]
		raftAddr = tags["raft_addr"]
		httpAddr = tags["http_addr"]
		serfAddr = tags["serf_addr"]
	)

	d.logger.Info("JOIN from", "id", id, "raft-addr", raftAddr, "rpc-addr",
		rpcAddr, "local", local)

	srv, ok := d.servers[id]
	if !ok {
		srv = &clusterApi.Server{
			ID: id,
		}
	}

	host, ip, raftPort, rpcPort, err := parseAddrs(raftAddr, rpcAddr)
	if err != nil {
		return err
	}

	oldRPCPort := srv.RPCPort
	srv.IP = ip
	srv.Host = host
	srv.RaftPort = raftPort
	srv.RPCPort = rpcPort
	srv.HTTPBindAddr = httpAddr
	srv.SerfBindAddr = serfAddr
	srv.IsLocal = local
	d.servers[id] = srv

	if oldRPCPort == "" && srv.IsLeader {
		d.leaderChanged(raft.ServerAddress(raftAddr))
	}

	for k, v := range d.servers {
		d.logger.Info("JOIN servers", "id", k, "host", v.Host, "ip", v.IP,
			"raft-port", v.RaftPort, "rpc-port", v.RPCPort,
			"isLeader", v.IsLeader)
	}

	if local {
		return nil
	}

	if !d.IsLeader() {
		d.logger.Debug(
			"JOIN from", "id", id, "raft-addr", raftAddr, "skip", "not leader")
		return nil
	}

	configFuture := d.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return err
	}

	serverID := raft.ServerID(id)
	serverAddr := raft.ServerAddress(raftAddr)
	servers := configFuture.Configuration().Servers

	d.logger.Debug("JOIN", "servers", servers)

	for _, srv := range servers {
		if srv.ID == serverID || srv.Address == serverAddr {
			if srv.ID == serverID && srv.Address == serverAddr {
				// server has already joined
				d.logger.Debug("JOIN from already joined", "id", id,
					"raft-addr", raftAddr)
				return nil
			}
			// remove old joined server with diff address
			//  (ex: k8s changed server address)
			d.logger.Debug("JOIN remove old joined", "id", id,
				"raft-addr", raftAddr)
			removeFuture := d.raft.RemoveServer(serverID, 0, 0)
			if err := removeFuture.Error(); err != nil {
				return err
			}
		}
	}

	d.logger.Debug("JOIN AddVoter", "id", id, "raft-addr", raftAddr)

	addFuture := d.raft.AddVoter(serverID, serverAddr, 0, 0)

	return addFuture.Error()
}

// Leave ...
// func (d *Backend) Leave(id, raftAddr string, local bool) error {
func (d *Backend) Leave(id string, tags map[string]string, local bool) error {

	raftAddr := tags["raft_addr"]

	server, ok := d.servers[id]
	delete(d.servers, id)

	d.logger.Info("LEAVE from", "id", id, "raft-addr", raftAddr, "local", local)

	for k, v := range d.servers {
		d.logger.Info("LEAVE servers", "id", k, "host", v.Host, "ip", v.IP,
			"raft-port", v.RaftPort, "rpc-port", v.RPCPort,
			"isLeader", v.IsLeader)
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
			"LEAVE from", "id", id, "raft-addr", raftAddr, "skip", "not leader")
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
		d.logger.Error("checkMembers GetConfiguration", "error", err)
		return
	}

	// find members not in cluster
	for _, server := range future.Configuration().Servers {
		_, ok := d.servers[string(server.ID)]
		if !ok {
			d.logger.Info("checkMembers remove Raft server",
				"id", server.ID, "addr", server.Address)
			removeFuture := d.raft.RemoveServer(server.ID, 0, 0)
			if err := removeFuture.Error(); err != nil {
				d.logger.Error("checkMembers remove Raft server",
					"id", server.ID, "addr", server.Address, "error", err)
			}
		}
	}

}

func parseAddrs(
	raftAddr, rpcAddr string) (string, string, string, string, error) {

	host, raftPort, err := net.SplitHostPort(raftAddr)
	if err != nil {
		return "", "", "", "", err
	}
	if host == "" {
		host = "localhost"
	}
	if host == ":" || host == "[::]" {
		host = "ip6-localhost"
	}
	ip, err := net.LookupHost(host)
	if err != nil {
		return "", "", "", "", err
	}
	if len(ip) == 0 {
		return "", "", "", "",
			fmt.Errorf("where are no ip addresses for hostname '%s'", host)
	}
	_, rpcPort, err := net.SplitHostPort(rpcAddr)
	if err != nil {
		return "", "", "", "", err
	}

	return host, ip[0], raftPort, rpcPort, nil
}
