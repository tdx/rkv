package raft

import (
	rkvApi "github.com/tdx/rkv/api"
	dbApi "github.com/tdx/rkv/db/api"
	rpcRaft "github.com/tdx/rkv/internal/rpc/raft"
)

const (
	putOp uint32 = 1 << iota
	deleteOp
	getOp
)

//
// Get returns value from local backend
//
func (d *Backend) Get(
	lvl rkvApi.ConsistencyLevel, tab, key []byte) ([]byte, error) {

	switch lvl {
	case rkvApi.ReadAny:
		return d.fsm.Get(tab, key)

	case rkvApi.ReadLeader:
		if d.IsLeader() {
			return d.fsm.Get(tab, key)
		}
		// TODO: rpc call to leader
		return nil, rkvApi.ErrNodeIsNotALeader

	case rkvApi.ReadRaft:
		if !d.IsLeader() {
			return nil, rkvApi.ErrNodeIsNotALeader
		}
	}

	// raft read from leader
	req := &rpcRaft.LogData{
		Operations: []*rpcRaft.LogOperation{
			{
				OpType: getOp,
				Tab:    tab,
				Key:    key,
			},
		},
	}

	res, err := d.applyLog(req)
	if err != nil {
		return nil, err
	}

	return res.([]byte), nil

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

// Batch apply multiple put, delete operations
func (d *Backend) Batch(commands []*dbApi.BatchEntry) error {
	ops := make([]*rpcRaft.LogOperation, 0, len(commands))
	for _, cmd := range commands {
		switch cmd.Operation {
		case dbApi.PutOperation:
			ops = append(ops, &rpcRaft.LogOperation{
				OpType: putOp,
				Tab:    cmd.Entry.Tab,
				Key:    cmd.Entry.Key,
				Val:    cmd.Entry.Val,
			})
		case dbApi.DeleteOperation:
			ops = append(ops, &rpcRaft.LogOperation{
				OpType: deleteOp,
				Tab:    cmd.Entry.Tab,
				Key:    cmd.Entry.Key,
			})
		}
	}

	req := &rpcRaft.LogData{
		Operations: ops,
	}

	_, err := d.applyLog(req)

	return err
}
