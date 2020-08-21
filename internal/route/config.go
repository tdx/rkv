package route

import (
	log "github.com/hashicorp/go-hclog"
)

// Config is a route svc config
type Config struct {
	Name   string
	Logger log.Logger
}
