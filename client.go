package rkv

import (
	"fmt"

	"rkv/api"
	"rkv/internal/agent"

	log "github.com/hashicorp/go-hclog"
)

// NewClient creates rkv client
func NewClient(config *api.Config) (api.Client, error) {
	if config.NodeName == "" {
		return nil, api.ErrNodeNameEmpty
	}

	logLevel := log.LevelFromString(config.LogLevel)

	logger := log.New(&log.LoggerOptions{
		Name:  fmt.Sprintf("agent-%s", config.NodeName),
		Level: logLevel,
	})

	return agent.New(agent.Config{
		Logger:         logger,
		NodeName:       config.NodeName,
		BindAddr:       config.DiscoveryAddr,
		RaftPort:       config.DistributedPort,
		DataDir:        config.DataDir,
		StartJoinAddrs: config.DiscoveryJoinAddrs,
		Bootstrap:      len(config.DiscoveryJoinAddrs) == 0,
	})
}
