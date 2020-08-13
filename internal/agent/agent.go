package agent

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/tdx/rkv/internal/discovery"
	remoteApi "github.com/tdx/rkv/internal/remote/api"
	rbk "github.com/tdx/rkv/internal/remote/raft"
	"github.com/tdx/rkv/internal/server"

	log "github.com/hashicorp/go-hclog"
	"google.golang.org/grpc"
)

// Agent ...
type Agent struct {
	*Config

	logger     log.Logger
	raftDb     remoteApi.Backend
	server     *grpc.Server
	membership *discovery.Membership

	shutdown     bool
	shutdownLock sync.Mutex
}

// New returns agent instance
func New(config *Config) (*Agent, error) {

	logger := config.Logger
	if logger == nil {
		logger = log.New(&log.LoggerOptions{
			Name:  fmt.Sprintf("agent-%s", config.NodeName),
			Level: log.Error,
		})
		config.Logger = logger
	}

	a := &Agent{
		Config: config,
		logger: logger,
	}

	setup := []func() error{
		a.setupDistributed,
		a.setupServer,
		a.setupMembership,
	}
	for _, fn := range setup {
		if err := fn(); err != nil {
			return nil, err
		}
	}
	return a, nil
}

func (a *Agent) setupDistributed() error {

	raftAddr, err := a.Config.RaftAddr()
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", raftAddr)
	if err != nil {
		return err
	}

	config := rbk.Config{}
	config.Raft.LocalID = config.ServerID(a.Config.NodeName)
	config.Raft.StreamLayer = rbk.NewStreamLayer(ln)
	config.Raft.Bootstrap = a.Config.Bootstrap
	config.Raft.Config.Logger = a.logger.Named("raft")
	if a.Config.Raft.CommitTimeout > 0 {
		config.Raft.CommitTimeout = a.Config.Raft.CommitTimeout
	}
	if a.Config.Raft.ElectionTimeout > 0 {
		config.Raft.ElectionTimeout = a.Config.Raft.ElectionTimeout
	}
	if a.Config.Raft.HeartbeatTimeout > 0 {
		config.Raft.HeartbeatTimeout = a.Config.Raft.HeartbeatTimeout
	}
	if a.Config.Raft.LeaderLeaseTimeout > 0 {
		config.Raft.LeaderLeaseTimeout = a.Config.Raft.LeaderLeaseTimeout
	}
	if a.Config.Raft.MaxAppendEntries > 0 {
		config.Raft.MaxAppendEntries = a.Config.Raft.MaxAppendEntries
	}
	if a.Config.Raft.SnapshotInterval > 0 {
		config.Raft.SnapshotInterval = a.Config.Raft.SnapshotInterval
	}
	if a.Config.Raft.SnapshotThreshold > 0 {
		config.Raft.SnapshotThreshold = a.Config.Raft.SnapshotThreshold
	}

	a.raftDb, err = rbk.New(a.Config.Backend, config)
	if err != nil {
		return err
	}
	if a.Config.Bootstrap {
		return a.raftDb.(remoteApi.Leader).WaitForLeader(3 * time.Second)
	}
	return err
}

func (a *Agent) setupServer() error {
	config := &server.Config{
		Db: a.raftDb,
	}

	var err error
	a.server, err = server.NewGRPCServer(config)
	if err != nil {
		return err
	}

	rpcAddr, err := a.Config.RPCAddr()
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", rpcAddr)
	if err != nil {
		return err
	}

	go func() {
		if err := a.server.Serve(ln); err != nil {
			_ = a.Shutdown()
		}
	}()

	return nil
}

func (a *Agent) setupMembership() error {
	RaftAddr, err := a.Config.RaftAddr()
	if err != nil {
		return err
	}

	discoHandler := a.raftDb.(discovery.Handler)

	a.membership, err = discovery.New(
		discoHandler,
		discovery.Config{
			Logger:   a.logger.Named("serf"),
			NodeName: a.Config.NodeName,
			BindAddr: a.Config.BindAddr,
			Tags: map[string]string{
				"raft_addr": RaftAddr,
			},
			StartJoinAddrs: a.Config.StartJoinAddrs,
		},
	)

	return err
}

// Shutdown ...
func (a *Agent) Shutdown() error {
	a.shutdownLock.Lock()
	defer a.shutdownLock.Unlock()

	if a.shutdown {
		return nil
	}
	a.shutdown = true

	shutdown := []func() error{
		a.membership.Leave,
		func() error {
			a.server.GracefulStop()
			return nil
		},
		a.raftDb.Close,
	}

	for _, fn := range shutdown {
		if err := fn(); err != nil {
			return err
		}
	}

	return nil
}
