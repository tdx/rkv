package raft

import (
	"fmt"
	"net"
	"time"

	"github.com/hashicorp/raft"
	clusterApi "github.com/tdx/rkv/internal/cluster/api"
)

var _ clusterApi.Cluster = (*Backend)(nil)

// WaitForLeader ...
func (d *Backend) WaitForLeader(timeout time.Duration) error {
	timeoutc := time.After(timeout)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-timeoutc:
			return fmt.Errorf("wait for leader timed out")
		case <-ticker.C:
			l := d.raft.Leader()
			d.logger.Info("wait for leader", "leader", l)
			if l != "" {
				return nil
			}
		}
	}
}

// IsLeader returns if raft is leader
func (d *Backend) IsLeader() bool {
	return d.raft.State() == raft.Leader
}

// LeaderAddr returns raft leader address
func (d *Backend) LeaderAddr() string {
	return string(d.raft.Leader())
}

// Servers returns cluster servers
func (d *Backend) Servers() ([]*clusterApi.Server, error) {
	future := d.raft.GetConfiguration()
	if err := future.Error(); err != nil {
		return nil, err
	}

	var (
		leaderAddr = string(d.raft.Leader())
		servers    []*clusterApi.Server
	)

	for _, server := range future.Configuration().Servers {
		isLeader, err := addrsEquals(leaderAddr, string(server.Address))
		if err != nil {
			return nil, err
		}

		var rpcAddr string
		s, ok := d.servers[string(server.ID)]
		if ok {
			rpcAddr = s.RPCAddr
		}

		srv := &clusterApi.Server{
			ID:       string(server.ID),
			RaftAddr: string(server.Address),
			RPCAddr:  rpcAddr,
			IsLeader: isLeader,
			Online:   ok == true,
		}
		servers = append(servers, srv)

		d.logger.Debug("cluster server info", "id", srv.ID,
			"raftAddr", srv.RaftAddr, "rpcAddr", srv.RPCAddr,
			"isLeader", srv.IsLeader)
	}

	return servers, nil
}

func addrsEquals(addr1, addr2 string) (bool, error) {
	h1, p1, err := net.SplitHostPort(addr1)
	if err != nil {
		return false, err
	}

	h2, p2, err := net.SplitHostPort(addr2)
	if err != nil {
		return false, err
	}

	if p1 != p2 {
		return false, nil
	}

	if (h1 == "" && (h2 == "" || h2 == "::")) ||
		(h2 == "" && (h1 == "" || h1 == "::")) {

		return true, nil
	}

	return h1 == h2, nil
}
