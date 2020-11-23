package rkv

import (
	"fmt"

	"github.com/tdx/rkv/api"
	"github.com/tdx/rkv/internal/agent"
	"github.com/tdx/rkv/registry"

	hlog "github.com/hashicorp/go-hclog"
)

// NewClient creates rkv client
func NewClient(config *api.Config) (api.Client, error) {
	if config.NodeName == "" {
		return nil, api.ErrNodeNameEmpty
	}

	if config.Logger == nil {
		logLevel := hlog.LevelFromString(config.LogLevel)
		if logLevel == hlog.NoLevel {
			logLevel = hlog.Info
		}
		config.Logger = hlog.New(&hlog.LoggerOptions{
			Name:            fmt.Sprintf("rkv-%s", config.NodeName),
			Level:           logLevel,
			IncludeLocation: config.LogIncludeLocation,
			Output:          config.LogOutput,
		})
	}
	config.Raft.LogLevel = config.LogLevel

	reg := config.Registry
	if reg == nil {
		reg = registry.NewApplyRegistrator()
	}

	return agent.New(&agent.Config{
		Raft:             config.Raft,
		DataDir:          config.DataDir,
		Backend:          config.Backend,
		Logger:           config.Logger,
		NodeName:         config.NodeName,
		BindAddr:         config.DiscoveryAddr,
		StartJoinAddrs:   config.DiscoveryJoinAddrs,
		RaftPort:         config.RaftPort,
		RPCPort:          config.RPCPort,
		BindHTTP:         config.HTTPAddr,
		Bootstrap:        len(config.DiscoveryJoinAddrs) == 0,
		OnLeaderChangeFn: config.OnChangeLeaderFn,
		Registry:         reg,
	})
}
