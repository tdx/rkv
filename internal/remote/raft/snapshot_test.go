package raft

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"hash/crc64"
	"io"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/require"
	dbApi "github.com/tdx/rkv/db/api"
	"github.com/tdx/rkv/db/bolt"
	"github.com/tdx/rkv/db/gmap"
	"github.com/travisjeffery/go-dynaport"
)

func TestRaftSnapshotBolt(t *testing.T) {
	runSnap(t, "bolt")
}

func TestRaftSnapshotMap(t *testing.T) {
	// failed now
	runSnap(t, "map")
}

func runSnap(t *testing.T, bkTyp string) {
	raft1, dir1 := getRaft(t, "1", true, bkTyp)
	raft2, dir2 := getRaft(t, "2", false, bkTyp)
	// raft3, dir3 := getRaft(t, "3", false)
	defer os.RemoveAll(dir1)
	defer os.RemoveAll(dir2)
	// defer os.RemoveAll(dir3)

	// Write some data
	tab := []byte{'t', 'a', 'b'}

	for i := 0; i < 1000; i++ {
		err := raft1.Put(
			tab,
			[]byte(fmt.Sprintf("key-%d", i)),
			[]byte(fmt.Sprintf("value-%d", i)),
		)
		require.NoError(t, err)
	}

	dbHash1 := mkSnapshotHash(t, raft1)

	// Add raft2 to the cluster
	addPeer(t, raft1, raft2)

	commitIdx := raft1.CommittedIndex()

	ensureCommitApplied(t, commitIdx, raft2)

	dbHash2 := mkDbBackupHash(t, raft2)

	require.True(t, bytes.Equal(dbHash1, dbHash2))
}

func getRaft(
	t testing.TB,
	id string,
	bootstrap bool,
	bkTyp string) (*Backend, string) {

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
	bkTyp string) (*Backend, string) {

	ports := dynaport.Get(1)

	ln, err := net.Listen(
		"tcp",
		fmt.Sprintf("127.0.0.1:%d", ports[0]),
	)
	require.NoError(t, err)

	config := &Config{}
	config.Bootstrap = bootstrap
	config.StreamLayer = NewStreamLayer(ln)
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

	r, err := New(db, config)
	require.NoError(t, err)

	if bootstrap {
		r.WaitForLeader(3 * time.Second)
	}

	return r, raftDir
}

func addPeer(t *testing.T, leader, follower *Backend) {
	t.Helper()
	err := leader.Join(
		string(follower.config.Raft.LocalID),
		follower.Addr().String())
	require.NoError(t, err)
}

func ensureCommitApplied(
	t *testing.T,
	leaderCommitIdx uint64,
	backend *Backend) {

	t.Helper()

	timeout := time.Now().Add(10 * time.Second)
	for {
		if time.Now().After(timeout) {
			t.Fatal("timeout reached while verifying applied index on raft backend")
		}

		if backend.AppliedIndex() >= leaderCommitIdx {
			break
		}

		time.Sleep(1 * time.Second)
	}
}

func mkSnapshotHash(t *testing.T, raft *Backend) []byte {

	snapFuture := raft.raft.Snapshot()
	require.NoError(t, snapFuture.Error())

	meta, reader, err := snapFuture.Open()
	require.NoError(t, err)

	t.Log("meta:", meta)

	// Create a CRC64 hash
	stateHash := crc64.New(crc64.MakeTable(crc64.ECMA))

	// Compute the hash
	n, err := io.Copy(stateHash, reader)
	require.NoError(t, err)
	require.True(t, n > 0)

	hash := stateHash.Sum(nil)

	raft.logger.Info("snapshot", "bytes", n, "hash", hex.EncodeToString(hash))

	return hash
}

func mkDbBackupHash(t *testing.T, raft *Backend) []byte {
	tmpFile, err := ioutil.TempFile("/tmp", "rkv-bk-*")
	// defer os.Remove(tmpFile.Name())

	require.NoError(t, err)

	require.NoError(t, raft.fsm.db.Backup(tmpFile))
	require.NoError(t, tmpFile.Close())

	snapFile, err := os.Open(tmpFile.Name())
	require.NoError(t, err)

	// Create a CRC64 hash
	stateHash := crc64.New(crc64.MakeTable(crc64.ECMA))

	// Compute the hash
	n, err := io.Copy(stateHash, snapFile)
	require.NoError(t, err)
	require.True(t, n > 0)

	hash := stateHash.Sum(nil)

	raft.logger.Info("backup", "bytes", n, "hash", hex.EncodeToString(hash))

	return hash
}
