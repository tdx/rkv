package rkv_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/tdx/rkv"
	"github.com/tdx/rkv/api"
	rkvApi "github.com/tdx/rkv/api"
	dbApi "github.com/tdx/rkv/db/api"
	"github.com/tdx/rkv/db/bitcask"
	rkvBolt "github.com/tdx/rkv/db/bolt"
	"github.com/tdx/rkv/db/gmap"

	"github.com/boltdb/bolt"
	"github.com/stretchr/testify/require"
	"github.com/travisjeffery/go-dynaport"
)

func TestClientBolt(t *testing.T) {
	run(t, "bolt")
}

func TestClientMap(t *testing.T) {
	run(t, "map")
}

// func TestClientBitcask(t *testing.T) {
// 	run(t, "bitcask")
// }

func run(t *testing.T, bkType string) {
	dataDir, err := ioutil.TempDir("", "client-test-*")
	require.NoError(t, err)

	defer os.RemoveAll(dataDir)

	var db dbApi.Backend
	switch bkType {
	case "map":
		db, err = gmap.New(dataDir)
	case "bitcask":
		db, err = bitcask.New(dataDir, 1<<20) // 1 MB
	default:
		db, err = rkvBolt.New(dataDir)
	}
	require.NoError(t, err)

	ports := dynaport.Get(2)

	config := &rkvApi.Config{
		Backend:       db,
		NodeName:      "1",
		LogLevel:      "debug",
		DiscoveryAddr: fmt.Sprintf("127.0.0.1:%d", ports[0]),
		RaftPort:      ports[1],
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

	v, err := client.Get(api.ReadAny, tab, key)
	require.NoError(t, err)
	require.Equal(t, val, v)

	err = client.Delete(tab, key)
	require.NoError(t, err)

	_, err = client.Get(api.ReadAny, tab, key)
	require.Equal(t, true, dbApi.IsNoKeyError(err))

	v, err = client.Get(api.ReadCluster, tab, key)
	t.Log("v:", v, "err:", err)
	require.Equal(t, true, dbApi.IsNoKeyError(err))
	require.Nil(t, v)

	if bkType != "bolt" {
		return
	}

	fnCount := func(dbCtx interface{}, args ...[]byte) (interface{}, error) {
		tx, ok := dbCtx.(*bolt.Tx)
		if !ok {
			return nil, fmt.Errorf("invalid dbCtx: %T %T", dbCtx, tx)
		}

		if len(args) != 1 {
			return nil, fmt.Errorf("invalid arguments, expectec 1")
		}

		var (
			tab       = args[0]
			count int = 0
		)

		b := tx.Bucket(tab)
		if b == nil {
			return 0, dbApi.ErrNoTable(tab)
		}

		st := b.Stats()
		count = st.KeyN

		return count, nil
	}

	err = client.RegisterApplyRead("count", fnCount)
	require.NoError(t, err)

	n, err := client.ApplyRead(rkvApi.ReadCluster, "count", tab)
	require.NoError(t, err)

	n, ok := n.(int)
	require.True(t, ok)
	require.Equal(t, 0, n)

	// apply test
	fn := func(dbCtx interface{}, args ...[]byte) (interface{}, error) {

		tx, ok := dbCtx.(*bolt.Tx)
		if !ok {
			return nil, fmt.Errorf("invalid dbCtx: %T %T", dbCtx, tx)
		}

		if len(args) != 3 {
			return nil, fmt.Errorf(
				"wrong count of arguments, expected 3, got %d", len(args))
		}

		tab := args[0]
		key := args[1]
		val := args[2]

		b, err := tx.CreateBucketIfNotExists(tab)
		if err != nil {
			return nil, err
		}

		err = b.Put(key, val)

		return val, err
	}

	err = client.RegisterApplyWrite("insert", fn)
	require.NoError(t, err)

	var (
		tab2 = []byte("test")
		key2 = []byte("key1")
		val2 = []byte("val1")
	)
	args := make(map[string][]byte)
	args["tab"] = tab2
	args["key"] = key2
	args["val"] = val2

	// bytes, err := json.Marshal(args)
	// require.NoError(t, err)

	r, err := client.ApplyWrite("insert", tab2, key2, val2)
	require.NoError(t, err)

	rb, ok := r.([]byte)
	require.True(t, ok)
	require.Equal(t, val2, rb)

	// read back
	v2, err := client.Get(rkvApi.ReadAny, tab2, key2)
	require.NoError(t, err)
	require.Equal(t, val2, v2)
}
