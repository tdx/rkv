package api

import (
	"io"

	hlog "github.com/hashicorp/go-hclog"
	"github.com/tdx/raft"
	dbApi "github.com/tdx/rkv/db/api"
)

// Config to create client
type Config struct {
	Raft     raft.Config
	NodeName string
	DataDir  string
	// Membership ip:port used by Serf to discover nodes and create cluster
	DiscoveryAddr string
	// DiscoveryJoinAddrs is empty for leader
	// For followers contains leader address [+running followers addresses]
	DiscoveryJoinAddrs []string
	RaftPort           int
	RPCPort            int
	HTTPAddr           string
	// LogLevel: error | warn | info | debug | trace
	LogLevel string
	// default stderr
	LogOutput          io.Writer
	LogIncludeLocation bool
	Logger             hlog.Logger
	// Backend
	Backend dbApi.Backend
}
