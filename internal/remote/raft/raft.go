package raft

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	dbApi "github.com/tdx/rkv/db/api"
	remoteApi "github.com/tdx/rkv/internal/remote/api"

	"github.com/gogo/protobuf/proto"
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	raftldb "github.com/tidwall/raft-leveldb"
)

const (
	raftLogCacheSize = 512
)

var _ remoteApi.Backend = (*Backend)(nil)

// Backend ...
type Backend struct {
	logger log.Logger
	config *Config
	fsm    *fsm

	raft        *raft.Raft
	snapStore   raft.SnapshotStore
	logStore    raft.LogStore
	stableStore raft.StableStore
	dataDir     string
	restarted   bool
}

// New ...
func New(
	db dbApi.Backend,
	config *Config) (*Backend, error) {

	logger := config.Raft.Config.Logger

	if logger == nil {
		logger = log.New(&log.LoggerOptions{
			Name:  fmt.Sprintf("raft-%s", config.Raft.LocalID),
			Level: log.Error,
		})
		config.Raft.Config.Logger = logger
	} else {
		logger = logger.Named(fmt.Sprintf("raft-%s", config.Raft.LocalID))
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
	// store, err := raftboltdb.NewBoltStore(filepath.Join(path, "raft.db"))
	store, err := raftldb.NewLevelDBStore(
		filepath.Join(path, "stable"), raftldb.High)
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
	if d.config.Raft.CommitTimeout > 0 {
		config.CommitTimeout = d.config.Raft.CommitTimeout
	}
	if d.config.Raft.ElectionTimeout > 0 {
		config.ElectionTimeout = d.config.Raft.ElectionTimeout
	}
	if d.config.Raft.HeartbeatTimeout > 0 {
		config.HeartbeatTimeout = d.config.Raft.HeartbeatTimeout
	}
	if d.config.Raft.LeaderLeaseTimeout > 0 {
		config.LeaderLeaseTimeout = d.config.Raft.LeaderLeaseTimeout
	}
	if d.config.Raft.MaxAppendEntries > 0 {
		config.MaxAppendEntries = d.config.Raft.MaxAppendEntries
	}
	if d.config.Raft.SnapshotInterval > 0 {
		config.SnapshotInterval = d.config.Raft.SnapshotInterval
	}
	if d.config.Raft.SnapshotThreshold > 0 {
		config.SnapshotThreshold = d.config.Raft.SnapshotThreshold
	}
	config.Logger = d.logger

	d.restarted, err = raft.HasExistingState(
		d.logStore, d.stableStore, d.snapStore)

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
		// error can be safety ignored
		d.raft.BootstrapCluster(config).Error()
	}

	return err
}

// Restarted check node has state
func (d *Backend) Restarted() bool {
	return d.restarted
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
			l := d.raft.Leader()
			d.logger.Info("wait for leader", "leader", l)
			if l != "" {
				return nil
			}
		}
	}
}

// Close ...
func (d *Backend) Close() error {
	if err := d.fsm.Close(); err != nil {
		return err
	}

	if err := d.stableStore.(*raftldb.LevelDBStore).Close(); err != nil {
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
	if err := future.Error(); err != nil {
		d.logger.Error("applyLog", "call error", err)
		return nil, err
	}

	res := future.Response()
	if err, ok := res.(error); ok {
		d.logger.Error("applyLog", "result error", err)
		return nil, err
	}

	return res, nil
}
