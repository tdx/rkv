package api

import (
	"time"

	dbApi "github.com/tdx/rkv/db/api"

	"github.com/hashicorp/raft"
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
	// Backend
	Backend dbApi.Backend
	// Used in k8s. Delay to allow balancer remove traffice from endpoint
	ShutdownDelay time.Duration
}
