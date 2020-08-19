package gmap_test

import (
	"fmt"
	"testing"

	dbApi "github.com/tdx/rkv/db/api"
	"github.com/tdx/rkv/db/gmap"

	"github.com/stretchr/testify/require"
)

func TestApply(t *testing.T) {

	db, err := gmap.New("/tmp/rkv2")
	require.NoError(t, err)

	dbFn, ok := db.(dbApi.Applier)
	require.True(t, ok)

	count, err := dbFn.Apply(fnCount, nil, true)
	require.NoError(t, err)

	n, ok := count.(int)
	require.True(t, ok)
	require.Equal(t, 0, n)

	var (
		tab = []byte{'t', 'a', 'b'}
		key = []byte{'k', 'e', 'y'}
		val = []byte{'v', 'a', 'l'}
	)

	err = db.Put(tab, key, val)
	require.NoError(t, err)

	count, err = dbFn.Apply(fnCount, nil, true)
	require.NoError(t, err)

	n, ok = count.(int)
	require.True(t, ok)
	require.Equal(t, 1, n)
}

func fnCount(dbCtx interface{}, args []byte) (interface{}, error) {
	m, ok := dbCtx.(map[string]map[string][]byte)
	if !ok {
		return nil, fmt.Errorf("invalid dbCtx: %T %T", dbCtx, m)
	}
	return len(m), nil
}
