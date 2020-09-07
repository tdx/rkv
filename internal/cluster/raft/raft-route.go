package raft

import (
	"fmt"
	"net"

	clusterApi "github.com/tdx/rkv/internal/cluster/api"
	rpcApi "github.com/tdx/rkv/internal/rpc/v1"

	"github.com/hashicorp/raft"
	"google.golang.org/grpc"
)

//
func (d *Backend) waitEvents() {
	for o := range d.observationCh {
		switch r := o.Data.(type) {
		case raft.LeaderObservation:
			d.leaderChanged(r.Leader)
		}
	}
}

//
func (d *Backend) leaderChanged(leader raft.ServerAddress) {

	leaderAddr := d.raft.Leader()

	d.logger.Info("leaderChanged", "event-leader-addr", leader,
		"raft-leader-addr", leaderAddr, "cur-rpc-leader-addr", d.rpcLeaderAddr)

	var leaderServer *clusterApi.Server
	for _, server := range d.servers {
		ok, _ := addrsEquals3(string(leaderAddr), server.IP, server.RaftPort)
		if ok {
			server.IsLeader = true
			leaderServer = server
			continue
		}
		if server.IsLeader {
			server.IsLeader = false
		}
	}

	for k, v := range d.servers {
		d.logger.Info("leaderChanged server",
			"id", k, "host", v.Host, "ip", v.IP,
			"raft-port", v.RaftPort, "rpc-port", v.RPCPort,
			"isLeader", v.IsLeader)
	}

	if leaderServer == nil {
		d.logger.Error("leaderChanged", "event-leader-addr", leader,
			"raft-leader-addr", leaderAddr, "error", "empty leader RPCAddr")
		return
	}

	// we are the leader
	if d.raft.State() == raft.Leader {

		d.checkMembers()
		d.closeLeaderCon()

		return
	}

	rpcLeaderAddr := net.JoinHostPort(leaderServer.IP, leaderServer.RPCPort)

	if d.rpcLeaderAddr == rpcLeaderAddr {
		return
	}

	d.closeLeaderCon()

	d.logger.Info("leaderChanged: setup route to leader",
		"event-leader", leader, "raft-leader-addr", leaderAddr,
		"rpc-leader-addr", rpcLeaderAddr)

	selfNodeName := string(d.config.Raft.LocalID)

	clientOpts := []grpc.DialOption{grpc.WithInsecure()}
	conn, err := grpc.Dial(fmt.Sprintf(
		"%s:///%s", selfNodeName, rpcLeaderAddr),
		clientOpts...,
	)
	if err != nil {
		d.logger.Error("leaderChanged",
			"event-leader", leader, "raft-leader-addr", leaderAddr,
			"rpc-leader-addr", rpcLeaderAddr,
			"grpc-leader-conn", err)
		return
	}

	d.rpcLeaderAddr = rpcLeaderAddr
	d.grpcLeaderConn = conn
	d.leaderConn = rpcApi.NewStorageClient(conn)
}

func (d *Backend) closeLeaderCon() {
	// close previouse connection to leader
	if d.grpcLeaderConn != nil {
		d.logger.Debug("closeLeaderCon", "rpc-leader-addr", d.rpcLeaderAddr)
		d.grpcLeaderConn.Close()
		d.grpcLeaderConn = nil
		d.leaderConn = nil
		d.rpcLeaderAddr = ""
	}

}
