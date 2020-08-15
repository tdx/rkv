package server_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	dbApi "github.com/tdx/rkv/db/api"
	"github.com/tdx/rkv/db/bolt"
	"github.com/tdx/rkv/db/gmap"
	remoteApi "github.com/tdx/rkv/internal/remote/api"
	rRaft "github.com/tdx/rkv/internal/remote/raft"
	rpcApi "github.com/tdx/rkv/internal/rpc/v1"
	"github.com/tdx/rkv/internal/server"
	"github.com/travisjeffery/go-dynaport"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

//
// It writes direct to dbApi.Backend instead of remoteApi.Backend
//
func TestGrpsServer(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
		client rpcApi.StorageClient,
		config *server.Config,
	){
		"put/get key-value pair over GRPC succeeds": testGrpcPutGet,
	} {
		t.Run(scenario, func(t *testing.T) {
			client, config, teardown := setupGrpcTest(t)
			defer teardown()
			fn(t, client, config)
		})
	}
}

func setupGrpcTest(
	t *testing.T) (rpcApi.StorageClient, *server.Config, func()) {

	t.Helper()

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	clientOptions := []grpc.DialOption{grpc.WithInsecure()}
	cc, err := grpc.Dial(l.Addr().String(), clientOptions...)
	require.NoError(t, err)

	raft, dir := getRaft(t, "1", true, "bolt")

	config := &server.Config{
		Db: raft,
	}

	server, err := server.NewGRPCServer(config)
	require.NoError(t, err)

	go func() {
		server.Serve(l)
	}()

	client := rpcApi.NewStorageClient(cc)

	return client, config, func() {
		server.Stop()
		cc.Close()
		l.Close()
		raft.Close()
		func() {
			os.RemoveAll(dir)
		}()
	}
}

func testGrpcPutGet(
	t *testing.T,
	client rpcApi.StorageClient,
	config *server.Config) {

	ctx := context.Background()

	var (
		tab = []byte{'t', 'a', 'b'}
		key = []byte{'k', 'e', 'y'}
		val = []byte{'v', 'a', 'l'}
	)

	putReply, err := client.Put(ctx, &rpcApi.StoragePutArgs{
		Tab: tab,
		Key: key,
		Val: val,
	})
	require.NoError(t, err)
	require.Equal(t, "", putReply.Err)

	getReply, err := client.Get(ctx, &rpcApi.StorageGetArgs{
		Tab: tab,
		Key: key,
	})
	require.NoError(t, err)
	require.Equal(t, "", getReply.Err)
	require.Equal(t, val, getReply.Val)
}

//
//
//
func getRaft(
	t testing.TB,
	id string,
	bootstrap bool,
	bkTyp string) (remoteApi.Backend, string) {

	raftDir, err := ioutil.TempDir("", "rkv-raft-")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("raft dir:", raftDir)

	return getRaftWithDir(t, id, bootstrap, raftDir, bkTyp)
}

func getRaftWithDir(
	t testing.TB,
	id string,
	bootstrap bool,
	raftDir string,
	bkTyp string) (remoteApi.Backend, string) {

	ports := dynaport.Get(1)

	ln, err := net.Listen(
		"tcp",
		fmt.Sprintf("127.0.0.1:%d", ports[0]),
	)
	require.NoError(t, err)

	config := &rRaft.Config{}
	config.Bootstrap = bootstrap
	config.StreamLayer = rRaft.NewStreamLayer(ln)
	config.Raft.LocalID = raft.ServerID(id)
	config.Raft.HeartbeatTimeout = 50 * time.Millisecond
	config.Raft.ElectionTimeout = 50 * time.Millisecond
	config.Raft.LeaderLeaseTimeout = 50 * time.Millisecond
	config.Raft.CommitTimeout = 5 * time.Millisecond

	var db dbApi.Backend
	switch bkTyp {
	case "map":
		db, err = gmap.New(raftDir)
	default:
		db, err = bolt.New(raftDir)
	}
	require.NoError(t, err)

	r, err := rRaft.New(db, config)
	require.NoError(t, err)

	if bootstrap {
		r.WaitForLeader(3 * time.Second)
	}

	return r, raftDir
}
