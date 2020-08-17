package agent

import (
	"github.com/tdx/rkv/api"
	clusterApi "github.com/tdx/rkv/internal/cluster/api"
)

var _ api.Client = (*Agent)(nil)

// Put ...
func (a *Agent) Put(tab, key, value []byte) error {
	if isLeader(a.raftDb) {
		return a.raftDb.Put(tab, key, value)
	}
	// TODO: rpc call
	return api.ErrNodeIsNotALeader
}

// Get ...
func (a *Agent) Get(
	lvl api.ConsistencyLevel, tab, key []byte) ([]byte, error) {

	return a.raftDb.Get(lvl, tab, key)
}

// Delete ...
func (a *Agent) Delete(tab, key []byte) error {
	if isLeader(a.raftDb) {
		return a.raftDb.Delete(tab, key)
	}
	// TODO: rpc call
	return api.ErrNodeIsNotALeader
}

//
func isLeader(db clusterApi.Backend) bool {
	return db.(clusterApi.Cluster).IsLeader()
}
