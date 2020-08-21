package server

import (
	log "github.com/hashicorp/go-hclog"
	clusterApi "github.com/tdx/rkv/internal/cluster/api"
)

// Config ...
type Config struct {
	Db     clusterApi.Backend
	Logger log.Logger
	Addr   string
}
