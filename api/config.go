package api

import (
	dbApi "github.com/tdx/rkv/db/api"

	"github.com/hashicorp/raft"
)

// Config to create client
type Config struct {
	// Raft config
	Raft raft.Config
	// NodeName
	NodeName string
	// Membership ip:port used by Serf to discover nodes and create cluster
	DiscoveryAddr string
	// DiscoveryJoinAddrs is empty for leader
	// For followers contains leader address [+running followers addresses]
	DiscoveryJoinAddrs []string
	// Distributed database port
	DistributedPort int
	// LogLevel: error | warn | info | debug | trace
	LogLevel string
	// Backend
	Backend dbApi.Backend
}
