package agent

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	rkvApi "github.com/tdx/rkv/api"
	clusterApi "github.com/tdx/rkv/internal/cluster/api"
	rbk "github.com/tdx/rkv/internal/cluster/raft"
	"github.com/tdx/rkv/internal/discovery"
	"github.com/tdx/rkv/internal/registry"
	"github.com/tdx/rkv/internal/route"
	"github.com/tdx/rkv/internal/server"

	hlog "github.com/hashicorp/go-hclog"
	"google.golang.org/grpc"
)

// Agent ...
type Agent struct {
	*Config

	logger     hlog.Logger
	raftDb     clusterApi.Backend
	grpcServer *grpc.Server
	httpServer *http.Server
	membership *discovery.Membership
	route      *route.Route

	shutdown     bool
	shutdownLock sync.Mutex

	registry rkvApi.ApplyRegistrator
}

// New returns agent instance
func New(config *Config) (*Agent, error) {

	logger := config.Logger
	if logger == nil {
		logLevel := hlog.LevelFromString(config.Raft.LogLevel)
		if logLevel == hlog.NoLevel {
			logLevel = hlog.Error
		}
		logger = hlog.New(&hlog.LoggerOptions{
			Name:  fmt.Sprintf("agent-%s", config.NodeName),
			Level: logLevel,
		})
	}
	config.Logger = logger

	hostname, _ := os.Hostname()
	rpcAddr, _ := config.RPCAddr()
	logger.Info("os", "hostname", hostname)
	logger.Info("config", "log-level", config.Raft.LogLevel)
	logger.Info("config", "node-name", config.NodeName)
	logger.Info("config", "data-dir", filepath.Dir(config.Backend.DSN()))
	logger.Info("config", "db", config.Backend.DSN())
	logger.Info("config", "discovery-join-address", config.StartJoinAddrs)
	logger.Info("config", "gRPC address", rpcAddr)
	logger.Info("config", "Raft.Heartbeat timeout", config.Raft.HeartbeatTimeout)
	logger.Info("config", "Raft.Election timeout", config.Raft.ElectionTimeout)

	a := &Agent{
		Config:   config,
		logger:   logger,
		registry: registry.NewApplyRegistrator(),
	}

	setup := []func() error{
		a.setupRoute,
		a.setupRaft,
		a.setupMembership,
		a.setupGrpcServer,
		a.setupHTTPServer,
	}
	for _, fn := range setup {
		if err := fn(); err != nil {
			return nil, err
		}
	}
	return a, nil
}

func (a *Agent) setupRoute() error {
	name := strings.Split(a.Config.NodeName, ".")
	config := &route.Config{
		Name:   name[0],
		Logger: a.logger.Named("route"),
	}
	a.route = route.New(config)

	return nil
}

func (a *Agent) setupRaft() error {

	raftAddr, err := a.Config.RaftAddr()
	if err != nil {
		return err
	}
	rpcAddr, err := a.Config.RPCAddr()
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
	config.RPCAddr = rpcAddr
	config.RaftAddr = raftAddr
	config.StreamLayer = rbk.NewStreamLayer(ln)
	config.Bootstrap = a.Config.Bootstrap
	config.ApplyRegistrator = a.registry

	a.raftDb, err = rbk.New(a.Config.Backend, config)
	if err != nil {
		return err
	}
	if a.Config.Bootstrap {
		return a.raftDb.(clusterApi.Cluster).WaitForLeader(3 * time.Second)
	}
	return err
}

func (a *Agent) setupMembership() error {
	RaftAddr, err := a.Config.RaftAddr()
	if err != nil {
		return err
	}
	RPCAddr, err := a.Config.RPCAddr()
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
				"rpc_addr":  RPCAddr,
				"raft_addr": RaftAddr,
				"http_addr": a.Config.BindHTTP,
			},
			StartJoinAddrs: a.Config.StartJoinAddrs,
		},
	)

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
		Logger: a.logger.Named("http"),
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
	}

	for _, fn := range shutdown {
		if err := fn(); err != nil {
			return err
		}
	}

	return nil
}
