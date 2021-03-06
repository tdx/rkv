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

// Leader returns raft leader address
func (d *Backend) Leader() (string, string) {
	leaderIP := string(d.raft.Leader())
	for _, server := range d.servers {
		if server.IsLeader {
			return server.Host, leaderIP
		}
	}
	return "", ""
}

// Servers returns cluster servers
func (d *Backend) Servers() ([]*clusterApi.Server, error) {
	future := d.raft.GetConfiguration()
	if err := future.Error(); err != nil {
		return nil, err
	}

	var (
		servers []*clusterApi.Server
	)

	for _, server := range future.Configuration().Servers {
		var (
			ip       string
			host     string
			rpcPort  string
			raftPort string
			httpPort string
			serfAddr string
			isLeader bool
		)
		s, ok := d.servers[string(server.ID)]
		if ok {
			ip = s.IP
			host = s.Host
			rpcPort = s.RPCPort
			raftPort = s.RaftPort
			httpPort = s.HTTPBindAddr
			serfAddr = s.SerfBindAddr
			isLeader = s.IsLeader
		}

		srv := &clusterApi.Server{
			ID:           string(server.ID),
			IP:           ip,
			Host:         host,
			RPCPort:      rpcPort,
			RaftPort:     raftPort,
			HTTPBindAddr: httpPort,
			SerfBindAddr: serfAddr,
			IsLeader:     isLeader,
			Online:       ok == true,
		}
		servers = append(servers, srv)

		d.logger.Debug("cluster server info", "id", srv.ID,
			"host", srv.Host, "ip", srv.IP,
			"raft-port", srv.RaftPort, "rpc-port", srv.RPCPort,
			"http-addr", srv.HTTPBindAddr, "serf-addr", srv.SerfBindAddr,
			"isLeader", srv.IsLeader)
	}

	return servers, nil
}

func addrsEquals2(addr1, addr2 string) (bool, error) {
	h2, p2, err := net.SplitHostPort(addr2)
	if err != nil {
		return false, err
	}

	return addrsEquals3(addr1, h2, p2)
}

func addrsEquals3(addr1, h2, p2 string) (bool, error) {

	h1, p1, err := net.SplitHostPort(addr1)
	if err != nil {
		return false, err
	}

	if p1 != p2 {
		return false, nil
	}

	if h1 == "" || h1 == "::" || h1 == ":" || h1 == "[::]" {
		h1 = "127.0.0.1"
	}

	if h2 == "" || h2 == "::" || h2 == ":" || h2 == "[::]" {
		h2 = "127.0.0.1"
	}

	return h1 == h2, nil
}
