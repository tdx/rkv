package agent

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	rkvApi "github.com/tdx/rkv/api"
	clusterApi "github.com/tdx/rkv/internal/cluster/api"
	rbk "github.com/tdx/rkv/internal/cluster/raft"
	"github.com/tdx/rkv/internal/discovery"
	"github.com/tdx/rkv/internal/route"
	"github.com/tdx/rkv/internal/server"
	"github.com/tdx/rkv/registry"

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
func New(cfg *Config) (*Agent, error) {

	logger := cfg.Logger
	if logger == nil {
		logLevel := hlog.LevelFromString(cfg.LogLevel)
		if logLevel == hlog.NoLevel {
			logLevel = hlog.Info
		}
		logFormat := cfg.LogTimeFormat
		if logFormat == "" {
			logFormat = rkvApi.DefaultTimeFormat
		}
		logOpts := &hlog.LoggerOptions{
			Name:            fmt.Sprintf("rkv-%s", cfg.NodeName),
			Level:           logLevel,
			IncludeLocation: cfg.LogIncludeLocation,
			Output:          cfg.LogOutput,
			TimeFormat:      logFormat,
		}
		if logLevel > hlog.Debug {
			// to skip serf and memberlist debug logs
			logOpts.Exclude = func(
				level hlog.Level, msg string, args ...interface{}) bool {

				return strings.Index(msg, "[DEBUG]") > -1
			}
		}
		logger = hlog.New(logOpts)
	}
	cfg.Logger = logger

	hostname, _ := os.Hostname()
	rpcAddr, _ := cfg.RPCAddr()
	logger.Info("os", "hostname", hostname)
	logger.Info("config", "log-level", cfg.LogLevel)
	logger.Info("config", "node-name", cfg.NodeName)
	logger.Info("config", "data-dir", cfg.DataDir)
	logger.Info("config", "db", cfg.Backend.DSN())
	logger.Info("config", "discovery-join-address", cfg.StartJoinAddrs)
	logger.Info("config", "gRPC address", rpcAddr)
	logger.Info("config", "Raft.Heartbeat timeout", cfg.Raft.HeartbeatTimeout)
	logger.Info("config", "Raft.Election timeout", cfg.Raft.ElectionTimeout)

	a := &Agent{
		Config:   cfg,
		logger:   logger,
		registry: cfg.Registry,
	}

	if a.registry == nil {
		a.registry = registry.NewApplyRegistrator()
	}

	var setup = []struct {
		name string
		fn   func() error
	}{
		{"setupRoute", a.setupRoute},
		{"setupRaft", a.setupRaft},
		{"setupMembership", a.setupMembership},
		{"setupGrpcServer", a.setupGrpcServer},
		{"setupHTTPServer", a.setupHTTPServer},
	}
	for _, s := range setup {
		if err := s.fn(); err != nil {
			return nil, fmt.Errorf("%v: %v", s.name, err)
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
	config.Raft.LogLevel = a.Config.LogLevel
	config.Raft.Logger = a.logger.Named("raft")
	config.Raft.LocalID = config.ServerID(a.Config.NodeName)
	config.RPCAddr = rpcAddr
	config.RaftAddr = raftAddr
	config.StreamLayer = rbk.NewStreamLayer(ln)
	config.Bootstrap = a.Config.Bootstrap
	config.ApplyRegistrator = a.registry
	config.OnLeaderChangeFn = a.Config.OnLeaderChangeFn

	a.raftDb, err = rbk.New(a.Config.Backend, config)
	if err != nil {
		return err
	}
	if a.Config.Bootstrap && !a.raftDb.(clusterApi.Cluster).Restarted() {
		return a.raftDb.(clusterApi.Cluster).WaitForLeader(3 * time.Second)
	}
	return nil
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
			DataDir:  a.Config.DataDir,
			NodeName: a.Config.NodeName,
			BindAddr: a.Config.BindAddr,
			Tags: map[string]string{
				"serf_addr": a.Config.BindAddr,
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
		Db:     a.raftDb,
		Joiner: a.membership,
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
		Joiner: a.membership,
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
