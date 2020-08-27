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

	logLevel := log.LevelFromString(config.LogLevel)
	if logLevel == log.NoLevel {
		logLevel = log.Error
	}
	logger := log.New(&log.LoggerOptions{
		Name:  fmt.Sprintf("agent-%s", config.NodeName),
		Level: logLevel,
	})
	config.Raft.LogLevel = config.LogLevel

	return agent.New(&agent.Config{
		Raft:           config.Raft,
		DataDir:        config.DataDir,
		Backend:        config.Backend,
		Logger:         logger,
		NodeName:       config.NodeName,
		BindAddr:       config.DiscoveryAddr,
		StartJoinAddrs: config.DiscoveryJoinAddrs,
		RaftPort:       config.RaftPort,
		RPCPort:        config.RPCPort,
		BindHTTP:       config.HTTPAddr,
		Bootstrap:      len(config.DiscoveryJoinAddrs) == 0,
	})
}
