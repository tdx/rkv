package raft

import (
	"sync"

	rpcApi "github.com/tdx/rkv/internal/rpc/v1"

	log "github.com/hashicorp/go-hclog"
	"google.golang.org/grpc"
)

type leader struct {
	logger         log.Logger
	leaderLock     sync.RWMutex
	rpcLeaderAddr  string
	grpcLeaderConn *grpc.ClientConn
	leaderConn     rpcApi.StorageClient
}

func newLeader(logger log.Logger) *leader {
	return &leader{
		logger: logger,
	}
}

func (l *leader) RPCLeaderAddr() string {
	return l.rpcLeaderAddr
}

func (l *leader) setLeader(rpcAddr string, conn *grpc.ClientConn) {

	l.closeLeader()

	l.leaderLock.Lock()
	defer l.leaderLock.Unlock()

	l.rpcLeaderAddr = rpcAddr
	l.grpcLeaderConn = conn
	l.leaderConn = rpcApi.NewStorageClient(conn)
}

func (l *leader) getLeader() rpcApi.StorageClient {
	l.leaderLock.RLock()
	defer l.leaderLock.RUnlock()

	return l.leaderConn
}

func (l *leader) closeLeader() {
	l.leaderLock.Lock()
	defer l.leaderLock.Unlock()

	if l.grpcLeaderConn != nil {
		l.logger.Debug("closeLeader", "rpc-leader-addr", l.rpcLeaderAddr)
		l.grpcLeaderConn.Close()
		l.grpcLeaderConn = nil
		l.leaderConn = nil
		l.rpcLeaderAddr = ""
	}
}
