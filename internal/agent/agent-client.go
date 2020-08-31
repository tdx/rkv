package agent

import (
	"log"

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

// ExitCluster stop membership
func (a *Agent) ExitCluster() error {
	a.exitClusterLock.Lock()
	defer a.exitClusterLock.Unlock()

	if a.exitCluster {
		return nil
	}
	a.exitCluster = true

	a.logger.Trace("exit cluster")
	return a.membership.Leave()
}

//
func isLeader(db clusterApi.Backend) bool {
	return db.(clusterApi.Cluster).IsLeader()
}
