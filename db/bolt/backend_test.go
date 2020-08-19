package bolt_test

import (
	"io/ioutil"
	"os"
	"testing"

	dbApi "github.com/tdx/rkv/db/api"
	"github.com/tdx/rkv/db/bolt"

	"github.com/stretchr/testify/require"
)

func TestBackend(t *testing.T) {
	defer os.RemoveAll("/tmp/rkv")

	db, err := bolt.New("/tmp/rkv")
	if err != nil {
		t.Fatal(err)
	}

	var (
		tab = []byte{'t', 'a', 'b'}
		key = []byte{'k', 'e', 'y'}
		val = []byte{'v', 'a', 'l'}
	)

	require.NoError(t, db.Put(tab, key, val))

	v, err := db.Get(tab, key)
	require.NoError(t, err)
	require.Equal(t, val, v)

	//
	// Put
	//

	// table is nil
	require.Equal(t, dbApi.TableNilError, db.Put(nil, nil, val))

	// key is nil
	require.Equal(t, dbApi.KeyNilError, db.Put(tab, nil, val))

	//
	// Get
	//

	// table is nil
	_, err = db.Get(nil, nil)
	require.Equal(t, dbApi.TableNilError, err)

	// no table
	_, err = db.Get([]byte{0}, key)
	require.Equal(t, true, dbApi.IsNoTableError(err))

	// key is nil
	_, err = db.Get(tab, nil)
	require.Equal(t, dbApi.KeyNilError, err)

	// no key
	_, err = db.Get(tab, []byte{0})
	require.Equal(t, true, dbApi.IsNoKeyError(err))

	//
	// Del
	//
	// table is nil
	err = db.Delete(nil, nil)
	require.Equal(t, dbApi.TableNilError, err)

	// no table
	err = db.Delete([]byte{0}, key)
	require.Equal(t, true, dbApi.IsNoTableError(err))

	// key is nil
	err = db.Delete(tab, nil)
	require.Equal(t, dbApi.KeyNilError, err)

	// ok
	err = db.Delete(tab, key)
	require.NoError(t, err)

	// get returns NoKeyError now
	_, err = db.Get(tab, key)
	require.Equal(t, true, dbApi.IsNoKeyError(err))
}

func TestBackupRestore(t *testing.T) {
	defer os.RemoveAll("/tmp/rkv")

	db, err := bolt.New("/tmp/rkv")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	t.Log("db file:", db.DSN())

	var (
		tab = []byte{'t', 'a', 'b'}
		key = []byte{'k', 'e', 'y'}
		val = []byte{'v', 'a', 'l'}
	)

	require.NoError(t, db.Put(tab, key, val))

	v, err := db.Get(tab, key)
	require.Equal(t, val, v)

	tmpFile, err := ioutil.TempFile("/tmp", "rkv-bk-*")
	defer os.Remove(tmpFile.Name())

	t.Log("backup db to:", tmpFile.Name())

	require.NoError(t, db.Backup(tmpFile))
	require.NoError(t, tmpFile.Close())

	snapFile, err := os.Open(tmpFile.Name())

	t.Log("restore db from:", snapFile.Name())
	require.NoError(t, db.Restore(snapFile))

	t.Log("db file:", db.DSN())

	// ensure old value exists
	v, err = db.Get(tab, key)
	require.NoError(t, err)
	require.Equal(t, val, v)
}

func TestBatch(t *testing.T) {
	defer os.RemoveAll("/tmp/rkv")

	db, err := bolt.New("/tmp/rkv")
	if err != nil {
		t.Fatal(err)
	}

	var (
		tab  = []byte{'t', 'a', 'b'}
		key  = []byte{'k', 'e', 'y'}
		val1 = []byte{'v', 'a', 'l', '1'}
		val2 = []byte{'v', 'a', 'l', '1'}
	)

	be := []*dbApi.BatchEntry{
		{Operation: dbApi.PutOperation,
			Entry: &dbApi.Entry{Tab: tab, Key: key, Val: val1}},
		{Operation: dbApi.PutOperation,
			Entry: &dbApi.Entry{Tab: tab, Key: key, Val: val2}},
	}

	err = db.Batch([][]*dbApi.BatchEntry{be}, false)
	require.NoError(t, err)

	v, err := db.Get(tab, key)
	require.NoError(t, err)
	require.Equal(t, val2, v)

	// 2 get in batch
	bg := []*dbApi.BatchEntry{
		{Operation: dbApi.GetOperation,
			Entry: &dbApi.Entry{Tab: tab, Key: key}},
		{Operation: dbApi.GetOperation,
			Entry: &dbApi.Entry{Tab: tab, Key: []byte("invalid")}},
	}

	err = db.Batch([][]*dbApi.BatchEntry{bg}, true)
	require.NoError(t, err)
	require.Equal(t, val2, bg[0].Result)

	err, ok := bg[1].Result.(error)
	require.True(t, ok)
	require.Equal(t, true, dbApi.IsNoKeyError(err))

}
