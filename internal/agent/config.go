package agent

import (
	"fmt"
	"io"
	"net"

	rkvApi "github.com/tdx/rkv/api"
	dbApi "github.com/tdx/rkv/db/api"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
)

// Config ...
type Config struct {
	DataDir  string
	Raft     raft.Config
	BindAddr string // Serf, Raft, RPC address. Serf with port
	RPCPort  int    // rpc API server
	RaftPort int
	BindHTTP string // bind address for HTTP server
	Backend  dbApi.Backend
	NodeName string // Raft server ID
	// Bootstrap should be set to true when starting the first node
	//  of the cluster
	StartJoinAddrs []string
	Bootstrap      bool
	RoutingPolicy  rkvApi.RoutingPolicy
	// Log oprtions: passed configured logger
	Logger log.Logger
	// Or configure options
	// LogLevel: error | warn | info | debug | trace
	LogLevel string
	// LogOutput default - stderr
	LogOutput          io.Writer
	LogIncludeLocation bool
	LogTimeFormat      string
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
