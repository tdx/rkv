package server_test

import (
	"context"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/tdx/rkv/db/bolt"
	remoteApi "github.com/tdx/rkv/internal/remote/api"
	rpcApi "github.com/tdx/rkv/internal/rpc/v1"
	"github.com/tdx/rkv/internal/server"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

//
// It writes direct to dbApi.Backend instead of remoteApi.Backend
//
func TestServer(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
		client rpcApi.StorageClient,
		config *server.Config,
	){
		"put/get key-value pair succeeds": testPutGet,
	} {
		t.Run(scenario, func(t *testing.T) {
			client, config, teardown := setupTest(t, nil)
			defer teardown()
			fn(t, client, config)
		})
	}
}

func setupTest(
	t *testing.T,
	fn func(*server.Config)) (rpcApi.StorageClient, *server.Config, func()) {

	t.Helper()

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	clientOptions := []grpc.DialOption{grpc.WithInsecure()}
	cc, err := grpc.Dial(l.Addr().String(), clientOptions...)
	require.NoError(t, err)

	dir, err := ioutil.TempDir("", "server-test")
	require.NoError(t, err)

	db, err := bolt.New(filepath.Join(dir, "bolt.db"))
	require.NoError(t, err)

	config := &server.Config{
		Db: db.(remoteApi.Backend),
	}

	if fn != nil {
		fn(config)
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
		db.Close()
		func() {
			os.RemoveAll(dir)
		}()
	}
}

func testPutGet(
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
