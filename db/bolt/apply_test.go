package bolt_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/boltdb/bolt"
	dbApi "github.com/tdx/rkv/db/api"
	rkvBolt "github.com/tdx/rkv/db/bolt"

	"github.com/stretchr/testify/require"
)

func TestApply(t *testing.T) {
	defer os.RemoveAll("/tmp/rkv2")

	db, err := rkvBolt.New("/tmp/rkv2")
	require.NoError(t, err)

	dbFn, ok := db.(dbApi.Applier)
	require.True(t, ok)

	var (
		tab = []byte{'t', 'a', 'b'}
		key = []byte{'k', 'e', 'y'}
		val = []byte{'v', 'a', 'l'}
	)

	// no table yet
	count, err := dbFn.Apply(fnCount, tab, true)
	require.Equal(t, true, dbApi.IsNoTableError(err))

	err = db.Put(tab, key, val)
	require.NoError(t, err)

	// count keys
	count, err = dbFn.Apply(fnCount, tab, true)
	require.NoError(t, err)

	n, ok := count.(int)
	require.True(t, ok)
	require.Equal(t, 1, n)

	// update
	_, err = dbFn.Apply(fnUp, tab, false)
	require.NoError(t, err)

	// count keys
	count, err = dbFn.Apply(fnCount, tab, true)
	require.NoError(t, err)

	n, ok = count.(int)
	require.True(t, ok)
	require.Equal(t, 2, n)
}


func fnCount(dbCtx interface{}, args []byte) (interface{}, error) {
	tx, ok := dbCtx.(*bolt.Tx)
	if !ok {
		return nil, fmt.Errorf("invalid dbCtx: %T %T", dbCtx, tx)
	}

	var (
		tab = args
	)

	b := tx.Bucket(tab)
	if b == nil {
		return 0, dbApi.ErrNoTable(tab)
	}

	st := b.Stats()

	return st.KeyN, nil
}

func fnUp(dbCtx interface{}, args []byte) (interface{}, error) {
	tx, ok := dbCtx.(*bolt.Tx)
	if !ok {
		return nil, fmt.Errorf("invalid dbCtx: %T %T", dbCtx, tx)
	}

	var (
		tab = args
	)

	b := tx.Bucket(tab)
	if b == nil {
		return nil, dbApi.ErrNoTable(tab)
	}

	err := b.Put([]byte("keyup"), []byte("valup"))
	if err != nil {
		return nil, err
	}

	return nil, nil
}
