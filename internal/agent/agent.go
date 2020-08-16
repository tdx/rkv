package agent

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
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
	grpcServer *grpc.Server
	httpServer *http.Server
	membership *discovery.Membership

	shutdown     bool
	shutdownLock sync.Mutex
}

// New returns agent instance
func New(config *Config) (*Agent, error) {

	logger := config.Logger
	if logger == nil {
		logLevel := log.LevelFromString(config.Raft.LogLevel)
		if logLevel == log.NoLevel {
			logLevel = log.Error
		}
		logger = log.New(&log.LoggerOptions{
			Name:  fmt.Sprintf("agent-%s", config.NodeName),
			Level: logLevel,
		})
	}
	config.Logger = logger

	logger.Info("rkvd", "node-name", config.NodeName)
	logger.Info("rkvd", "data-dir", filepath.Dir(config.Backend.DSN()))
	logger.Info("rkvd", "discovery-join-address", config.StartJoinAddrs)

	a := &Agent{
		Config: config,
		logger: logger,
	}

	setup := []func() error{
		a.setupRaft,
		a.setupGrpcServer,
		a.setupHTTPServer,
		a.setupMembership,
	}
	for _, fn := range setup {
		if err := fn(); err != nil {
			return nil, err
		}
	}
	return a, nil
}

func (a *Agent) setupRaft() error {

	raftAddr, err := a.Config.RaftAddr()
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", raftAddr)
	if err != nil {
		return err
	}

	config := &rbk.Config{}
	config.Raft = a.Config.Raft
	config.Raft.Logger = a.logger.Named("raft")
	config.Raft.LocalID = config.ServerID(a.Config.NodeName)
	config.StreamLayer = rbk.NewStreamLayer(ln)
	config.Bootstrap = a.Config.Bootstrap

	a.raftDb, err = rbk.New(a.Config.Backend, config)
	if err != nil {
		return err
	}
	if a.Config.Bootstrap {
		return a.raftDb.(remoteApi.Leader).WaitForLeader(3 * time.Second)
	}
	return err
}

func (a *Agent) setupGrpcServer() error {
	config := &server.Config{
		Db: a.raftDb,
	}

	var err error
	a.grpcServer, err = server.NewGRPCServer(config)
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

	logger := a.logger.Named("grpc")
	go func() {
		if err := a.grpcServer.Serve(ln); err != nil {
			_ = a.Shutdown()
			logger.Error("server stopped", "error", err)
		}
	}()

	logger.Info("server started", "address", rpcAddr)

	return nil
}

func (a *Agent) setupHTTPServer() error {
	if a.Config.BindHTTP == "" {
		a.logger.Info("skip start HTTP server", "BindHTTP", "empty")
		return nil
	}

	config := &server.Config{
		Db:     a.raftDb,
		Logger: a.logger,
	}

	var err error
	a.httpServer, err = server.NewHTTPServer(config)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", a.Config.BindHTTP)
	if err != nil {
		return err
	}

	go func() {
		if err := a.httpServer.Serve(ln); err != nil {
			_ = a.Shutdown()
			config.Logger.Error("server stopped", "error", err)
		}
	}()

	config.Logger.Info("server started", "address", a.Config.BindHTTP)

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
			a.grpcServer.GracefulStop()
			return nil
		},
		func() error {
			ctx, cancel := context.WithTimeout(
				context.Background(), 5*time.Second)
			defer func() {
				cancel()
			}()
			return a.httpServer.Shutdown(ctx)
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
