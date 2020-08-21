package agent_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	rkvApi "github.com/tdx/rkv/api"
	dbApi "github.com/tdx/rkv/db/api"
	"github.com/tdx/rkv/db/bitcask"
	"github.com/tdx/rkv/db/bolt"
	"github.com/tdx/rkv/db/gmap"
	"github.com/tdx/rkv/internal/agent"

	log "github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
	"github.com/travisjeffery/go-dynaport"
)

func TestAgentBolt(t *testing.T) {
	runAgent(t, "bolt")
}

func TestAgentMap(t *testing.T) {
	runAgent(t, "map")
}

func TestAgentBitcask(t *testing.T) {
	runAgent(t, "bitcask")
}

func runAgent(t *testing.T, bkType string) {
	var agents []*agent.Agent
	for i := 0; i < 3; i++ {
		ports := dynaport.Get(3)
		bindAddr := fmt.Sprintf("127.0.0.1:%d", ports[0]) // serf

		dataDir, err := ioutil.TempDir("", "agent-test")
		require.NoError(t, err)

		var db dbApi.Backend
		switch bkType {
		case "map":
			db, err = gmap.New(dataDir)
		case "bitcask":
			db, err = bitcask.New(dataDir, 1<<20) // 1 MB
		default:
			db, err = bolt.New(dataDir)
		}
		require.NoError(t, err)

		var startJoinAddrs []string
		if i != 0 {
			startJoinAddrs = append(
				startJoinAddrs,
				agents[0].Config.BindAddr,
			)
		}

		bootstrap := i == 0
		nodeName := fmt.Sprintf("%d", i)

		logger := log.New(&log.LoggerOptions{
			Name:  fmt.Sprintf("agent-%s", nodeName),
			Level: log.Debug,
		})

		agent, err := agent.New(&agent.Config{
			Backend:        db,
			Logger:         logger,
			NodeName:       nodeName,
			BindAddr:       bindAddr,
			RPCPort:        ports[1],
			RaftPort:       ports[2],
			StartJoinAddrs: startJoinAddrs,
			Bootstrap:      bootstrap,
		})
		require.NoError(t, err)

		agents = append(agents, agent)
	}
	defer func() {
		for _, agent := range agents {
			dir := filepath.Dir(agent.Config.Backend.DSN())

			err := agent.Shutdown()
			require.NoError(t, err)
			require.NoError(t, os.RemoveAll(dir))
		}
	}()
	time.Sleep(3 * time.Second)

	var (
		tab  = []byte{'t', 'a', 'b'}
		key  = []byte{'k', 'e', 'y'}
		key2 = []byte{'k', 'e', 'y', '2'}
		val  = []byte{'v', 'a', 'l'}
		val2 = []byte{'v', 'a', 'l', '2'}
	)

	leaderClient := agents[0]
	err := leaderClient.Put(tab, key, val)
	require.NoError(t, err)

	v, err := leaderClient.Get(rkvApi.ReadCluster, tab, key)
	require.NoError(t, err)
	require.Equal(t, v, val)

	// wait until replication has finished
	time.Sleep(3 * time.Second)

	followerClient := agents[1]
	followerVal, err := followerClient.Get(rkvApi.ReadAny, tab, key)
	require.NoError(t, err)
	require.Equal(t, val, followerVal)

	//
	// Routing to leader
	//
	// rpcAddrFollower, err := agents[1].Config.RPCAddr()
	// require.NoError(t, err)

	// clientOpts := []grpc.DialOption{grpc.WithInsecure()}
	// conn, err := grpc.Dial(fmt.Sprintf(
	// 	"%s:///%s", agents[1].Config.NodeName, rpcAddrFollower),
	// 	clientOpts...,
	// )
	// require.NoError(t, err)

	// grpcFollowerClient := rpcApi.NewStorageClient(conn)

	// ctx := context.Background()
	// resp, err := grpcFollowerClient.Put(ctx, &rpcApi.StoragePutArgs{
	// 	Tab: tab,
	// 	Key: key,
	// 	Val: val,
	// })
	// require.NoError(t, err)
	// require.Empty(t, resp.GetErr())

	// try write to follower
	err = followerClient.Put(tab, key2, val2)
	require.NoError(t, err)

	v2, err := followerClient.Get(rkvApi.ReadLeader, tab, key2)
	require.NoError(t, err)
	require.Equal(t, val2, v2)

	// TODO: test routing for all Client API
}
