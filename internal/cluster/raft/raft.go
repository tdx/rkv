package raft

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	dbApi "github.com/tdx/rkv/db/api"
	clusterApi "github.com/tdx/rkv/internal/cluster/api"
	rpcApi "github.com/tdx/rkv/internal/rpc/v1"
	"google.golang.org/grpc"

	"github.com/gogo/protobuf/proto"
	log "github.com/hashicorp/go-hclog"
	"github.com/tdx/raft"
	raftboltdb "github.com/tdx/raft-boltdb"
)

const (
	raftLogCacheSize = 1024
)

var _ clusterApi.Backend = (*Backend)(nil)

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

	observationCh  chan raft.Observation
	servers        map[string]*clusterApi.Server
	rpcLeaderAddr  string
	grpcLeaderConn *grpc.ClientConn
	leaderConn     rpcApi.StorageClient
}

// New ...
func New(
	db dbApi.Backend,
	config *Config) (*Backend, error) {

	logger := config.Raft.Logger

	if logger == nil {
		logLevel := log.LevelFromString(config.Raft.LogLevel)
		if logLevel == log.NoLevel {
			logLevel = log.Error
		}
		logger = log.New(&log.LoggerOptions{
			Name:  fmt.Sprintf("raft-%s", config.Raft.LocalID),
			Level: logLevel,
		})
	}
	config.Raft.Logger = logger

	if err := checkConfig(config); err != nil {
		return nil, err
	}

	d := &Backend{
		logger:  logger,
		config:  config,
		servers: make(map[string]*clusterApi.Server),
		fsm: &fsm{
			id:     string(config.Raft.LocalID),
			logger: logger.Named("fsm"),
			db:     db,
			appReg: config.ApplyRegistrator,
		},
	}

	//
	// add self to servers
	//
	host, ip, raftPort, rpcPort, err := d.parseAddrs(
		config.RaftAddr,
		config.RPCAddr,
	)
	if err != nil {
		return nil, err
	}

	id := string(config.Raft.LocalID)
	d.servers[id] = &clusterApi.Server{
		ID:       id,
		IP:       ip,
		Host:     host,
		RaftPort: raftPort,
		RPCPort:  rpcPort,
		IsLocal:  true,
	}

	dir := filepath.Dir(db.DSN())

	if err := d.setupRaft(dir); err != nil {
		return nil, err
	}

	return d, nil
}

// RaftAddr returns raft transport address
func (d *Backend) RaftAddr() net.Addr {
	return d.config.StreamLayer.ln.Addr()
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
	// store, err := raftldb.NewLevelDBStore(
	// 	filepath.Join(path, "stable"), raftldb.High)
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

	transport := raft.NewNetworkTransportWithConfig(
		&raft.NetworkTransportConfig{
			Logger:  d.logger.Named("net"),
			Stream:  d.config.StreamLayer,
			MaxPool: 5,
			Timeout: 10 * time.Second,
		})

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

	d.observationCh = make(chan raft.Observation, 1024)
	d.raft.RegisterObserver(raft.NewObserver(d.observationCh, false, nil))

	go d.waitEvents()

	if d.config.Bootstrap {
		config := raft.Configuration{
			Servers: []raft.Server{{
				ID:      config.LocalID,
				Address: raft.ServerAddress(d.config.RaftAddr),
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

// Close ...
func (d *Backend) Close() error {
	d.logger.Trace("stopping")
	if err := d.fsm.Close(); err != nil {
		return err
	}

	d.logger.Trace("closing stable store")
	if err := d.stableStore.(*raftboltdb.BoltStore).Close(); err != nil {
		return err
	}

	d.logger.Trace("shutdown")
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

//
func checkConfig(c *Config) error {
	if c.RPCAddr == "" {
		return errors.New("config.RPCAddr is empty")
	}
	if c.RaftAddr == "" {
		return errors.New("config.RaftAddr is empty")
	}
	return nil
}
