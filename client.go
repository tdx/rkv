package rkv

import (
	"fmt"

	"github.com/tdx/rkv/api"
	"github.com/tdx/rkv/internal/agent"

	log "github.com/hashicorp/go-hclog"
)

// NewClient creates rkv client
func NewClient(config *api.Config) (api.Client, error) {
	if config.NodeName == "" {
		return nil, api.ErrNodeNameEmpty
	}

	logger := log.New(&log.LoggerOptions{
		Name:  fmt.Sprintf("agent-%s", config.NodeName),
		Level: log.LevelFromString(config.LogLevel),
	})

	return agent.New(&agent.Config{
		Logger:         logger,
		NodeName:       config.NodeName,
		BindAddr:       config.DiscoveryAddr,
		RaftPort:       config.DistributedPort,
		Backend:        config.Backend,
		StartJoinAddrs: config.DiscoveryJoinAddrs,
		Bootstrap:      len(config.DiscoveryJoinAddrs) == 0,
	})
}
