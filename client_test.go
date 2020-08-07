package rkv_test

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/tdx/rkv"
	"github.com/tdx/rkv/api"

	"github.com/stretchr/testify/require"
	"github.com/travisjeffery/go-dynaport"
)

func TestClient(t *testing.T) {

	dataDir, err := ioutil.TempDir("", "client-test-*")
	require.NoError(t, err)

	ports := dynaport.Get(2)

	config := &api.Config{
		DataDir:         dataDir,
		DiscoveryAddr:   fmt.Sprintf("127.0.0.1:%d", ports[0]),
		DistributedPort: ports[1],
		LogLevel:        "debug",
		NodeName:        "1",
	}

	client, err := rkv.NewClient(config)
	require.NoError(t, err)

	defer client.Shutdown()

	var (
		tab = []byte{'t', 'a', 'b'}
		key = []byte{'k', 'e', 'y'}
		val = []byte{'v', 'a', 'l'}
	)

	err = client.Put(tab, key, val)
	require.NoError(t, err)

	v, err := client.Get(tab, key)
	require.Equal(t, val, v)

	err = client.Delete(tab, key)
	require.NoError(t, err)

	_, err = client.Get(tab, key)
	require.Error(t, err)
}
