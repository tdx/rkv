package server

import (
	"context"
	"net"

	rkvApi "github.com/tdx/rkv/api"
	clusterApi "github.com/tdx/rkv/internal/cluster/api"
	rpcApi "github.com/tdx/rkv/internal/rpc/v1"

	log "github.com/hashicorp/go-hclog"
	"google.golang.org/grpc"
)

var _ rpcApi.StorageServer = (*grpcServer)(nil)

type grpcServer struct {
	*Config
}

// NewGRPCServer ...
func NewGRPCServer(config *Config) (*grpc.Server, error) {
	gsrv := grpc.NewServer()
	srv, err := newGrpcServer(config)
	if err != nil {
		return nil, err
	}
	rpcApi.RegisterStorageServer(gsrv, srv)
	return gsrv, nil
}

func newGrpcServer(config *Config) (*grpcServer, error) {
	// check logger
	logger := config.Logger
	if logger == nil {
		logger = log.New(&log.LoggerOptions{
			Name:  "http",
			Level: log.Error,
		})
	} else {
		logger = logger.Named("http")
	}
	config.Logger = logger

	return &grpcServer{Config: config}, nil
}

func (s *grpcServer) Put(
	ctx context.Context,
	req *rpcApi.StoragePutArgs) (*rpcApi.StoragePutReply, error) {

	err := s.Db.Put(req.Tab, req.Key, req.Val)
	if err != nil {
		return &rpcApi.StoragePutReply{Err: err.Error()}, nil
	}

	return &rpcApi.StoragePutReply{Err: ""}, nil
}

func (s *grpcServer) Get(
	ctx context.Context,
	req *rpcApi.StorageGetArgs) (*rpcApi.StorageGetReply, error) {

	val, err := s.Db.Get(rkvApi.ConsistencyLevel(req.Lvl), req.Tab, req.Key)
	if err != nil {
		return &rpcApi.StorageGetReply{Val: nil, Err: err.Error()}, err
	}

	return &rpcApi.StorageGetReply{Val: val, Err: ""}, nil
}

func (s *grpcServer) Delete(
	ctx context.Context,
	req *rpcApi.StorageDeleteArgs) (*rpcApi.StorageDeleteReply, error) {

	err := s.Db.Delete(req.Tab, req.Key)
	if err != nil {
		return &rpcApi.StorageDeleteReply{Err: err.Error()}, err
	}

	return &rpcApi.StorageDeleteReply{Err: ""}, nil
}

//
func (s *grpcServer) Servers(
	ctx context.Context,
	req *rpcApi.ServersArgs) (*rpcApi.ServersReply, error) {

	cluster := s.Db.(clusterApi.Cluster)
	servers, err := cluster.Servers()
	if err != nil {
		return nil, err
	}

	grpcServers := make([]*rpcApi.Server, 0, len(servers))

	for i := range servers {
		server := servers[i]
		grpcServers = append(grpcServers, &rpcApi.Server{
			Id:       server.ID,
			RpcAddr:  server.RPCAddr,
			RaftAddr: server.RaftAddr,
			IsLeader: server.IsLeader,
			Online:   server.Online,
		})
	}

	s.Logger.Debug("Servers", "servers", grpcServers)

	return &rpcApi.ServersReply{Servers: grpcServers}, nil
}

func addrsEquals(addr1, addr2 string) (bool, error) {
	h1, p1, err := net.SplitHostPort(addr1)
	if err != nil {
		return false, err
	}

	h2, p2, err := net.SplitHostPort(addr2)
	if err != nil {
		return false, err
	}

	if p1 != p2 {
		return false, nil
	}

	if (h1 == "" && (h2 == "" || h2 == "::")) ||
		(h2 == "" && (h1 == "" || h1 == "::")) {

		return true, nil
	}

	return h1 == h2, nil
}
