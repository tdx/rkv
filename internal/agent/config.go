package agent

import (
	"fmt"
	"net"

	log "github.com/hashicorp/go-hclog"
)

// Config ...
type Config struct {
	Logger   log.Logger
	BindAddr string // Serf, Raft, RPC address. Serf with port
	RPCPort  int    // rpc API server
	RaftPort int
	DataDir  string
	NodeName string // Raft server ID
	// Bootstrap should be set to true when starting the first node
	//  of the cluster
	StartJoinAddrs []string
	Bootstrap      bool
}

// RPCAddr returns host:port
func (c Config) RPCAddr() (string, error) {
	host, _, err := net.SplitHostPort(c.BindAddr)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", host, c.RPCPort), nil
}

// RaftAddr returns host:port
func (c Config) RaftAddr() (string, error) {
	host, _, err := net.SplitHostPort(c.BindAddr)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", host, c.RaftPort), nil
}
