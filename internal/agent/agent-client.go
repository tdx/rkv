package agent

import (
	"context"
	"log"
	"time"

	"github.com/tdx/rkv/api"
	clusterApi "github.com/tdx/rkv/internal/cluster/api"

	hlog "github.com/hashicorp/go-hclog"
)

var _ api.Client = (*Agent)(nil)

// Put ...
func (a *Agent) Put(tab, key, value []byte) error {
	if a.Config.RoutingPolicy == api.RouteToLeader || isLeader(a.raftDb) {
		return a.raftDb.Put(tab, key, value)
	}

	return api.ErrNodeIsNotALeader
}

// Get ...
func (a *Agent) Get(
	lvl api.ConsistencyLevel, tab, key []byte) ([]byte, error) {

	return a.raftDb.Get(lvl, tab, key)
}

// Delete ...
func (a *Agent) Delete(tab, key []byte) error {
	if a.Config.RoutingPolicy == api.RouteToLeader || isLeader(a.raftDb) {
		return a.raftDb.Delete(tab, key)
	}

	return api.ErrNodeIsNotALeader
}

// Logger for client use
func (a *Agent) Logger(subSystem string) *log.Logger {
	out := a.logger.Named(subSystem).StandardWriter(&hlog.StandardLoggerOptions{
		InferLevels: true,
	})
	return log.New(out, "", 0)
}

// Shutdown ...
func (a *Agent) Shutdown() error {
	a.shutdownLock.Lock()
	defer a.shutdownLock.Unlock()

	if a.shutdown {
		return nil
	}
	a.shutdown = true

	a.logger.Trace("shutdown")

	shutdown := []func() error{
		func() error {
			if a.grpcServer == nil {
				return nil
			}
			a.grpcServer.GracefulStop()
			return nil
		},
		func() error {
			if a.httpServer == nil {
				return nil
			}
			ctx, cancel := context.WithTimeout(
				context.Background(), 5*time.Second)
			defer func() {
				cancel()
			}()
			return a.httpServer.Shutdown(ctx)
		},
		a.raftDb.Close,
		a.membership.Leave,
	}

	for _, fn := range shutdown {
		if err := fn(); err != nil {
			return err
		}
	}

	return nil
}

// Leader ...
func (a *Agent) Leader() (string, string) {
	return a.raftDb.(clusterApi.Cluster).Leader()
}

// IsLeader ...
func (a *Agent) IsLeader() bool {
	return a.raftDb.(clusterApi.Cluster).IsLeader()
}

//
func isLeader(db clusterApi.Backend) bool {
	return db.(clusterApi.Cluster).IsLeader()
}
