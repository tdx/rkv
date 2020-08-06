package agent

import (
	"fmt"
	"net"
	"sync"
	"time"

	"rkv/internal/discovery"
	remoteApi "rkv/internal/remote/api"
	rbk "rkv/internal/remote/raft"
	"rkv/internal/server"

	log "github.com/hashicorp/go-hclog"
	"google.golang.org/grpc"
)

// Agent ...
type Agent struct {
	Config

	logger     log.Logger
	db         remoteApi.Backend
	server     *grpc.Server
	membership *discovery.Membership

	shutdown     bool
	shutdownLock sync.Mutex
}

// New returns agent instance
func New(config Config) (*Agent, error) {

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
		a.setupBackend,
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

func (a *Agent) setupBackend() error {

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

	a.db, err = rbk.NewBackend(a.Config.DataDir, config)
	if a.Config.Bootstrap {
		return a.db.(remoteApi.Leader).WaitForLeader(3 * time.Second)
	}
	return err
}

func (a *Agent) setupServer() error {
	config := &server.Config{
		Db: a.db,
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

	discoHandler := a.db.(discovery.Handler)

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
		a.db.Close,
	}

	for _, fn := range shutdown {
		if err := fn(); err != nil {
			return err
		}
	}

	return nil
}
