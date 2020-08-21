package raft

import (
	"fmt"
	"time"

	rpcApi "github.com/tdx/rkv/internal/rpc/v1"

	"github.com/hashicorp/raft"
	"google.golang.org/grpc"
)

//
func (d *Backend) waitEvents() {
	timeout := 15 * time.Second
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		if d.grpcLeaderConn == nil {
			timeout = time.Second
		} else {
			timeout = 15 * time.Second
		}
		timer.Reset(timeout)

		select {
		case o, ok := <-d.observationCh:
			if !ok {
				return
			}
			switch r := o.Data.(type) {
			case raft.LeaderObservation:
				d.leaderChanged(r.Leader)
			}
		case _, ok := <-timer.C:
			if !ok {
				d.logger.Error("waitEvents timer channel errored")
				return
			}
			d.leaderCheck()
		}
	}
}

//
func (d *Backend) leaderChanged(leader raft.ServerAddress) {

	leaderAddr := d.raft.Leader()

	d.logger.Info("leaderChanged", "event-leader", leader,
		"raft-leader", leaderAddr, "cur-rpc-leader-addr", d.rpcLeaderAddr)

	var rpcLeaderAddr string
	for _, server := range d.servers {
		ok, _ := addrsEquals(string(leaderAddr), server.RaftAddr)
		if ok {
			server.IsLeader = true
			rpcLeaderAddr = server.RPCAddr
			continue
		}
		if server.IsLeader {
			server.IsLeader = false
		}
	}

	if rpcLeaderAddr == "" {
		d.logger.Error("leaderChanged", "event-leader", leader,
			"raft-leader", leaderAddr, "error", "empty leader RPCAddr")
		return
	}

	// we are the leader
	if d.raft.State() == raft.Leader {

		d.checkMembers()
		d.closeLeaderCon()

		return
	}

	if d.rpcLeaderAddr == rpcLeaderAddr {
		return
	}

	d.closeLeaderCon()

	d.logger.Info("leaderChanged: setup route to leader",
		"event-leader", leader, "raft-leader", leaderAddr,
		"rpc-leader-addr", rpcLeaderAddr)

	selfNodeName := string(d.config.Raft.LocalID)

	clientOpts := []grpc.DialOption{grpc.WithInsecure()}
	conn, err := grpc.Dial(fmt.Sprintf(
		"%s:///%s", selfNodeName, rpcLeaderAddr),
		clientOpts...,
	)
	if err != nil {
		d.logger.Error("leaderChanged",
			"event-leader", leader, "raft-leader", leaderAddr,
			"rpc-leader-addr", rpcLeaderAddr,
			"grpc-leader-conn", err)
		return
	}

	d.rpcLeaderAddr = rpcLeaderAddr
	d.grpcLeaderConn = conn
	d.leaderConn = rpcApi.NewStorageClient(conn)
}

func (d *Backend) leaderCheck() {
	// send current leader to balancer routine
	// currentLeader(d.LeaderAddr())
}

func (d *Backend) closeLeaderCon() {
	// close previouse connection to leader
	if d.grpcLeaderConn != nil {
		d.logger.Debug("closeLeaderCon", "rpc-leader", d.rpcLeaderAddr)
		d.grpcLeaderConn.Close()
		d.grpcLeaderConn = nil
		d.leaderConn = nil
		d.rpcLeaderAddr = ""
	}

}
