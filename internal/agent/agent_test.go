package agent_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"rkv/internal/agent"

	log "github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
	"github.com/travisjeffery/go-dynaport"
)

func TestAgent(t *testing.T) {
	var agents []*agent.Agent
	for i := 0; i < 3; i++ {
		ports := dynaport.Get(3)
		bindAddr := fmt.Sprintf("127.0.0.1:%d", ports[0]) // serf

		dataDir, err := ioutil.TempDir("", "agent-test")
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

		agent, err := agent.New(agent.Config{
			Logger:         logger,
			NodeName:       nodeName,
			BindAddr:       bindAddr,
			RPCPort:        ports[1],
			RaftPort:       ports[2],
			DataDir:        dataDir,
			StartJoinAddrs: startJoinAddrs,
			Bootstrap:      bootstrap,
		})
		require.NoError(t, err)

		agents = append(agents, agent)
	}
	defer func() {
		for _, agent := range agents {
			err := agent.Shutdown()
			require.NoError(t, err)
			require.NoError(t,
				os.RemoveAll(agent.Config.DataDir),
			)
		}
	}()
	time.Sleep(3 * time.Second)

	var (
		tab = []byte{'t', 'a', 'b'}
		key = []byte{'k', 'e', 'y'}
		val = []byte{'v', 'a', 'l'}
	)

	leaderClient := agents[0]
	err := leaderClient.Put(tab, key, val)
	require.NoError(t, err)

	v, err := leaderClient.Get(tab, key)
	require.NoError(t, err)
	require.Equal(t, v, val)

	// wait until replication has finished
	time.Sleep(3 * time.Second)

	followerClient := agents[1]
	followerVal, err := followerClient.Get(tab, key)
	require.NoError(t, err)
	require.Equal(t, val, followerVal)
}
