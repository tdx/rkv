package api

import (
	"io"

	hlog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	dbApi "github.com/tdx/rkv/db/api"
)

// DefaultTimeFormat for logger
const DefaultTimeFormat = "2006-01-02 15:04:05.000000"

// Config to create client
type Config struct {
	Raft     raft.Config
	NodeName string
	DataDir  string
	// Backend
	Backend dbApi.Backend
	// Membership ip:port used by Serf to discover nodes and create cluster
	DiscoveryAddr string
	// DiscoveryJoinAddrs is empty for leader
	// For followers contains leader address [+running followers addresses]
	DiscoveryJoinAddrs []string
	RaftPort           int
	RPCPort            int
	HTTPAddr           string
	// Log options
	Logger hlog.Logger
	// LogLevel: error | warn | info | debug | trace
	LogLevel string
	// default stderr
	LogOutput          io.Writer
	LogIncludeLocation bool
	LotTimeFormat      string
}
