package discovery

import (
	log "github.com/hashicorp/go-hclog"
)

// Config ...
type Config struct {
	Logger         log.Logger
	DataDir        string
	NodeName       string
	BindAddr       string
	Tags           map[string]string
	StartJoinAddrs []string
}
