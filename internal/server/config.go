package server

import (
	log "github.com/hashicorp/go-hclog"
	clusterApi "github.com/tdx/rkv/internal/cluster/api"
)

// Joiner is an interface to join to cluster
type Joiner interface {
	Join(existing []string) (int, error)
}

// Config ...
type Config struct {
	Db     clusterApi.Backend
	Logger log.Logger
	Addr   string
	Joiner Joiner
}
