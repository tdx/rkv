package raft

import (
	"bytes"
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
	"github.com/travisjeffery/go-dynaport"
)

func TestRaftSnapshot(t *testing.T) {
	raft1, dir1 := getRaft(t, "1", true)
	raft2, dir2 := getRaft(t, "2", false)
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

	dbHash1 := mkDbHash(t, raft1)

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
	bootstrap bool) (*Backend, string) {

	raftDir, err := ioutil.TempDir("", "rkv-raft-")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("raft dir:", raftDir)

	return getRaftWithDir(t, id, bootstrap, raftDir)
}

func getRaftWithDir(
	t testing.TB,
	id string,
	bootstrap bool,
	raftDir string) (*Backend, string) {

	ports := dynaport.Get(1)

	ln, err := net.Listen(
		"tcp",
		fmt.Sprintf("127.0.0.1:%d", ports[0]),
	)
	require.NoError(t, err)

	config := Config{}
	config.Raft.StreamLayer = NewStreamLayer(ln)
	config.Raft.LocalID = raft.ServerID(id)
	config.Raft.HeartbeatTimeout = 50 * time.Millisecond
	config.Raft.ElectionTimeout = 50 * time.Millisecond
	config.Raft.LeaderLeaseTimeout = 50 * time.Millisecond
	config.Raft.CommitTimeout = 5 * time.Millisecond
	config.Raft.Bootstrap = bootstrap

	r, err := NewBackend(raftDir, config)
	require.NoError(t, err)

	r.WaitForLeader(1 * time.Second)

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

func mkDbHash(t *testing.T, raft *Backend) []byte {

	snapFuture := raft.raft.Snapshot()
	require.NoError(t, snapFuture.Error())

	meta, reader, err := snapFuture.Open()
	require.NoError(t, err)

	t.Log("meta:", meta)

	// Create a CRC64 hash
	stateHash := crc64.New(crc64.MakeTable(crc64.ECMA))

	// Compute the hash
	_, err = io.Copy(stateHash, reader)
	require.NoError(t, err)

	return stateHash.Sum(nil)
}

func mkDbBackupHash(t *testing.T, raft *Backend) []byte {
	tmpFile, err := ioutil.TempFile("/tmp", "rkv-bk-*")
	defer os.Remove(tmpFile.Name())

	require.NoError(t, err)

	require.NoError(t, raft.fsm.db.Backup(tmpFile))
	require.NoError(t, tmpFile.Close())

	snapFile, err := os.Open(tmpFile.Name())
	require.NoError(t, err)

	// Create a CRC64 hash
	stateHash := crc64.New(crc64.MakeTable(crc64.ECMA))

	// Compute the hash
	_, err = io.Copy(stateHash, snapFile)
	require.NoError(t, err)

	return stateHash.Sum(nil)
}
