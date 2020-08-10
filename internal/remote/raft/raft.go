package raft

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	dbApi "github.com/tdx/rkv/db/api"
	remoteApi "github.com/tdx/rkv/internal/remote/api"

	"github.com/gogo/protobuf/proto"
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

const (
	raftLogCacheSize = 512
)

var _ remoteApi.Backend = (*Backend)(nil)

// Backend ...
type Backend struct {
	logger log.Logger
	config Config
	l      sync.RWMutex
	fsm    *fsm

	raft        *raft.Raft
	snapStore   raft.SnapshotStore
	logStore    raft.LogStore
	stableStore raft.StableStore
	dataDir     string
}

// New ...
func New(
	db dbApi.Backend,
	config Config) (*Backend, error) {

	logger := config.Raft.Config.Logger

	if logger == nil {
		logger = log.New(&log.LoggerOptions{
			Name:  fmt.Sprintf("raft-%s", config.Raft.LocalID),
			Level: log.Debug,
		})
		config.Raft.Config.Logger = logger
	}

	d := &Backend{
		logger: logger,
		config: config,
		fsm: &fsm{
			id:     string(config.Raft.LocalID),
			logger: logger.Named("fsm"),
			db:     db,
		},
	}

	dir := filepath.Dir(db.DSN())

	if err := d.setupRaft(dir); err != nil {
		return nil, err
	}

	return d, nil
}

// Addr returns raft transport address
func (d *Backend) Addr() net.Addr {
	return d.config.Raft.StreamLayer.ln.Addr()
}

// CommittedIndex returns the latest index committed to stable storage
func (d *Backend) CommittedIndex() uint64 {
	return d.raft.LastIndex()
}

// AppliedIndex returns the latest index applied to the FSM
func (d *Backend) AppliedIndex() uint64 {
	return d.raft.AppliedIndex()
}

func (d *Backend) setupRaft(dataDir string) error {

	path := filepath.Join(dataDir, "raft")
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}

	// Create the backend raft store for logs and stable storage.
	store, err := raftboltdb.NewBoltStore(filepath.Join(path, "raft.db"))
	if err != nil {
		return err
	}
	d.stableStore = store

	// Wrap the store in a LogCache to improve performance.
	d.logStore, err = raft.NewLogCache(raftLogCacheSize, store)
	if err != nil {
		return err
	}

	retain := 1
	d.snapStore, err = raft.NewFileSnapshotStoreWithLogger(
		path,
		retain,
		d.logger.Named("snapshot"),
	)

	maxPool := 5
	timeout := 10 * time.Second
	transport := raft.NewNetworkTransport(
		d.config.Raft.StreamLayer,
		maxPool,
		timeout,
		d.logger.Named("net").StandardLogger(&log.StandardLoggerOptions{
			InferLevels: true,
		}).Writer(),
	)

	config := raft.DefaultConfig()
	config.LocalID = d.config.Raft.LocalID
	if d.config.Raft.HeartbeatTimeout != 0 {
		config.HeartbeatTimeout = d.config.Raft.HeartbeatTimeout
	}
	if d.config.Raft.ElectionTimeout != 0 {
		config.ElectionTimeout = d.config.Raft.ElectionTimeout
	}
	if d.config.Raft.LeaderLeaseTimeout != 0 {
		config.LeaderLeaseTimeout = d.config.Raft.LeaderLeaseTimeout
	}
	if d.config.Raft.CommitTimeout != 0 {
		config.CommitTimeout = d.config.Raft.CommitTimeout
	}
	config.Logger = d.logger

	d.l.Lock()
	defer d.l.Unlock()

	d.raft, err = raft.NewRaft(
		config,
		d.fsm,
		d.logStore,
		d.stableStore,
		d.snapStore,
		transport,
	)
	if err != nil {
		return err
	}

	if d.config.Raft.Bootstrap {
		config := raft.Configuration{
			Servers: []raft.Server{{
				ID:      config.LocalID,
				Address: transport.LocalAddr(),
			}},
		}
		err = d.raft.BootstrapCluster(config).Error()
	}

	return err
}

// WaitForLeader ...
func (d *Backend) WaitForLeader(timeout time.Duration) error {
	timeoutc := time.After(timeout)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-timeoutc:
			return fmt.Errorf("wait for leader timed out")
		case <-ticker.C:
			if l := d.raft.Leader(); l != "" {
				return nil
			}
		}
	}
}

// Close ...
func (d *Backend) Close() error {
	d.l.Lock()
	defer d.l.Unlock()

	if err := d.fsm.Close(); err != nil {
		return err
	}

	if err := d.stableStore.(*raftboltdb.BoltStore).Close(); err != nil {
		return err
	}

	return d.raft.Shutdown().Error()
}

func (d *Backend) applyLog(req proto.Marshaler) (interface{}, error) {

	b, err := req.Marshal()
	if err != nil {
		return nil, err
	}

	future := d.raft.Apply(b, 0)
	if future.Error() != nil {
		return nil, fmt.Errorf(
			"%s apply failed: %v", d.logger.Name(), future.Error())
	}

	res := future.Response()
	if err, ok := res.(error); ok {
		return nil, fmt.Errorf("%s apply failed: %v", d.logger.Name(), err)
	}

	return res, nil
}
