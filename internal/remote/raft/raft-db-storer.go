package raft

import (
	rpcRaft "rkv/internal/rpc/raft"
)

const (
	putOp uint32 = 1 << iota
	deleteOp
)

//
// Get returns value from local backend
//
func (d *Backend) Get(tab, key []byte) ([]byte, error) {
	return d.fsm.Get(tab, key)
}

// Put apply value via raft
func (d *Backend) Put(tab, key, val []byte) error {
	req := &rpcRaft.LogData{
		Operations: []*rpcRaft.LogOperation{
			{
				OpType: putOp,
				Tab:    tab,
				Key:    key,
				Val:    val,
			},
		},
	}

	_, err := d.applyLog(req)

	return err
}

// Delete inserts an entry in the log to delete the given key
func (d *Backend) Delete(tab, key []byte) error {
	req := &rpcRaft.LogData{
		Operations: []*rpcRaft.LogOperation{
			{
				OpType: deleteOp,
				Tab:    tab,
				Key:    key,
			},
		},
	}

	_, err := d.applyLog(req)

	return err
}
