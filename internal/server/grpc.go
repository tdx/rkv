package server

import (
	"context"

	rkvApi "github.com/tdx/rkv/api"
	clusterApi "github.com/tdx/rkv/internal/cluster/api"
	rpcApi "github.com/tdx/rkv/internal/rpc/v1"

	log "github.com/hashicorp/go-hclog"
	"google.golang.org/grpc"
)

var _ rpcApi.StorageServer = (*grpcServer)(nil)

// Config ...
type Config struct {
	Db     clusterApi.Backend
	Logger log.Logger
	Addr   string
}

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
